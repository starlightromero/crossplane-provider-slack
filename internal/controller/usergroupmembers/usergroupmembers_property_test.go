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
	"fmt"
	"sort"
	"strings"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pgregory.net/rapid"

	usergroupmembersv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/usergroupmembers/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
)

// mockClientAPI implements slack.ClientAPI for testing.
type mockClientAPI struct {
	listUserGroupMembersFn   func(ctx context.Context, groupID string) ([]string, error)
	updateUserGroupMembersFn func(ctx context.Context, groupID string, userIDs []string) error
	lookupUserByEmailFn      func(ctx context.Context, email string) (*slack.User, error)
}

func (m *mockClientAPI) ListUserGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	if m.listUserGroupMembersFn != nil {
		return m.listUserGroupMembersFn(ctx, groupID)
	}
	return nil, nil
}

func (m *mockClientAPI) UpdateUserGroupMembers(ctx context.Context, groupID string, userIDs []string) error {
	if m.updateUserGroupMembersFn != nil {
		return m.updateUserGroupMembersFn(ctx, groupID, userIDs)
	}
	return nil
}

func (m *mockClientAPI) LookupUserByEmail(ctx context.Context, email string) (*slack.User, error) {
	if m.lookupUserByEmailFn != nil {
		return m.lookupUserByEmailFn(ctx, email)
	}
	return nil, nil
}

// Stub implementations for non-usergroupmembers methods.
func (m *mockClientAPI) CreateConversation(context.Context, string, bool) (*slack.Conversation, error) {
	panic("not implemented")
}
func (m *mockClientAPI) GetConversationInfo(context.Context, string) (*slack.Conversation, error) {
	panic("not implemented")
}
func (m *mockClientAPI) RenameConversation(context.Context, string, string) error {
	panic("not implemented")
}
func (m *mockClientAPI) SetConversationTopic(context.Context, string, string) error {
	panic("not implemented")
}
func (m *mockClientAPI) SetConversationPurpose(context.Context, string, string) error {
	panic("not implemented")
}
func (m *mockClientAPI) ArchiveConversation(context.Context, string) error {
	panic("not implemented")
}
func (m *mockClientAPI) AddBookmark(context.Context, string, slack.BookmarkParams) (*slack.Bookmark, error) {
	panic("not implemented")
}
func (m *mockClientAPI) ListBookmarks(context.Context, string) ([]slack.Bookmark, error) {
	panic("not implemented")
}
func (m *mockClientAPI) EditBookmark(context.Context, string, string, slack.BookmarkParams) error {
	panic("not implemented")
}
func (m *mockClientAPI) RemoveBookmark(context.Context, string, string) error {
	panic("not implemented")
}
func (m *mockClientAPI) AddPin(context.Context, string, string) error { panic("not implemented") }
func (m *mockClientAPI) ListPins(context.Context, string) ([]slack.Pin, error) {
	panic("not implemented")
}
func (m *mockClientAPI) RemovePin(context.Context, string, string) error { panic("not implemented") }
func (m *mockClientAPI) CreateUserGroup(context.Context, slack.UserGroupParams) (*slack.UserGroup, error) {
	panic("not implemented")
}
func (m *mockClientAPI) ListUserGroups(context.Context) ([]slack.UserGroup, error) {
	panic("not implemented")
}
func (m *mockClientAPI) UpdateUserGroup(context.Context, string, slack.UserGroupParams) error {
	panic("not implemented")
}
func (m *mockClientAPI) DisableUserGroup(context.Context, string) error { panic("not implemented") }

// Generators

// genEmail generates valid-looking email addresses.
func genEmail() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		user := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "user")
		domain := rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "domain")
		return fmt.Sprintf("%s@%s.com", user, domain)
	})
}

// genUserID generates mock Slack user IDs (U + 8-11 alphanumeric chars).
func genUserID() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		length := rapid.IntRange(8, 11).Draw(t, "idLen")
		chars := make([]byte, length)
		alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		for i := range chars {
			idx := rapid.IntRange(0, len(alphabet)-1).Draw(t, "idx")
			chars[i] = alphabet[idx]
		}
		return "U" + string(chars)
	})
}

// genGroupID generates mock Slack user group IDs.
func genGroupID() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		length := rapid.IntRange(8, 11).Draw(t, "idLen")
		chars := make([]byte, length)
		alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		for i := range chars {
			idx := rapid.IntRange(0, len(alphabet)-1).Draw(t, "idx")
			chars[i] = alphabet[idx]
		}
		return "S" + string(chars)
	})
}

