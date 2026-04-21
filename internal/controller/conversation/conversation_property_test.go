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

package conversation

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pgregory.net/rapid"

	conversationv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/conversation/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
)

// mockClientAPI implements slack.ClientAPI for testing conversation controller.
type mockClientAPI struct {
	createConversationFn     func(ctx context.Context, name string, isPrivate bool) (*slack.Conversation, error)
	getConversationInfoFn    func(ctx context.Context, channelID string) (*slack.Conversation, error)
	renameConversationFn     func(ctx context.Context, channelID, name string) error
	setConversationTopicFn   func(ctx context.Context, channelID, topic string) error
	setConversationPurposeFn func(ctx context.Context, channelID, purpose string) error
	archiveConversationFn    func(ctx context.Context, channelID string) error
}

func (m *mockClientAPI) CreateConversation(ctx context.Context, name string, isPrivate bool) (*slack.Conversation, error) {
	if m.createConversationFn != nil {
		return m.createConversationFn(ctx, name, isPrivate)
	}
	return nil, nil
}

func (m *mockClientAPI) GetConversationInfo(ctx context.Context, channelID string) (*slack.Conversation, error) {
	if m.getConversationInfoFn != nil {
		return m.getConversationInfoFn(ctx, channelID)
	}
	return nil, nil
}

func (m *mockClientAPI) RenameConversation(ctx context.Context, channelID, name string) error {
	if m.renameConversationFn != nil {
		return m.renameConversationFn(ctx, channelID, name)
	}
	return nil
}

func (m *mockClientAPI) SetConversationTopic(ctx context.Context, channelID, topic string) error {
	if m.setConversationTopicFn != nil {
		return m.setConversationTopicFn(ctx, channelID, topic)
	}
	return nil
}

func (m *mockClientAPI) SetConversationPurpose(ctx context.Context, channelID, purpose string) error {
	if m.setConversationPurposeFn != nil {
		return m.setConversationPurposeFn(ctx, channelID, purpose)
	}
	return nil
}

func (m *mockClientAPI) ArchiveConversation(ctx context.Context, channelID string) error {
	if m.archiveConversationFn != nil {
		return m.archiveConversationFn(ctx, channelID)
	}
	return nil
}

// Stub implementations for non-conversation methods.
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

// genChannelName generates valid Slack channel names (1-80 lowercase alphanumeric + hyphens).
func genChannelName() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		length := rapid.IntRange(1, 80).Draw(t, "nameLen")
		chars := make([]byte, length)
		for i := range chars {
			chars[i] = rapid.ByteRange('a', 'z').Draw(t, "char")
		}
		return string(chars)
	})
}

// genChannelID generates mock Slack channel IDs (C + 8-11 alphanumeric chars).
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

// genOptionalString generates an optional string pointer (nil or non-empty string up to 250 chars).
func genOptionalString() *rapid.Generator[*string] {
	return rapid.Custom[*string](func(t *rapid.T) *string {
		isNil := rapid.Bool().Draw(t, "isNil")
		if isNil {
			return nil
		}
		s := rapid.StringMatching(`[a-z]{0,250}`).Draw(t, "str")
		return &s
	})
}

// newConversationCR creates a Conversation custom resource for testing.
func newConversationCR(name string, topic, purpose *string) *conversationv1alpha1.Conversation {
	cr := &conversationv1alpha1.Conversation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "conversation.slack.crossplane.io/v1alpha1",
			Kind:       "Conversation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-conversation",
			Annotations: map[string]string{},
		},
		Spec: conversationv1alpha1.ConversationSpec{
			ForProvider: conversationv1alpha1.ConversationParameters{
				Name:    name,
				Topic:   topic,
				Purpose: purpose,
			},
		},
	}
	return cr
}

// Feature: crossplane-provider-slack, Property 7: Create stores the Slack-returned ID as external-name
// **Validates: Requirements 3.4, 8.3**

func TestProperty_CreateStoresExternalName(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		channelName := genChannelName().Draw(t, "channelName")
		channelID := genChannelID().Draw(t, "channelID")

		mock := &mockClientAPI{
			createConversationFn: func(_ context.Context, name string, _ bool) (*slack.Conversation, error) {
				return &slack.Conversation{ID: channelID, Name: name}, nil
			},
			setConversationTopicFn: func(_ context.Context, _, _ string) error {
				return nil
			},
			setConversationPurposeFn: func(_ context.Context, _, _ string) error {
				return nil
			},
		}

		ext := &external{client: mock}
		cr := newConversationCR(channelName, nil, nil)

		_, err := ext.Create(context.Background(), cr)
		if err != nil {
			t.Fatalf("Create returned unexpected error: %v", err)
		}

		got := meta.GetExternalName(cr)
		if got != channelID {
			t.Fatalf("external-name = %q, want %q", got, channelID)
		}
	})
}

// Feature: crossplane-provider-slack, Property 8: Observe correctly detects state drift between desired and remote
// **Validates: Requirements 3.5**

