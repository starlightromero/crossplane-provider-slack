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

package usergroupmembers

import (
	"context"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
)

func TestObserve_MembersMatch(t *testing.T) {
	mock := &mockClientAPI{
		listUserGroupMembersFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"U001", "U002", "U003"}, nil
		},
		lookupUserByEmailFn: func(_ context.Context, email string) (*slack.User, error) {
			mapping := map[string]string{
				"alice@example.com": "U001",
				"bob@example.com":   "U002",
				"carol@example.com": "U003",
			}
			if id, ok := mapping[email]; ok {
				return &slack.User{ID: id}, nil
			}
			return nil, &slack.SlackError{Code: "users_not_found"}
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupMembersCR("S12345678", []string{"alice@example.com", "bob@example.com", "carol@example.com"})
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
	if cr.Status.AtProvider.MemberCount != 3 {
		t.Fatalf("expected MemberCount=3, got %d", cr.Status.AtProvider.MemberCount)
	}
}

func TestObserve_MembersDiffer(t *testing.T) {
	mock := &mockClientAPI{
		listUserGroupMembersFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"U001", "U002"}, nil
		},
		lookupUserByEmailFn: func(_ context.Context, email string) (*slack.User, error) {
			mapping := map[string]string{
				"alice@example.com": "U001",
				"bob@example.com":   "U002",
				"carol@example.com": "U003",
			}
			if id, ok := mapping[email]; ok {
				return &slack.User{ID: id}, nil
			}
			return nil, &slack.SlackError{Code: "users_not_found"}
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupMembersCR("S12345678", []string{"alice@example.com", "bob@example.com", "carol@example.com"})
	meta.SetExternalName(cr, "S12345678")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=false when members differ")
	}
}

func TestObserve_NoExternalName(t *testing.T) {
	mock := &mockClientAPI{}
	ext := &external{client: mock}
	cr := newUserGroupMembersCR("S12345678", []string{"alice@example.com"})

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false when no external-name")
	}
}

func TestObserve_UnresolvableEmail(t *testing.T) {
	mock := &mockClientAPI{
		listUserGroupMembersFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"U001"}, nil
		},
		lookupUserByEmailFn: func(_ context.Context, email string) (*slack.User, error) {
			if email == "alice@example.com" {
				return &slack.User{ID: "U001"}, nil
			}
			return nil, &slack.SlackError{Code: "users_not_found"}
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupMembersCR("S12345678", []string{"alice@example.com", "unknown@example.com"})
	meta.SetExternalName(cr, "S12345678")

	_, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonUserNotFound {
		t.Fatalf("expected Synced reason=%s, got %s", ReasonUserNotFound, cond.Reason)
	}
	if cond.Status != "False" {
		t.Fatalf("expected Synced status=False, got %s", cond.Status)
	}
}

