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

package bookmark

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pgregory.net/rapid"

	bookmarkv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/bookmark/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
)

// mockClientAPI implements slack.ClientAPI for testing bookmark controller.
type mockClientAPI struct {
	addBookmarkFn    func(ctx context.Context, channelID string, params slack.BookmarkParams) (*slack.Bookmark, error)
	listBookmarksFn  func(ctx context.Context, channelID string) ([]slack.Bookmark, error)
	editBookmarkFn   func(ctx context.Context, channelID, bookmarkID string, params slack.BookmarkParams) error
	removeBookmarkFn func(ctx context.Context, channelID, bookmarkID string) error
}

func (m *mockClientAPI) AddBookmark(ctx context.Context, channelID string, params slack.BookmarkParams) (*slack.Bookmark, error) {
	if m.addBookmarkFn != nil {
		return m.addBookmarkFn(ctx, channelID, params)
	}
	return nil, nil
}

func (m *mockClientAPI) ListBookmarks(ctx context.Context, channelID string) ([]slack.Bookmark, error) {
	if m.listBookmarksFn != nil {
		return m.listBookmarksFn(ctx, channelID)
	}
	return nil, nil
}

func (m *mockClientAPI) EditBookmark(ctx context.Context, channelID, bookmarkID string, params slack.BookmarkParams) error {
	if m.editBookmarkFn != nil {
		return m.editBookmarkFn(ctx, channelID, bookmarkID, params)
	}
	return nil
}

func (m *mockClientAPI) RemoveBookmark(ctx context.Context, channelID, bookmarkID string) error {
	if m.removeBookmarkFn != nil {
		return m.removeBookmarkFn(ctx, channelID, bookmarkID)
	}
	return nil
}

// Stub implementations for non-bookmark methods.
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

func genBookmarkID() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		length := rapid.IntRange(8, 11).Draw(t, "idLen")
		chars := make([]byte, length)
		alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		for i := range chars {
			idx := rapid.IntRange(0, len(alphabet)-1).Draw(t, "idx")
			chars[i] = alphabet[idx]
		}
		return "Bk" + string(chars)
	})
}

func genChannelID() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		length := rapid.IntRange(8, 11).Draw(t, "idLen")
		chars := make([]byte, length)
		alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		for i := range chars {
			idx := rapid.IntRange(0, len(alphabet)-1).Draw(t, "idx")
			chars[i] = alphabet[idx]
		}
		return "C" + string(chars)
	})
}

func genTitle() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		length := rapid.IntRange(1, 50).Draw(t, "titleLen")
		chars := make([]byte, length)
		for i := range chars {
			chars[i] = rapid.ByteRange('a', 'z').Draw(t, "char")
		}
		return string(chars)
	})
}

func genLink() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		length := rapid.IntRange(3, 30).Draw(t, "pathLen")
		chars := make([]byte, length)
		for i := range chars {
			chars[i] = rapid.ByteRange('a', 'z').Draw(t, "char")
		}
		return "https://example.com/" + string(chars)
	})
}

func newBookmarkCR(channelID, title, link string) *bookmarkv1alpha1.ConversationBookmark {
	return &bookmarkv1alpha1.ConversationBookmark{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "bookmark.slack.crossplane.io/v1alpha1",
			Kind:       "ConversationBookmark",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-bookmark",
			Annotations: map[string]string{},
		},
		Spec: bookmarkv1alpha1.ConversationBookmarkSpec{
			ForProvider: bookmarkv1alpha1.ConversationBookmarkParameters{
				ConversationID: &channelID,
				Title:          title,
				Type:           "link",
				Link:           link,
			},
		},
	}
}

// Feature: crossplane-provider-slack, Property 10: Bookmark Update dispatches edit for changed title or link
// **Validates: Requirements 4.6**

func TestProperty_BookmarkUpdateDispatchesEdit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		desiredTitle := genTitle().Draw(t, "desiredTitle")
		desiredLink := genLink().Draw(t, "desiredLink")
		observedTitle := genTitle().Draw(t, "observedTitle")
		observedLink := genLink().Draw(t, "observedLink")

		channelID := genChannelID().Draw(t, "channelID")
		bookmarkID := genBookmarkID().Draw(t, "bookmarkID")

		// Ensure at least one field differs
		if desiredTitle == observedTitle && desiredLink == observedLink {
			desiredTitle = desiredTitle + "x"
		}

		var editCalled bool
		var editParams slack.BookmarkParams

		mock := &mockClientAPI{
			editBookmarkFn: func(_ context.Context, _, _ string, params slack.BookmarkParams) error {
				editCalled = true
				editParams = params
				return nil
			},
		}

		ext := &external{client: mock, kube: nil}
		cr := newBookmarkCR(channelID, desiredTitle, desiredLink)
		meta.SetExternalName(cr, bookmarkID)

		_, err := ext.Update(context.Background(), cr)
		if err != nil {
			t.Fatalf("Update returned unexpected error: %v", err)
		}

		if !editCalled {
			t.Fatal("expected EditBookmark to be called when title or link differs")
		}

		if editParams.Title != desiredTitle {
			t.Fatalf("EditBookmark title = %q, want %q", editParams.Title, desiredTitle)
		}
		if editParams.Link != desiredLink {
			t.Fatalf("EditBookmark link = %q, want %q", editParams.Link, desiredLink)
		}
	})
}

// Ensure external implements managed.ExternalClient at compile time.
var _ managed.ExternalClient = &external{}

// Ensure mockClientAPI implements slack.ClientAPI at compile time.
var _ slack.ClientAPI = &mockClientAPI{}

// Ensure ConversationBookmark implements resource.Managed at compile time.
var _ resource.Managed = &bookmarkv1alpha1.ConversationBookmark{}