func TestProperty_ObserveDetectsStateDrift(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		desiredName := genChannelName().Draw(t, "desiredName")
		desiredTopic := genOptionalString().Draw(t, "desiredTopic")
		desiredPurpose := genOptionalString().Draw(t, "desiredPurpose")

		remoteName := genChannelName().Draw(t, "remoteName")
		remoteTopic := rapid.StringMatching(`[a-z]{0,250}`).Draw(t, "remoteTopic")
		remotePurpose := rapid.StringMatching(`[a-z]{0,250}`).Draw(t, "remotePurpose")

		channelID := genChannelID().Draw(t, "channelID")

		mock := &mockClientAPI{
			getConversationInfoFn: func(_ context.Context, _ string) (*slack.Conversation, error) {
				return &slack.Conversation{
					ID:      channelID,
					Name:    remoteName,
					Topic:   slack.Topic{Value: remoteTopic},
					Purpose: slack.Topic{Value: remotePurpose},
				}, nil
			},
		}

		ext := &external{client: mock}
		cr := newConversationCR(desiredName, desiredTopic, desiredPurpose)
		meta.SetExternalName(cr, channelID)

		obs, err := ext.Observe(context.Background(), cr)
		if err != nil {
			t.Fatalf("Observe returned unexpected error: %v", err)
		}

		if !obs.ResourceExists {
			t.Fatal("Observe returned ResourceExists=false, expected true")
		}

		// Compute expected up-to-date status
		expectedUpToDate := desiredName == remoteName &&
			ptrValueOrEmpty(desiredTopic) == remoteTopic &&
			ptrValueOrEmpty(desiredPurpose) == remotePurpose

		if obs.ResourceUpToDate != expectedUpToDate {
			t.Fatalf("ResourceUpToDate = %v, want %v (desired: name=%q topic=%v purpose=%v, remote: name=%q topic=%q purpose=%q)",
				obs.ResourceUpToDate, expectedUpToDate,
				desiredName, desiredTopic, desiredPurpose,
				remoteName, remoteTopic, remotePurpose)
		}
	})
}

// Feature: crossplane-provider-slack, Property 9: Conversation Update dispatches the correct API call for each changed field
// **Validates: Requirements 3.6, 3.7, 3.8**

func TestProperty_UpdateDispatchesCorrectAPICalls(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		desiredName := genChannelName().Draw(t, "desiredName")
		desiredTopic := genOptionalString().Draw(t, "desiredTopic")
		desiredPurpose := genOptionalString().Draw(t, "desiredPurpose")

		remoteName := genChannelName().Draw(t, "remoteName")
		remoteTopic := rapid.StringMatching(`[a-z]{0,250}`).Draw(t, "remoteTopic")
		remotePurpose := rapid.StringMatching(`[a-z]{0,250}`).Draw(t, "remotePurpose")

		channelID := genChannelID().Draw(t, "channelID")

		// Ensure at least one field differs
		if desiredName == remoteName &&
			ptrValueOrEmpty(desiredTopic) == remoteTopic &&
			ptrValueOrEmpty(desiredPurpose) == remotePurpose {
			// Force a difference in name
			desiredName = desiredName + "x"
			if len(desiredName) > 80 {
				desiredName = desiredName[:80]
			}
		}

		var renameCalled, topicCalled, purposeCalled bool

		mock := &mockClientAPI{
			getConversationInfoFn: func(_ context.Context, _ string) (*slack.Conversation, error) {
				return &slack.Conversation{
					ID:      channelID,
					Name:    remoteName,
					Topic:   slack.Topic{Value: remoteTopic},
					Purpose: slack.Topic{Value: remotePurpose},
				}, nil
			},
			renameConversationFn: func(_ context.Context, _, _ string) error {
				renameCalled = true
				return nil
			},
			setConversationTopicFn: func(_ context.Context, _, _ string) error {
				topicCalled = true
				return nil
			},
			setConversationPurposeFn: func(_ context.Context, _, _ string) error {
				purposeCalled = true
				return nil
			},
		}

		ext := &external{client: mock}
		cr := newConversationCR(desiredName, desiredTopic, desiredPurpose)
		meta.SetExternalName(cr, channelID)

		_, err := ext.Update(context.Background(), cr)
		if err != nil {
			t.Fatalf("Update returned unexpected error: %v", err)
		}

		// Verify rename called iff name differs
		nameChanged := desiredName != remoteName
		if renameCalled != nameChanged {
			t.Fatalf("RenameConversation called=%v, want %v (desired=%q, remote=%q)",
				renameCalled, nameChanged, desiredName, remoteName)
		}

		// Verify topic called iff topic differs
		topicChanged := ptrValueOrEmpty(desiredTopic) != remoteTopic
		if topicCalled != topicChanged {
			t.Fatalf("SetConversationTopic called=%v, want %v (desired=%v, remote=%q)",
				topicCalled, topicChanged, desiredTopic, remoteTopic)
		}

		// Verify purpose called iff purpose differs
		purposeChanged := ptrValueOrEmpty(desiredPurpose) != remotePurpose
		if purposeCalled != purposeChanged {
			t.Fatalf("SetConversationPurpose called=%v, want %v (desired=%v, remote=%q)",
				purposeCalled, purposeChanged, desiredPurpose, remotePurpose)
		}
	})
}

// Ensure external implements managed.ExternalClient at compile time.
var _ managed.ExternalClient = &external{}

// Ensure mockClientAPI implements slack.ClientAPI at compile time.
var _ slack.ClientAPI = &mockClientAPI{}

// Ensure Conversation implements resource.Managed at compile time.
var _ resource.Managed = &conversationv1alpha1.Conversation{}
