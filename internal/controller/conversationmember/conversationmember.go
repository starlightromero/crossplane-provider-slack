/*
Copyright 2026 Starlight Romero.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// Package conversationmember implements the ConversationMember managed resource controller.
package conversationmember

import (
	"context"
	"errors"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xperrors "github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	conversationmemberv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/conversationmember/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
	"github.com/avodah-inc/crossplane-provider-slack/internal/controller/providerconfig"
)

const (
	errNotConversationMember = "managed resource is not a ConversationMember"
	errTrackUsage            = "cannot track ProviderConfig usage"
	errGetPC                 = "cannot get ProviderConfig"
	errExtractToken          = "cannot extract bot token from secret"
	errValidateToken         = "invalid bot token"
	errResolveConv           = "cannot resolve conversation ID"
	errResolveUser           = "cannot resolve user"
	errInvite                = "cannot invite user to conversation"
	errKick                  = "cannot remove user from conversation"
	errListMembers           = "cannot list conversation members"
)

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage *resource.LegacyProviderConfigUsageTracker
	newFn func(token string, opts ...slack.ClientOption) slack.ClientAPI
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*conversationmemberv1alpha1.ConversationMember)
	if !ok {
		return nil, xperrors.New(errNotConversationMember)
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

	return &external{client: c.newFn(token), kube: c.kube}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client slack.ClientAPI
	kube   client.Reader
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*conversationmemberv1alpha1.ConversationMember)
	if !ok {
		return managed.ExternalObservation{}, xperrors.New(errNotConversationMember)
	}

	// external-name format: "<channelID>/<userID>"
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	channelID, userID, ok := parseExternalName(externalName)
	if !ok {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	members, err := e.client.GetConversationMembers(ctx, channelID)
	if err != nil {
		return managed.ExternalObservation{}, xperrors.Wrap(err, errListMembers)
	}

	for _, m := range members {
		if m == userID {
			cr.Status.AtProvider = conversationmemberv1alpha1.ConversationMemberObservation{
				ResolvedUserID: userID,
				ChannelID:      channelID,
			}
			cr.SetConditions(xpv1.Available())
			return managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: true,
			}, nil
		}
	}

	// User is not in the channel — resource doesn't exist
	return managed.ExternalObservation{ResourceExists: false}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*conversationmemberv1alpha1.ConversationMember)
	if !ok {
		return managed.ExternalCreation{}, xperrors.New(errNotConversationMember)
	}

	channelID, err := conversationmemberv1alpha1.ResolveConversationID(ctx, e.kube, &cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalCreation{}, xperrors.Wrap(err, errResolveConv)
	}

	userID, err := e.resolveUserID(ctx, cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalCreation{}, xperrors.Wrap(err, errResolveUser)
	}

	// Ensure the bot is in the channel before inviting
	_ = e.client.JoinConversation(ctx, channelID)

	err = e.client.InviteToConversation(ctx, channelID, userID)
	if err != nil {
		// "already_in_channel" is not an error per the requirements
		var slackErr *slack.SlackError
		if errors.As(err, &slackErr) && slackErr.Code == "already_in_channel" {
			// Already a member — set external name and succeed
		} else {
			return managed.ExternalCreation{}, xperrors.Wrap(err, errInvite)
		}
	}

	meta.SetExternalName(cr, fmt.Sprintf("%s/%s", channelID, userID))
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(_ context.Context, _ resource.Managed) (managed.ExternalUpdate, error) {
	// Membership is binary — no update needed
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*conversationmemberv1alpha1.ConversationMember)
	if !ok {
		return managed.ExternalDelete{}, xperrors.New(errNotConversationMember)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	channelID, userID, ok := parseExternalName(externalName)
	if !ok {
		return managed.ExternalDelete{}, nil
	}

	err := e.client.KickFromConversation(ctx, channelID, userID)
	if err != nil {
		var slackErr *slack.SlackError
		if errors.As(err, &slackErr) && (slackErr.Code == "not_in_channel" || slackErr.Code == "channel_not_found") {
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, xperrors.Wrap(err, errKick)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op for the Slack client.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// resolveUserID resolves the user ID from the spec fields.
func (e *external) resolveUserID(ctx context.Context, params conversationmemberv1alpha1.ConversationMemberParameters) (string, error) {
	if params.UserID != nil && *params.UserID != "" {
		return *params.UserID, nil
	}

	if params.UserEmail != nil && *params.UserEmail != "" {
		user, err := e.client.LookupUserByEmail(ctx, *params.UserEmail)
		if err != nil {
			return "", err
		}
		if user == nil || user.ID == "" {
			return "", fmt.Errorf("user not found for email %s", *params.UserEmail)
		}
		return user.ID, nil
	}

	return "", fmt.Errorf("one of userEmail or userId is required")
}

// parseExternalName splits "channelID/userID" into its components.
func parseExternalName(name string) (channelID, userID string, ok bool) {
	for i := range name {
		if name[i] == '/' {
			return name[:i], name[i+1:], true
		}
	}
	return "", "", false
}
