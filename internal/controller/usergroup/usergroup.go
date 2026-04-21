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

// Package usergroup implements the UserGroup managed resource controller.
package usergroup

import (
	"context"
	"errors"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xperrors "github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	usergroupv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/usergroup/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
	"github.com/avodah-inc/crossplane-provider-slack/internal/controller/providerconfig"
)

// Condition reasons for UserGroup status.
const (
	ReasonNameConflict xpv1.ConditionReason = "NameConflict"
)

const (
	errNotUserGroup  = "managed resource is not a UserGroup"
	errTrackUsage    = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errExtractToken  = "cannot extract bot token from secret"
	errValidateToken = "invalid bot token"
	errObserve       = "cannot list user groups"
	errCreate        = "cannot create user group"
	errUpdate        = "cannot update user group"
	errDelete        = "cannot disable user group"
)

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage *resource.LegacyProviderConfigUsageTracker
	newFn func(token string, opts ...slack.ClientOption) slack.ClientAPI
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*usergroupv1alpha1.UserGroup)
	if !ok {
		return nil, xperrors.New(errNotUserGroup)
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
	cr, ok := mg.(*usergroupv1alpha1.UserGroup)
	if !ok {
		return managed.ExternalObservation{}, xperrors.New(errNotUserGroup)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	groups, err := e.client.ListUserGroups(ctx)
	if err != nil {
		return managed.ExternalObservation{}, xperrors.Wrap(err, errObserve)
	}

	var found *slack.UserGroup
	for i := range groups {
		if groups[i].ID == externalName {
			found = &groups[i]
			break
		}
	}

	if found == nil || !found.IsEnabled {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider = usergroupv1alpha1.UserGroupObservation{
		ID:         found.ID,
		IsEnabled:  found.IsEnabled,
		CreatedBy:  found.CreatedBy,
		DateCreate: found.DateCreate,
	}

	upToDate := isUpToDate(cr.Spec.ForProvider, found)
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*usergroupv1alpha1.UserGroup)
	if !ok {
		return managed.ExternalCreation{}, xperrors.New(errNotUserGroup)
	}

	params := slack.UserGroupParams{
		Name:        cr.Spec.ForProvider.Name,
		Handle:      cr.Spec.ForProvider.Handle,
		Description: ptrValueOrEmpty(cr.Spec.ForProvider.Description),
	}

	ug, err := e.client.CreateUserGroup(ctx, params)
	if err != nil {
		var slackErr *slack.SlackError
		if errors.As(err, &slackErr) && slackErr.Code == "name_already_exists" {
			cr.SetConditions(xpv1.Condition{
				Type:    xpv1.TypeSynced,
				Status:  "False",
				Reason:  ReasonNameConflict,
				Message: "user group name already exists",
			})
		}
		return managed.ExternalCreation{}, xperrors.Wrap(err, errCreate)
	}

	meta.SetExternalName(cr, ug.ID)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*usergroupv1alpha1.UserGroup)
	if !ok {
		return managed.ExternalUpdate{}, xperrors.New(errNotUserGroup)
	}

	externalName := meta.GetExternalName(cr)

	params := slack.UserGroupParams{
		Name:        cr.Spec.ForProvider.Name,
		Handle:      cr.Spec.ForProvider.Handle,
		Description: ptrValueOrEmpty(cr.Spec.ForProvider.Description),
	}

	if err := e.client.UpdateUserGroup(ctx, externalName, params); err != nil {
		var slackErr *slack.SlackError
		if errors.As(err, &slackErr) && slackErr.Code == "name_already_exists" {
			cr.SetConditions(xpv1.Condition{
				Type:    xpv1.TypeSynced,
				Status:  "False",
				Reason:  ReasonNameConflict,
				Message: "user group name already exists",
			})
		}
		return managed.ExternalUpdate{}, xperrors.Wrap(err, errUpdate)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*usergroupv1alpha1.UserGroup)
	if !ok {
		return managed.ExternalDelete{}, xperrors.New(errNotUserGroup)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	err := e.client.DisableUserGroup(ctx, externalName)
	if err != nil {
		return managed.ExternalDelete{}, xperrors.Wrap(err, errDelete)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op for the Slack client.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

func isUpToDate(desired usergroupv1alpha1.UserGroupParameters, observed *slack.UserGroup) bool {
	if desired.Name != observed.Name {
		return false
	}
	if desired.Handle != observed.Handle {
		return false
	}
	if ptrValueOrEmpty(desired.Description) != observed.Description {
		return false
	}
	return true
}

// ptrValueOrEmpty returns the dereferenced string pointer value, or empty
// string if the pointer is nil.
func ptrValueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