// newUserGroupMembersCR creates a UserGroupMembers custom resource for testing.
func newUserGroupMembersCR(groupID string, emails []string) *usergroupmembersv1alpha1.UserGroupMembers {
	return &usergroupmembersv1alpha1.UserGroupMembers{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "usergroupmembers.slack.crossplane.io/v1alpha1",
			Kind:       "UserGroupMembers",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-members",
			Annotations: map[string]string{},
		},
		Spec: usergroupmembersv1alpha1.UserGroupMembersSpec{
			ForProvider: usergroupmembersv1alpha1.UserGroupMembersParameters{
				UserGroupID: &groupID,
				UserEmails:  emails,
			},
		},
	}
}

// Feature: crossplane-provider-slack, Property 12: Email resolution produces correct user ID list for membership update
// **Validates: Requirements 7.4, 7.6**

func TestProperty_EmailResolutionCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 1-10 unique emails with corresponding user IDs
		numEmails := rapid.IntRange(1, 10).Draw(t, "numEmails")
		emails := make([]string, numEmails)
		emailToID := make(map[string]string)

		for i := 0; i < numEmails; i++ {
			email := genEmail().Draw(t, fmt.Sprintf("email%d", i))
			// Ensure uniqueness
			for _, exists := emailToID[email]; exists; _, exists = emailToID[email] {
				email = genEmail().Draw(t, fmt.Sprintf("email%d_retry", i))
			}
			userID := genUserID().Draw(t, fmt.Sprintf("userID%d", i))
			emails[i] = email
			emailToID[email] = userID
		}

		groupID := genGroupID().Draw(t, "groupID")

		var updatedIDs []string
		mock := &mockClientAPI{
			lookupUserByEmailFn: func(_ context.Context, email string) (*slack.User, error) {
				id, ok := emailToID[email]
				if !ok {
					return nil, &slack.SlackError{Code: "users_not_found"}
				}
				return &slack.User{ID: id}, nil
			},
			updateUserGroupMembersFn: func(_ context.Context, gid string, userIDs []string) error {
				if gid != groupID {
					t.Fatalf("UpdateUserGroupMembers called with groupID=%q, want %q", gid, groupID)
				}
				updatedIDs = userIDs
				return nil
			},
		}

		ext := &external{client: mock}
		cr := newUserGroupMembersCR(groupID, emails)
		meta.SetExternalName(cr, groupID)

		_, err := ext.Update(context.Background(), cr)
		if err != nil {
			t.Fatalf("Update returned unexpected error: %v", err)
		}

		// Verify the resolved IDs match expected
		expectedIDs := make([]string, 0, len(emailToID))
		for _, email := range emails {
			expectedIDs = append(expectedIDs, emailToID[email])
		}

		sort.Strings(expectedIDs)
		sort.Strings(updatedIDs)

		if len(updatedIDs) != len(expectedIDs) {
			t.Fatalf("UpdateUserGroupMembers called with %d IDs, want %d", len(updatedIDs), len(expectedIDs))
		}
		for i := range expectedIDs {
			if updatedIDs[i] != expectedIDs[i] {
				t.Fatalf("UpdateUserGroupMembers ID[%d]=%q, want %q", i, updatedIDs[i], expectedIDs[i])
			}
		}
	})
}

// Feature: crossplane-provider-slack, Property 13: Unresolvable emails are reported with UserNotFound
// **Validates: Requirements 7.5**

