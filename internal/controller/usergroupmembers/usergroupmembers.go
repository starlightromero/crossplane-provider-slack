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

// Package usergroupmembers implements the UserGroupMembers managed resource controller.
package usergroupmembers

import (
	"context"
	"fmt"
	"sort"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	xperrors "github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	usergroupmembersv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/usergroupmembers/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
	"github.com/avodah-inc/crossplane-provider-slack/internal/controller/providerconfig"
)

// Condition reasons for UserGroupMembers status.
const (
	ReasonUserNotFound xpv1.ConditionReason = "UserNotFound"
)

const (
	errNotUserGroupMembers = "managed resource is not a UserGroupMembers"
	errTrackUsage          = "cannot track ProviderConfig usage"
	errGetPC               = "cannot get ProviderConfig"
	errExtractToken        = "cannot extract bot token from secret"
	errValidateToken       = "invalid bot token"
	errListMembers         = "cannot list user group members"
	errResolveEmails       = "cannot resolve user emails"
	errUpdateMembers       = "cannot update user group members"
	errResolveGroupID      = "cannot resolve user group ID"
)

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
	newFn func(token string, opts ...slack.ClientOption) slack.ClientAPI
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*usergroupmembersv1alpha1.UserGroupMembers)
	if !ok {
		return nil, xperrors.New(errNotUserGroupMembers)
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
	kube   client.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*usergroupmembersv1alpha1.UserGroupMembers)
	if !ok {
		return managed.ExternalObservation{}, xperrors.New(errNotUserGroupMembers)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Get current members from Slack
	currentMembers, err := e.client.ListUserGroupMembers(ctx, externalName)
	if err != nil {
		return managed.ExternalObservation{}, xperrors.Wrap(err, errListMembers)
	}

	// Resolve desired emails to user IDs
	resolvedIDs, unresolvable, err := e.resolveEmails(ctx, cr.Spec.ForProvider.UserEmails)
	if err != nil {
		return managed.ExternalObservation{}, xperrors.Wrap(err, errResolveEmails)
	}

	// If there are unresolvable emails, set condition but still report observation
	if len(unresolvable) > 0 {
		cr.SetConditions(xpv1.Condition{
			Type:    xpv1.TypeSynced,
			Status:  "False",
			Reason:  ReasonUserNotFound,
			Message: fmt.Sprintf("cannot resolve email(s): %s", joinStrings(unresolvable)),
		})
	}

	// Update status observation
	cr.Status.AtProvider = usergroupmembersv1alpha1.UserGroupMembersObservation{
		ResolvedUserIds: currentMembers,
		MemberCount:     len(currentMembers),
	}

	// Compare sets (order-independent)
	upToDate := setsEqual(resolvedIDs, currentMembers)
	if upToDate && len(unresolvable) == 0 {
		cr.SetConditions(xpv1.Available())
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*usergroupmembersv1alpha1.UserGroupMembers)
	if !ok {
		return managed.ExternalCreation{}, xperrors.New(errNotUserGroupMembers)
	}

	groupID, err := usergroupmembersv1alpha1.ResolveUserGroupID(ctx, e.kube, &cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalCreation{}, xperrors.Wrap(err, errResolveGroupID)
	}

	// Set external-name to the usergroup ID
	meta.SetExternalName(cr, groupID)

	// Resolve emails and update members
	resolvedIDs, unresolvable, err := e.resolveEmails(ctx, cr.Spec.ForProvider.UserEmails)
	if err != nil {
		return managed.ExternalCreation{}, xperrors.Wrap(err, errResolveEmails)
	}

	if len(unresolvable) > 0 {
		cr.SetConditions(xpv1.Condition{
			Type:    xpv1.TypeSynced,
			Status:  "False",
			Reason:  ReasonUserNotFound,
			Message: fmt.Sprintf("cannot resolve email(s): %s", joinStrings(unresolvable)),
		})
	}

	if len(resolvedIDs) > 0 {
		if err := e.client.UpdateUserGroupMembers(ctx, groupID, resolvedIDs); err != nil {
			return managed.ExternalCreation{}, xperrors.Wrap(err, errUpdateMembers)
		}
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*usergroupmembersv1alpha1.UserGroupMembers)
	if !ok {
		return managed.ExternalUpdate{}, xperrors.New(errNotUserGroupMembers)
	}

	externalName := meta.GetExternalName(cr)

	// Resolve emails and update members (full replacement)
	resolvedIDs, unresolvable, err := e.resolveEmails(ctx, cr.Spec.ForProvider.UserEmails)
	if err != nil {
		return managed.ExternalUpdate{}, xperrors.Wrap(err, errResolveEmails)
	}

	if len(unresolvable) > 0 {
		cr.SetConditions(xpv1.Condition{
			Type:    xpv1.TypeSynced,
			Status:  "False",
			Reason:  ReasonUserNotFound,
			Message: fmt.Sprintf("cannot resolve email(s): %s", joinStrings(unresolvable)),
		})
	}

	if len(resolvedIDs) > 0 {
		if err := e.client.UpdateUserGroupMembers(ctx, externalName, resolvedIDs); err != nil {
			return managed.ExternalUpdate{}, xperrors.Wrap(err, errUpdateMembers)
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*usergroupmembersv1alpha1.UserGroupMembers)
	if !ok {
		return managed.ExternalDelete{}, xperrors.New(errNotUserGroupMembers)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	// Clear membership by setting empty user list
	if err := e.client.UpdateUserGroupMembers(ctx, externalName, []string{}); err != nil {
		return managed.ExternalDelete{}, xperrors.Wrap(err, errUpdateMembers)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op for the Slack client.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// resolveEmails resolves a list of email addresses to Slack user IDs.
// Returns the resolved IDs and any emails that could not be resolved.
func (e *external) resolveEmails(ctx context.Context, emails []string) ([]string, []string, error) {
	var resolved []string
	var unresolvable []string

	for _, email := range emails {
		user, err := e.client.LookupUserByEmail(ctx, email)
		if err != nil {
			// Check if it's a "users_not_found" error (user doesn't exist)
			var slackErr *slack.SlackError
			if isSlackError(err, &slackErr) && slackErr.Code == "users_not_found" {
				unresolvable = append(unresolvable, email)
				continue
			}
			// For other errors, return immediately
			return nil, nil, err
		}
		if user != nil && user.ID != "" {
			resolved = append(resolved, user.ID)
		} else {
			unresolvable = append(unresolvable, email)
		}
	}

	return resolved, unresolvable, nil
}

// isSlackError checks if an error is a SlackError and assigns it.
func isSlackError(err error, target **slack.SlackError) bool {
	if se, ok := err.(*slack.SlackError); ok {
		*target = se
		return true
	}
	return false
}

// setsEqual returns true if two string slices contain the same elements
// regardless of order.
func setsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA := make([]string, len(a))
	copy(sortedA, a)
	sort.Strings(sortedA)

	sortedB := make([]string, len(b))
	copy(sortedB, b)
	sort.Strings(sortedB)

	for i := range sortedA {
		if sortedA[i] != sortedB[i] {
			return false
		}
	}
	return true
}

// joinStrings joins strings with ", ".
func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