func TestCreate_Success(t *testing.T) {
	var updatedIDs []string
	mock := &mockClientAPI{
		lookupUserByEmailFn: func(_ context.Context, email string) (*slack.User, error) {
			mapping := map[string]string{
				"alice@example.com": "U001",
				"bob@example.com":   "U002",
			}
			if id, ok := mapping[email]; ok {
				return &slack.User{ID: id}, nil
			}
			return nil, &slack.SlackError{Code: "users_not_found"}
		},
		updateUserGroupMembersFn: func(_ context.Context, groupID string, userIDs []string) error {
			if groupID != "S12345678" {
				t.Fatalf("expected groupID=S12345678, got %q", groupID)
			}
			updatedIDs = userIDs
			return nil
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupMembersCR("S12345678", []string{"alice@example.com", "bob@example.com"})

	_, err := ext.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if meta.GetExternalName(cr) != "S12345678" {
		t.Fatalf("expected external-name=S12345678, got %q", meta.GetExternalName(cr))
	}
	if len(updatedIDs) != 2 {
		t.Fatalf("expected 2 user IDs, got %d", len(updatedIDs))
	}
}

func TestUpdate_Success(t *testing.T) {
	var updatedIDs []string
	mock := &mockClientAPI{
		lookupUserByEmailFn: func(_ context.Context, email string) (*slack.User, error) {
			mapping := map[string]string{
				"alice@example.com": "U001",
				"bob@example.com":   "U002",
				"carol@example.com": "U003",
			}
			if id, ok := mapping[email]; ok {
				return &slack.User{ID: id}, nil
			}
			return nil, &slack.SlackError{Code: "users_not_found"}
		},
		updateUserGroupMembersFn: func(_ context.Context, groupID string, userIDs []string) error {
			if groupID != "S12345678" {
				t.Fatalf("expected groupID=S12345678, got %q", groupID)
			}
			updatedIDs = userIDs
			return nil
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupMembersCR("S12345678", []string{"alice@example.com", "bob@example.com", "carol@example.com"})
	meta.SetExternalName(cr, "S12345678")

	_, err := ext.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if len(updatedIDs) != 3 {
		t.Fatalf("expected 3 user IDs, got %d", len(updatedIDs))
	}
}

func TestUpdate_WithUnresolvableEmails(t *testing.T) {
	var updatedIDs []string
	mock := &mockClientAPI{
		lookupUserByEmailFn: func(_ context.Context, email string) (*slack.User, error) {
			if email == "alice@example.com" {
				return &slack.User{ID: "U001"}, nil
			}
			return nil, &slack.SlackError{Code: "users_not_found"}
		},
		updateUserGroupMembersFn: func(_ context.Context, _ string, userIDs []string) error {
			updatedIDs = userIDs
			return nil
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupMembersCR("S12345678", []string{"alice@example.com", "unknown@example.com"})
	meta.SetExternalName(cr, "S12345678")

	_, err := ext.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	// Should still update with the resolved IDs
	if len(updatedIDs) != 1 {
		t.Fatalf("expected 1 user ID, got %d", len(updatedIDs))
	}
	if updatedIDs[0] != "U001" {
		t.Fatalf("expected U001, got %q", updatedIDs[0])
	}

	// Should set Synced=False
	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonUserNotFound {
		t.Fatalf("expected Synced reason=%s, got %s", ReasonUserNotFound, cond.Reason)
	}
}

func TestDelete_Success(t *testing.T) {
	var deletedGroupID string
	var deletedIDs []string
	mock := &mockClientAPI{
		updateUserGroupMembersFn: func(_ context.Context, groupID string, userIDs []string) error {
			deletedGroupID = groupID
			deletedIDs = userIDs
			return nil
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupMembersCR("S12345678", []string{"alice@example.com"})
	meta.SetExternalName(cr, "S12345678")

	_, err := ext.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deletedGroupID != "S12345678" {
		t.Fatalf("expected groupID=S12345678, got %q", deletedGroupID)
	}
	if len(deletedIDs) != 0 {
		t.Fatalf("expected empty user list on delete, got %v", deletedIDs)
	}
}

func TestDelete_NoExternalName(t *testing.T) {
	mock := &mockClientAPI{}
	ext := &external{client: mock}
	cr := newUserGroupMembersCR("S12345678", []string{"alice@example.com"})

	_, err := ext.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete should not return error when no external-name, got: %v", err)
	}
}

func TestObserve_OrderIndependent(t *testing.T) {
	// Current members in one order, desired resolves to same set in different order
	mock := &mockClientAPI{
		listUserGroupMembersFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"U003", "U001", "U002"}, nil
		},
		lookupUserByEmailFn: func(_ context.Context, email string) (*slack.User, error) {
			mapping := map[string]string{
				"alice@example.com": "U001",
				"bob@example.com":   "U002",
				"carol@example.com": "U003",
			}
			if id, ok := mapping[email]; ok {
				return &slack.User{ID: id}, nil
			}
			return nil, &slack.SlackError{Code: "users_not_found"}
		},
	}

	ext := &external{client: mock}
	cr := newUserGroupMembersCR("S12345678", []string{"alice@example.com", "bob@example.com", "carol@example.com"})
	meta.SetExternalName(cr, "S12345678")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if !obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=true for same set in different order")
	}
}