func TestProperty_UnresolvableEmailReporting(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate some resolvable emails
		numResolvable := rapid.IntRange(0, 5).Draw(t, "numResolvable")
		resolvableEmails := make([]string, numResolvable)
		emailToID := make(map[string]string)

		for i := 0; i < numResolvable; i++ {
			email := genEmail().Draw(t, fmt.Sprintf("resolvable%d", i))
			for _, exists := emailToID[email]; exists; _, exists = emailToID[email] {
				email = genEmail().Draw(t, fmt.Sprintf("resolvable%d_retry", i))
			}
			userID := genUserID().Draw(t, fmt.Sprintf("userID%d", i))
			resolvableEmails[i] = email
			emailToID[email] = userID
		}

		// Generate at least one unresolvable email
		numUnresolvable := rapid.IntRange(1, 3).Draw(t, "numUnresolvable")
		unresolvableEmails := make([]string, numUnresolvable)
		for i := 0; i < numUnresolvable; i++ {
			email := genEmail().Draw(t, fmt.Sprintf("unresolvable%d", i))
			for _, exists := emailToID[email]; exists; _, exists = emailToID[email] {
				email = genEmail().Draw(t, fmt.Sprintf("unresolvable%d_retry", i))
			}
			unresolvableEmails[i] = email
		}

		allEmails := append(resolvableEmails, unresolvableEmails...)
		groupID := genGroupID().Draw(t, "groupID")

		mock := &mockClientAPI{
			lookupUserByEmailFn: func(_ context.Context, email string) (*slack.User, error) {
				id, ok := emailToID[email]
				if !ok {
					return nil, &slack.SlackError{Code: "users_not_found"}
				}
				return &slack.User{ID: id}, nil
			},
			listUserGroupMembersFn: func(_ context.Context, _ string) ([]string, error) {
				return []string{}, nil
			},
			updateUserGroupMembersFn: func(_ context.Context, _ string, _ []string) error {
				return nil
			},
		}

		ext := &external{client: mock}
		cr := newUserGroupMembersCR(groupID, allEmails)
		meta.SetExternalName(cr, groupID)

		_, err := ext.Observe(context.Background(), cr)
		if err != nil {
			t.Fatalf("Observe returned unexpected error: %v", err)
		}

		// Verify Synced=False with reason UserNotFound
		cond := cr.GetCondition(xpv1.TypeSynced)
		if cond.Reason != ReasonUserNotFound {
			t.Fatalf("expected Synced condition reason=%s, got %s", ReasonUserNotFound, cond.Reason)
		}
		if cond.Status != "False" {
			t.Fatalf("expected Synced status=False, got %s", cond.Status)
		}

		// Verify at least one unresolvable email appears in the message
		found := false
		for _, email := range unresolvableEmails {
			if strings.Contains(cond.Message, email) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected at least one unresolvable email in condition message, got: %q", cond.Message)
		}
	})
}

// Feature: crossplane-provider-slack, Property 14: Member set comparison is order-independent
// **Validates: Requirements 7.7**

func TestProperty_MemberSetComparisonOrderIndependent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a set of unique user IDs
		numMembers := rapid.IntRange(1, 20).Draw(t, "numMembers")
		memberSet := make(map[string]bool)
		members := make([]string, 0, numMembers)

		for len(members) < numMembers {
			id := genUserID().Draw(t, fmt.Sprintf("id%d", len(members)))
			if !memberSet[id] {
				memberSet[id] = true
				members = append(members, id)
			}
		}

		// Create a shuffled permutation using rapid
		permuted := rapid.Permutation(members).Draw(t, "perm")

		// Generate corresponding emails for the members
		emails := make([]string, len(members))
		emailToID := make(map[string]string)
		for i, id := range members {
			email := genEmail().Draw(t, fmt.Sprintf("email%d", i))
			for _, exists := emailToID[email]; exists; _, exists = emailToID[email] {
				email = genEmail().Draw(t, fmt.Sprintf("email%d_retry", i))
			}
			emails[i] = email
			emailToID[email] = id
		}

		groupID := genGroupID().Draw(t, "groupID")

		mock := &mockClientAPI{
			lookupUserByEmailFn: func(_ context.Context, email string) (*slack.User, error) {
				id, ok := emailToID[email]
				if !ok {
					return nil, &slack.SlackError{Code: "users_not_found"}
				}
				return &slack.User{ID: id}, nil
			},
			listUserGroupMembersFn: func(_ context.Context, _ string) ([]string, error) {
				// Return the permuted order
				return permuted, nil
			},
		}

		ext := &external{client: mock}
		cr := newUserGroupMembersCR(groupID, emails)
		meta.SetExternalName(cr, groupID)

		obs, err := ext.Observe(context.Background(), cr)
		if err != nil {
			t.Fatalf("Observe returned unexpected error: %v", err)
		}

		// The sets are the same, just in different order - should be up to date
		if !obs.ResourceUpToDate {
			t.Fatalf("expected ResourceUpToDate=true for same set in different order, members=%v, permuted=%v", members, permuted)
		}
	})
}

// Ensure external implements managed.ExternalClient at compile time.
var _ managed.ExternalClient = &external{}

// Ensure mockClientAPI implements slack.ClientAPI at compile time.
var _ slack.ClientAPI = &mockClientAPI{}

// Ensure UserGroupMembers implements resource.Managed at compile time.
var _ resource.Managed = &usergroupmembersv1alpha1.UserGroupMembers{}
