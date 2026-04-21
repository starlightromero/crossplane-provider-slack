/*
Copyright 2024 Avodah Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package bookmark implements the ConversationBookmark managed resource controller.
package bookmark

import (
	"context"
	"errors"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	xperrors "github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	bookmarkv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/bookmark/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
	"github.com/avodah-inc/crossplane-provider-slack/internal/controller/providerconfig"
)

// Condition reasons for ConversationBookmark status.
const (
	ReasonChannelUnavailable xpv1.ConditionReason = "ChannelUnavailable"
)

const (
	errNotBookmark   = "managed resource is not a ConversationBookmark"
	errTrackUsage    = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errExtractToken  = "cannot extract bot token from secret"
	errValidateToken = "invalid bot token"
	errResolveConv   = "cannot resolve conversation ID"
	errObserve       = "cannot list bookmarks"
	errCreate        = "cannot add bookmark"
	errUpdate        = "cannot edit bookmark"
	errDelete        = "cannot remove bookmark"
)

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
	newFn func(token string, opts ...slack.ClientOption) slack.ClientAPI
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*bookmarkv1alpha1.ConversationBookmark)
	if !ok {
		return nil, xperrors.New(errNotBookmark)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, xperrors.Wrap(err, errTrackUsage)
	}

	pc := &v1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{
		Name: cr.GetProviderConfigReference().Name,
	}, pc); err != nil {
		return nil, xperrors.Wrap(err, errGetPC)
	}

	token, err := providerconfig.ExtractToken(ctx, c.kube, pc)
	if err != nil {
		return nil, xperrors.Wrap(err, errExtractToken)
	}

	if err := providerconfig.ValidateToken(token); err != nil {
		return nil, xperrors.Wrap(err, errValidateToken)
	}

	return &external{client: c.newFn(token), kube: c.kube}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client slack.ClientAPI
	kube   client.Reader
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*bookmarkv1alpha1.ConversationBookmark)
	if !ok {
		return managed.ExternalObservation{}, xperrors.New(errNotBookmark)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	channelID, err := bookmarkv1alpha1.ResolveConversationID(ctx, e.kube, &cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalObservation{}, xperrors.Wrap(err, errResolveConv)
	}
	if channelID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	bookmarks, err := e.client.ListBookmarks(ctx, channelID)
	if err != nil {
		if isChannelUnavailable(err) {
			setChannelUnavailableCondition(cr, err)
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, xperrors.Wrap(err, errObserve)
	}

	var found *slack.Bookmark
	for i := range bookmarks {
		if bookmarks[i].ID == externalName {
			found = &bookmarks[i]
			break
		}
	}

	if found == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider = bookmarkv1alpha1.ConversationBookmarkObservation{
		ID:          found.ID,
		ChannelID:   found.ChannelID,
		DateCreated: found.DateCreated,
	}

	upToDate := isUpToDate(cr.Spec.ForProvider, found)
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*bookmarkv1alpha1.ConversationBookmark)
	if !ok {
		return managed.ExternalCreation{}, xperrors.New(errNotBookmark)
	}

	channelID, err := bookmarkv1alpha1.ResolveConversationID(ctx, e.kube, &cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalCreation{}, xperrors.Wrap(err, errResolveConv)
	}

	bm, err := e.client.AddBookmark(ctx, channelID, slack.BookmarkParams{
		Title: cr.Spec.ForProvider.Title,
		Type:  cr.Spec.ForProvider.Type,
		Link:  cr.Spec.ForProvider.Link,
	})
	if err != nil {
		if isChannelUnavailable(err) {
			setChannelUnavailableCondition(cr, err)
		}
		return managed.ExternalCreation{}, xperrors.Wrap(err, errCreate)
	}

	meta.SetExternalName(cr, bm.ID)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*bookmarkv1alpha1.ConversationBookmark)
	if !ok {
		return managed.ExternalUpdate{}, xperrors.New(errNotBookmark)
	}

	externalName := meta.GetExternalName(cr)

	channelID, err := bookmarkv1alpha1.ResolveConversationID(ctx, e.kube, &cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalUpdate{}, xperrors.Wrap(err, errResolveConv)
	}

	err = e.client.EditBookmark(ctx, channelID, externalName, slack.BookmarkParams{
		Title: cr.Spec.ForProvider.Title,
		Link:  cr.Spec.ForProvider.Link,
	})
	if err != nil {
		if isChannelUnavailable(err) {
			setChannelUnavailableCondition(cr, err)
		}
		return managed.ExternalUpdate{}, xperrors.Wrap(err, errUpdate)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*bookmarkv1alpha1.ConversationBookmark)
	if !ok {
		return managed.ExternalDelete{}, xperrors.New(errNotBookmark)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	channelID, err := bookmarkv1alpha1.ResolveConversationID(ctx, e.kube, &cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalDelete{}, xperrors.Wrap(err, errResolveConv)
	}

	err = e.client.RemoveBookmark(ctx, channelID, externalName)
	if err != nil {
		if isChannelUnavailable(err) {
			return managed.ExternalDelete{}, nil
		}
		var slackErr *slack.SlackError
		if errors.As(err, &slackErr) && slackErr.Code == "bookmark_not_found" {
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, xperrors.Wrap(err, errDelete)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op for the Slack client.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

func isUpToDate(desired bookmarkv1alpha1.ConversationBookmarkParameters, observed *slack.Bookmark) bool {
	if desired.Title != observed.Title {
		return false
	}
	if desired.Link != observed.Link {
		return false
	}
	return true
}

func isChannelUnavailable(err error) bool {
	var slackErr *slack.SlackError
	if errors.As(err, &slackErr) {
		return slackErr.Code == "channel_not_found" || slackErr.Code == "is_archived"
	}
	return false
}

func setChannelUnavailableCondition(cr *bookmarkv1alpha1.ConversationBookmark, err error) {
	var slackErr *slack.SlackError
	if errors.As(err, &slackErr) {
		cr.SetConditions(xpv1.Condition{
			Type:    xpv1.TypeSynced,
			Status:  "False",
			Reason:  ReasonChannelUnavailable,
			Message: slackErr.Message,
		})
	}
}
