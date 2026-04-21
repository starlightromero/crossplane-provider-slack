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

package usergroup

import (
	"context"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"

	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
)

func strPtr(s string) *string { return &s }

func TestObserve_ExistingGroup(t *testing.T) {
	mock := &mockClientAPI{
		listUserGroupsFn: func(_ context.Context) ([]slack.UserGroup, error) {
			return []slack.UserGroup{
				{
					ID:          "S12345678",
					Name:        "my-group",
					Handle:      "my-handle",
					Description: "a description",
					IsEnabled:   true,
					CreatedBy:   "U001",
					DateCreate:  1700000000,
				},
			}, nil
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupCR("my-group", "my-handle", strPtr("a description"))
	meta.SetExternalName(cr, "S12345678")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=true")
	}
	if cr.Status.AtProvider.ID != "S12345678" {
		t.Fatalf("expected AtProvider.ID=S12345678, got %q", cr.Status.AtProvider.ID)
	}
	if !cr.Status.AtProvider.IsEnabled {
		t.Fatal("expected AtProvider.IsEnabled=true")
	}
}

func TestObserve_NonExistentGroup(t *testing.T) {
	mock := &mockClientAPI{
		listUserGroupsFn: func(_ context.Context) ([]slack.UserGroup, error) {
			return []slack.UserGroup{}, nil
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupCR("missing-group", "missing", nil)
	meta.SetExternalName(cr, "S99999999")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false")
	}
}

func TestObserve_DisabledGroup(t *testing.T) {
	mock := &mockClientAPI{
		listUserGroupsFn: func(_ context.Context) ([]slack.UserGroup, error) {
			return []slack.UserGroup{
				{
					ID:        "S12345678",
					Name:      "disabled-group",
					Handle:    "disabled",
					IsEnabled: false,
				},
			}, nil
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupCR("disabled-group", "disabled", nil)
	meta.SetExternalName(cr, "S12345678")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false for disabled group")
	}
}

func TestObserve_NoExternalName(t *testing.T) {
	mock := &mockClientAPI{}
	ext := &external{client: mock}
	cr := newUserGroupCR("new-group", "new-handle", nil)

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false when no external-name")
	}
}

func TestObserve_DriftDetection(t *testing.T) {
	tests := []struct {
		name          string
		desiredName   string
		desiredHandle string
		desiredDesc   *string
		remoteName    string
		remoteHandle  string
		remoteDesc    string
		wantUpToDate  bool
	}{
		{
			name:          "all match",
			desiredName:   "group",
			desiredHandle: "handle",
			desiredDesc:   strPtr("desc"),
			remoteName:    "group",
			remoteHandle:  "handle",
			remoteDesc:    "desc",
			wantUpToDate:  true,
		},
		{
			name:          "name differs",
			desiredName:   "new-name",
			desiredHandle: "handle",
			desiredDesc:   strPtr("desc"),
			remoteName:    "old-name",
			remoteHandle:  "handle",
			remoteDesc:    "desc",
			wantUpToDate:  false,
		},
		{
			name:          "handle differs",
			desiredName:   "group",
			desiredHandle: "new-handle",
			desiredDesc:   strPtr("desc"),
			remoteName:    "group",
			remoteHandle:  "old-handle",
			remoteDesc:    "desc",
			wantUpToDate:  false,
		},
		{
			name:          "description differs",
			desiredName:   "group",
			desiredHandle: "handle",
			desiredDesc:   strPtr("new-desc"),
			remoteName:    "group",
			remoteHandle:  "handle",
			remoteDesc:    "old-desc",
			wantUpToDate:  false,
		},
		{
			name:          "nil description matches empty remote",
			desiredName:   "group",
			desiredHandle: "handle",
			desiredDesc:   nil,
			remoteName:    "group",
			remoteHandle:  "handle",
			remoteDesc:    "",
			wantUpToDate:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClientAPI{
				listUserGroupsFn: func(_ context.Context) ([]slack.UserGroup, error) {
					return []slack.UserGroup{
						{
							ID:          "S12345678",
							Name:        tt.remoteName,
							Handle:      tt.remoteHandle,
							Description: tt.remoteDesc,
							IsEnabled:   true,
						},
					}, nil
				},
			}

			ext := &external{client: mock}
			cr := newUserGroupCR(tt.desiredName, tt.desiredHandle, tt.desiredDesc)
			meta.SetExternalName(cr, "S12345678")

			obs, err := ext.Observe(context.Background(), cr)
			if err != nil {
				t.Fatalf("Observe returned error: %v", err)
			}
			if obs.ResourceUpToDate != tt.wantUpToDate {
				t.Fatalf("ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tt.wantUpToDate)
			}
		})
	}
}

func TestCreate_Success(t *testing.T) {
	mock := &mockClientAPI{
		createUserGroupFn: func(_ context.Context, params slack.UserGroupParams) (*slack.UserGroup, error) {
			return &slack.UserGroup{
				ID:     "SNEWGRP01",
				Name:   params.Name,
				Handle: params.Handle,
			}, nil
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupCR("new-group", "new-handle", strPtr("a description"))

	_, err := ext.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	got := meta.GetExternalName(cr)
	if got != "SNEWGRP01" {
		t.Fatalf("external-name = %q, want SNEWGRP01", got)
	}
}

func TestCreate_NameAlreadyExistsError(t *testing.T) {
	mock := &mockClientAPI{
		createUserGroupFn: func(_ context.Context, _ slack.UserGroupParams) (*slack.UserGroup, error) {
			return nil, &slack.SlackError{Code: "name_already_exists", Message: "user group name already exists"}
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupCR("taken-group", "taken-handle", nil)

	_, err := ext.Create(context.Background(), cr)
	if err == nil {
		t.Fatal("expected error from Create with name_already_exists")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonNameConflict {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonNameConflict, cond.Reason)
	}
	if cond.Status != "False" {
		t.Fatalf("expected Synced status=False, got %s", cond.Status)
	}
}

func TestUpdate_Success(t *testing.T) {
	var updateCalled bool
	mock := &mockClientAPI{
		updateUserGroupFn: func(_ context.Context, groupID string, params slack.UserGroupParams) error {
			updateCalled = true
			if groupID != "S12345678" {
				t.Fatalf("update called with ID=%q, want S12345678", groupID)
			}
			if params.Name != "updated-name" {
				t.Fatalf("update called with name=%q, want updated-name", params.Name)
			}
			return nil
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupCR("updated-name", "updated-handle", strPtr("updated desc"))
	meta.SetExternalName(cr, "S12345678")

	_, err := ext.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !updateCalled {
		t.Fatal("expected UpdateUserGroup to be called")
	}
}

func TestUpdate_NameAlreadyExistsError(t *testing.T) {
	mock := &mockClientAPI{
		updateUserGroupFn: func(_ context.Context, _ string, _ slack.UserGroupParams) error {
			return &slack.SlackError{Code: "name_already_exists", Message: "user group name already exists"}
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupCR("conflict-name", "conflict-handle", nil)
	meta.SetExternalName(cr, "S12345678")

	_, err := ext.Update(context.Background(), cr)
	if err == nil {
		t.Fatal("expected error from Update with name_already_exists")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonNameConflict {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonNameConflict, cond.Reason)
	}
}

func TestDelete_Success(t *testing.T) {
	var disableCalled bool
	mock := &mockClientAPI{
		disableUserGroupFn: func(_ context.Context, groupID string) error {
			disableCalled = true
			if groupID != "S12345678" {
				t.Fatalf("disable called with %q, want S12345678", groupID)
			}
			return nil
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupCR("my-group", "my-handle", nil)
	meta.SetExternalName(cr, "S12345678")

	_, err := ext.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !disableCalled {
		t.Fatal("expected DisableUserGroup to be called")
	}
}

func TestDelete_NoExternalName(t *testing.T) {
	mock := &mockClientAPI{}
	ext := &external{client: mock}
	cr := newUserGroupCR("my-group", "my-handle", nil)

	_, err := ext.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete should not return error when no external-name, got: %v", err)
	}
}
