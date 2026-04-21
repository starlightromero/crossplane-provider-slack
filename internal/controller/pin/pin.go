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

// Package pin implements the ConversationPin managed resource controller.
package pin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	xperrors "github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pinv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/pin/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
	"github.com/avodah-inc/crossplane-provider-slack/internal/controller/providerconfig"
)

// Condition reasons for ConversationPin status.
const (
	ReasonChannelUnavailable xpv1.ConditionReason = "ChannelUnavailable"
	ReasonMessageNotFound    xpv1.ConditionReason = "MessageNotFound"
)

const (
	errNotPin        = "managed resource is not a ConversationPin"
	errTrackUsage    = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errExtractToken  = "cannot extract bot token from secret"
	errValidateToken = "invalid bot token"
	errResolveConv   = "cannot resolve conversation ID"
	errObserve       = "cannot list pins"
	errCreate        = "cannot add pin"
	errDelete        = "cannot remove pin"
)

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
	newFn func(token string, opts ...slack.ClientOption) slack.ClientAPI
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*pinv1alpha1.ConversationPin)
	if !ok {
		return nil, xperrors.New(errNotPin)
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
	cr, ok := mg.(*pinv1alpha1.ConversationPin)
	if !ok {
		return managed.ExternalObservation{}, xperrors.New(errNotPin)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	channelID, messageTS := parseExternalName(externalName)
	if channelID == "" || messageTS == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	pins, err := e.client.ListPins(ctx, channelID)
	if err != nil {
		if isChannelUnavailable(err) {
			setChannelUnavailableCondition(cr, err)
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, xperrors.Wrap(err, errObserve)
	}

	var found *slack.Pin
	for i := range pins {
		if pins[i].Message.Ts == messageTS {
			found = &pins[i]
			break
		}
	}

	if found == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider = pinv1alpha1.ConversationPinObservation{
		ChannelID: channelID,
		PinnedAt:  found.Created,
	}

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true, // Pins are immutable - no update needed
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*pinv1alpha1.ConversationPin)
	if !ok {
		return managed.ExternalCreation{}, xperrors.New(errNotPin)
	}

	channelID, err := pinv1alpha1.ResolveConversationID(ctx, e.kube, &cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalCreation{}, xperrors.Wrap(err, errResolveConv)
	}

	messageTS := cr.Spec.ForProvider.MessageTimestamp

	err = e.client.AddPin(ctx, channelID, messageTS)
	if err != nil {
		if isChannelUnavailable(err) {
			setChannelUnavailableCondition(cr, err)
		}
		if isMessageNotFound(err) {
			setMessageNotFoundCondition(cr, err)
		}
		return managed.ExternalCreation{}, xperrors.Wrap(err, errCreate)
	}

	meta.SetExternalName(cr, formatExternalName(channelID, messageTS))
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(_ context.Context, _ resource.Managed) (managed.ExternalUpdate, error) {
	// Pins are immutable - no update operation exists.
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*pinv1alpha1.ConversationPin)
	if !ok {
		return managed.ExternalDelete{}, xperrors.New(errNotPin)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	channelID, messageTS := parseExternalName(externalName)
	if channelID == "" || messageTS == "" {
		return managed.ExternalDelete{}, nil
	}

	err := e.client.RemovePin(ctx, channelID, messageTS)
	if err != nil {
		if isChannelUnavailable(err) {
			return managed.ExternalDelete{}, nil
		}
		var slackErr *slack.SlackError
		if errors.As(err, &slackErr) && slackErr.Code == "no_pin" {
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

// formatExternalName creates the composite external name for a pin.
func formatExternalName(channelID, messageTS string) string {
	return fmt.Sprintf("%s:%s", channelID, messageTS)
}

// parseExternalName splits the composite external name into channel ID and message timestamp.
func parseExternalName(externalName string) (channelID, messageTS string) {
	parts := strings.SplitN(externalName, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func isChannelUnavailable(err error) bool {
	var slackErr *slack.SlackError
	if errors.As(err, &slackErr) {
		return slackErr.Code == "channel_not_found" || slackErr.Code == "is_archived"
	}
	return false
}

func isMessageNotFound(err error) bool {
	var slackErr *slack.SlackError
	if errors.As(err, &slackErr) {
		return slackErr.Code == "message_not_found"
	}
	return false
}

func setChannelUnavailableCondition(cr *pinv1alpha1.ConversationPin, err error) {
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

func setMessageNotFoundCondition(cr *pinv1alpha1.ConversationPin, err error) {
	var slackErr *slack.SlackError
	if errors.As(err, &slackErr) {
		cr.SetConditions(xpv1.Condition{
			Type:    xpv1.TypeSynced,
			Status:  "False",
			Reason:  ReasonMessageNotFound,
			Message: slackErr.Message,
		})
	}
}
