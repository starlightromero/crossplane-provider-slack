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

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pgregory.net/rapid"

	usergroupv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/usergroup/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
)

// mockClientAPI implements slack.ClientAPI for testing usergroup controller.
type mockClientAPI struct {
	createUserGroupFn  func(ctx context.Context, params slack.UserGroupParams) (*slack.UserGroup, error)
	listUserGroupsFn   func(ctx context.Context) ([]slack.UserGroup, error)
	updateUserGroupFn  func(ctx context.Context, groupID string, params slack.UserGroupParams) error
	disableUserGroupFn func(ctx context.Context, groupID string) error
}

func (m *mockClientAPI) CreateUserGroup(ctx context.Context, params slack.UserGroupParams) (*slack.UserGroup, error) {
	if m.createUserGroupFn != nil {
		return m.createUserGroupFn(ctx, params)
	}
	return nil, nil
}

func (m *mockClientAPI) ListUserGroups(ctx context.Context) ([]slack.UserGroup, error) {
	if m.listUserGroupsFn != nil {
		return m.listUserGroupsFn(ctx)
	}
	return nil, nil
}

func (m *mockClientAPI) UpdateUserGroup(ctx context.Context, groupID string, params slack.UserGroupParams) error {
	if m.updateUserGroupFn != nil {
		return m.updateUserGroupFn(ctx, groupID, params)
	}
	return nil
}

func (m *mockClientAPI) DisableUserGroup(ctx context.Context, groupID string) error {
	if m.disableUserGroupFn != nil {
		return m.disableUserGroupFn(ctx, groupID)
	}
	return nil
}

// Stub implementations for non-usergroup methods.
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
func (m *mockClientAPI) ListUserGroupMembers(context.Context, string) ([]string, error) {
	panic("not implemented")
}
func (m *mockClientAPI) UpdateUserGroupMembers(context.Context, string, []string) error {
	panic("not implemented")
}
func (m *mockClientAPI) LookupUserByEmail(context.Context, string) (*slack.User, error) {
	panic("not implemented")
}

// Generators

// genGroupName generates valid user group names (1-50 chars).
func genGroupName() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		length := rapid.IntRange(1, 50).Draw(t, "nameLen")
		chars := make([]byte, length)
		for i := range chars {
			chars[i] = rapid.ByteRange('a', 'z').Draw(t, "char")
		}
		return string(chars)
	})
}

// genHandle generates valid user group handles matching ^[a-z0-9][a-z0-9._-]*$.
func genHandle() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		length := rapid.IntRange(1, 30).Draw(t, "handleLen")
		chars := make([]byte, length)
		// First char must be [a-z0-9]
		firstAlphabet := "abcdefghijklmnopqrstuvwxyz0123456789"
		idx := rapid.IntRange(0, len(firstAlphabet)-1).Draw(t, "firstIdx")
		chars[0] = firstAlphabet[idx]
		// Remaining chars can be [a-z0-9._-]
		restAlphabet := "abcdefghijklmnopqrstuvwxyz0123456789._-"
		for i := 1; i < length; i++ {
			idx := rapid.IntRange(0, len(restAlphabet)-1).Draw(t, "restIdx")
			chars[i] = restAlphabet[idx]
		}
		return string(chars)
	})
}

// genGroupID generates mock Slack user group IDs (S + 8-11 alphanumeric chars).
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

// genOptionalString generates an optional string pointer.
func genOptionalString() *rapid.Generator[*string] {
	return rapid.Custom[*string](func(t *rapid.T) *string {
		isNil := rapid.Bool().Draw(t, "isNil")
		if isNil {
			return nil
		}
		s := rapid.StringMatching(`[a-z]{0,100}`).Draw(t, "str")
		return &s
	})
}

// newUserGroupCR creates a UserGroup custom resource for testing.
func newUserGroupCR(name, handle string, description *string) *usergroupv1alpha1.UserGroup {
	return &usergroupv1alpha1.UserGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "usergroup.slack.crossplane.io/v1alpha1",
			Kind:       "UserGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-usergroup",
			Annotations: map[string]string{},
		},
		Spec: usergroupv1alpha1.UserGroupSpec{
			ForProvider: usergroupv1alpha1.UserGroupParameters{
				Name:        name,
				Handle:      handle,
				Description: description,
			},
		},
	}
}

// Feature: crossplane-provider-slack, Property 11: UserGroup Update dispatches update for changed name, handle, or description
// **Validates: Requirements 6.6**

func TestProperty_UserGroupUpdateDispatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		desiredName := genGroupName().Draw(t, "desiredName")
		desiredHandle := genHandle().Draw(t, "desiredHandle")
		desiredDescription := genOptionalString().Draw(t, "desiredDescription")

		remoteName := genGroupName().Draw(t, "remoteName")
		remoteHandle := genHandle().Draw(t, "remoteHandle")
		remoteDescription := rapid.StringMatching(`[a-z]{0,100}`).Draw(t, "remoteDescription")

		groupID := genGroupID().Draw(t, "groupID")

		// Ensure at least one field differs so Update is meaningful
		if desiredName == remoteName &&
			desiredHandle == remoteHandle &&
			ptrValueOrEmpty(desiredDescription) == remoteDescription {
			desiredName = desiredName + "x"
		}

		var updateCalled bool
		var receivedParams slack.UserGroupParams

		mock := &mockClientAPI{
			updateUserGroupFn: func(_ context.Context, id string, params slack.UserGroupParams) error {
				updateCalled = true
				receivedParams = params
				if id != groupID {
					t.Fatalf("UpdateUserGroup called with ID=%q, want %q", id, groupID)
				}
				return nil
			},
		}

		ext := &external{client: mock}
		cr := newUserGroupCR(desiredName, desiredHandle, desiredDescription)
		meta.SetExternalName(cr, groupID)

		_, err := ext.Update(context.Background(), cr)
		if err != nil {
			t.Fatalf("Update returned unexpected error: %v", err)
		}

		// Update should always be called when there's a difference
		if !updateCalled {
			t.Fatal("expected UpdateUserGroup to be called")
		}

		// Verify the params passed to update match desired state
		if receivedParams.Name != desiredName {
			t.Fatalf("UpdateUserGroup name=%q, want %q", receivedParams.Name, desiredName)
		}
		if receivedParams.Handle != desiredHandle {
			t.Fatalf("UpdateUserGroup handle=%q, want %q", receivedParams.Handle, desiredHandle)
		}
		expectedDesc := ptrValueOrEmpty(desiredDescription)
		if receivedParams.Description != expectedDesc {
			t.Fatalf("UpdateUserGroup description=%q, want %q", receivedParams.Description, expectedDesc)
		}
	})
}

// Ensure external implements managed.ExternalClient at compile time.
var _ managed.ExternalClient = &external{}

// Ensure mockClientAPI implements slack.ClientAPI at compile time.
var _ slack.ClientAPI = &mockClientAPI{}

// Ensure UserGroup implements resource.Managed at compile time.
var _ resource.Managed = &usergroupv1alpha1.UserGroup{}
