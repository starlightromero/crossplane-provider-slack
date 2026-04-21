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

// Package conversation implements the Conversation managed resource controller.
package conversation

import (
	"context"
	"errors"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xperrors "github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	conversationv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/conversation/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
	"github.com/avodah-inc/crossplane-provider-slack/internal/controller/providerconfig"
)

// Condition reasons for Conversation status.
const (
	ReasonNameConflict xpv1.ConditionReason = "NameConflict"
	ReasonNotFound     xpv1.ConditionReason = "NotFound"
)

const (
	errNotConversation = "managed resource is not a Conversation"
	errTrackUsage      = "cannot track ProviderConfig usage"
	errGetPC           = "cannot get ProviderConfig"
	errExtractToken    = "cannot extract bot token from secret"
	errValidateToken   = "invalid bot token"
	errObserve         = "cannot get conversation info"
	errCreate          = "cannot create conversation"
	errDelete          = "cannot archive conversation"
	errRename          = "cannot rename conversation"
	errSetTopic        = "cannot set conversation topic"
	errSetPurpose      = "cannot set conversation purpose"
	errUpdate          = "cannot update conversation"
)

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage *resource.LegacyProviderConfigUsageTracker
	newFn func(token string, opts ...slack.ClientOption) slack.ClientAPI
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*conversationv1alpha1.Conversation)
	if !ok {
		return nil, xperrors.New(errNotConversation)
	}

	if err := c.usage.Track(ctx, cr); err != nil {
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

	return &external{client: c.newFn(token)}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client slack.ClientAPI
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*conversationv1alpha1.Conversation)
	if !ok {
		return managed.ExternalObservation{}, xperrors.New(errNotConversation)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	conv, err := e.client.GetConversationInfo(ctx, externalName)
	if err != nil {
		var slackErr *slack.SlackError
		if errors.As(err, &slackErr) && slackErr.Code == "channel_not_found" {
			cr.SetConditions(xpv1.Condition{
				Type:    xpv1.TypeSynced,
				Status:  "False",
				Reason:  ReasonNotFound,
				Message: "channel not found in Slack",
			})
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, xperrors.Wrap(err, errObserve)
	}

	if conv.IsArchived {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider = conversationv1alpha1.ConversationObservation{
		ID:         conv.ID,
		IsArchived: conv.IsArchived,
		NumMembers: conv.NumMembers,
		Created:    conv.Created,
	}

	upToDate := isUpToDate(cr.Spec.ForProvider, conv)
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*conversationv1alpha1.Conversation)
	if !ok {
		return managed.ExternalCreation{}, xperrors.New(errNotConversation)
	}

	isPrivate := false
	if cr.Spec.ForProvider.IsPrivate != nil {
		isPrivate = *cr.Spec.ForProvider.IsPrivate
	}

	conv, err := e.client.CreateConversation(ctx, cr.Spec.ForProvider.Name, isPrivate)
	if err != nil {
		var slackErr *slack.SlackError
		if errors.As(err, &slackErr) && slackErr.Code == "name_taken" {
			cr.SetConditions(xpv1.Condition{
				Type:    xpv1.TypeSynced,
				Status:  "False",
				Reason:  ReasonNameConflict,
				Message: "channel name is already taken",
			})
		}
		return managed.ExternalCreation{}, xperrors.Wrap(err, errCreate)
	}

	meta.SetExternalName(cr, conv.ID)

	if cr.Spec.ForProvider.Topic != nil && *cr.Spec.ForProvider.Topic != "" {
		if err := e.client.SetConversationTopic(ctx, conv.ID, *cr.Spec.ForProvider.Topic); err != nil {
			return managed.ExternalCreation{}, xperrors.Wrap(err, errSetTopic)
		}
	}
	if cr.Spec.ForProvider.Purpose != nil && *cr.Spec.ForProvider.Purpose != "" {
		if err := e.client.SetConversationPurpose(ctx, conv.ID, *cr.Spec.ForProvider.Purpose); err != nil {
			return managed.ExternalCreation{}, xperrors.Wrap(err, errSetPurpose)
		}
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*conversationv1alpha1.Conversation)
	if !ok {
		return managed.ExternalUpdate{}, xperrors.New(errNotConversation)
	}

	externalName := meta.GetExternalName(cr)

	conv, err := e.client.GetConversationInfo(ctx, externalName)
	if err != nil {
		return managed.ExternalUpdate{}, xperrors.Wrap(err, errUpdate)
	}

	if cr.Spec.ForProvider.Name != conv.Name {
		if err := e.client.RenameConversation(ctx, externalName, cr.Spec.ForProvider.Name); err != nil {
			setNameConflictCondition(cr, err)
			return managed.ExternalUpdate{}, xperrors.Wrap(err, errRename)
		}
	}

	desiredTopic := ptrValueOrEmpty(cr.Spec.ForProvider.Topic)
	if desiredTopic != conv.Topic.Value {
		if err := e.client.SetConversationTopic(ctx, externalName, desiredTopic); err != nil {
			return managed.ExternalUpdate{}, xperrors.Wrap(err, errSetTopic)
		}
	}

	desiredPurpose := ptrValueOrEmpty(cr.Spec.ForProvider.Purpose)
	if desiredPurpose != conv.Purpose.Value {
		if err := e.client.SetConversationPurpose(ctx, externalName, desiredPurpose); err != nil {
			return managed.ExternalUpdate{}, xperrors.Wrap(err, errSetPurpose)
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*conversationv1alpha1.Conversation)
	if !ok {
		return managed.ExternalDelete{}, xperrors.New(errNotConversation)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	err := e.client.ArchiveConversation(ctx, externalName)
	if err != nil {
		var slackErr *slack.SlackError
		if errors.As(err, &slackErr) && slackErr.Code == "channel_not_found" {
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

func isUpToDate(desired conversationv1alpha1.ConversationParameters, observed *slack.Conversation) bool {
	if desired.Name != observed.Name {
		return false
	}
	if ptrValueOrEmpty(desired.Topic) != observed.Topic.Value {
		return false
	}
	if ptrValueOrEmpty(desired.Purpose) != observed.Purpose.Value {
		return false
	}
	return true
}

// setNameConflictCondition sets a Synced=False condition if the error is a
// name_taken Slack error.
func setNameConflictCondition(cr *conversationv1alpha1.Conversation, err error) {
	var slackErr *slack.SlackError
	if errors.As(err, &slackErr) && slackErr.Code == "name_taken" {
		cr.SetConditions(xpv1.Condition{
			Type:    xpv1.TypeSynced,
			Status:  "False",
			Reason:  ReasonNameConflict,
			Message: "channel name is already taken",
		})
	}
}

// ptrValueOrEmpty returns the dereferenced string pointer value, or empty
// string if the pointer is nil.
func ptrValueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
