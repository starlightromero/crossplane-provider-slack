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

package pin

import (
	"context"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pinv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/pin/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
)

// mockClientAPI implements slack.ClientAPI for testing pin controller.
type mockClientAPI struct {
	addPinFn    func(ctx context.Context, channelID, messageTS string) error
	listPinsFn  func(ctx context.Context, channelID string) ([]slack.Pin, error)
	removePinFn func(ctx context.Context, channelID, messageTS string) error
}

func (m *mockClientAPI) AddPin(ctx context.Context, channelID, messageTS string) error {
	if m.addPinFn != nil {
		return m.addPinFn(ctx, channelID, messageTS)
	}
	return nil
}

func (m *mockClientAPI) ListPins(ctx context.Context, channelID string) ([]slack.Pin, error) {
	if m.listPinsFn != nil {
		return m.listPinsFn(ctx, channelID)
	}
	return nil, nil
}

func (m *mockClientAPI) RemovePin(ctx context.Context, channelID, messageTS string) error {
	if m.removePinFn != nil {
		return m.removePinFn(ctx, channelID, messageTS)
	}
	return nil
}

// Stub implementations for non-pin methods.
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

// Ensure mockClientAPI implements slack.ClientAPI at compile time.
var _ slack.ClientAPI = &mockClientAPI{}

// Ensure external implements managed.ExternalClient at compile time.
var _ managed.ExternalClient = &external{}

// Ensure ConversationPin implements resource.Managed at compile time.
var _ resource.Managed = &pinv1alpha1.ConversationPin{}

func newPinCR(channelID, messageTS string) *pinv1alpha1.ConversationPin {
	return &pinv1alpha1.ConversationPin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "pin.slack.crossplane.io/v1alpha1",
			Kind:       "ConversationPin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pin",
			Annotations: map[string]string{},
		},
		Spec: pinv1alpha1.ConversationPinSpec{
			ForProvider: pinv1alpha1.ConversationPinParameters{
				ConversationID:   &channelID,
				MessageTimestamp: messageTS,
			},
		},
	}
}

func TestObserve_PinExists(t *testing.T) {
	channelID := "C12345678"
	messageTS := "1234567890.123456"

	mock := &mockClientAPI{
		listPinsFn: func(_ context.Context, _ string) ([]slack.Pin, error) {
			return []slack.Pin{
				{Channel: channelID, Message: slack.Message{Ts: messageTS}, Created: 1700000000},
			}, nil
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newPinCR(channelID, messageTS)
	meta.SetExternalName(cr, formatExternalName(channelID, messageTS))

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=true (pins are immutable)")
	}
	if cr.Status.AtProvider.ChannelID != channelID {
		t.Fatalf("expected AtProvider.ChannelID=%s, got %q", channelID, cr.Status.AtProvider.ChannelID)
	}
	if cr.Status.AtProvider.PinnedAt != 1700000000 {
		t.Fatalf("expected AtProvider.PinnedAt=1700000000, got %d", cr.Status.AtProvider.PinnedAt)
	}
}

func TestObserve_PinNotFound(t *testing.T) {
	channelID := "C12345678"
	messageTS := "1234567890.123456"

	mock := &mockClientAPI{
		listPinsFn: func(_ context.Context, _ string) ([]slack.Pin, error) {
			return []slack.Pin{}, nil
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newPinCR(channelID, messageTS)
	meta.SetExternalName(cr, formatExternalName(channelID, messageTS))

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false")
	}
}

func TestObserve_NoExternalName(t *testing.T) {
	mock := &mockClientAPI{}
	ext := &external{client: mock, kube: nil}
	cr := newPinCR("C12345678", "1234567890.123456")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false when no external-name")
	}
}

func TestCreate_Success(t *testing.T) {
	channelID := "C12345678"
	messageTS := "1234567890.123456"
	var addCalled bool

	mock := &mockClientAPI{
		addPinFn: func(_ context.Context, ch, ts string) error {
			addCalled = true
			if ch != channelID {
				t.Fatalf("AddPin channelID = %q, want %q", ch, channelID)
			}
			if ts != messageTS {
				t.Fatalf("AddPin messageTS = %q, want %q", ts, messageTS)
			}
			return nil
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newPinCR(channelID, messageTS)

	_, err := ext.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if !addCalled {
		t.Fatal("expected AddPin to be called")
	}

	expectedExtName := formatExternalName(channelID, messageTS)
	got := meta.GetExternalName(cr)
	if got != expectedExtName {
		t.Fatalf("external-name = %q, want %q", got, expectedExtName)
	}
}

func TestDelete_Success(t *testing.T) {
	channelID := "C12345678"
	messageTS := "1234567890.123456"
	var removeCalled bool

	mock := &mockClientAPI{
		removePinFn: func(_ context.Context, ch, ts string) error {
			removeCalled = true
			if ch != channelID {
				t.Fatalf("RemovePin channelID = %q, want %q", ch, channelID)
			}
			if ts != messageTS {
				t.Fatalf("RemovePin messageTS = %q, want %q", ts, messageTS)
			}
			return nil
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newPinCR(channelID, messageTS)
	meta.SetExternalName(cr, formatExternalName(channelID, messageTS))

	_, err := ext.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !removeCalled {
		t.Fatal("expected RemovePin to be called")
	}
}

func TestObserve_ChannelNotFound(t *testing.T) {
	channelID := "C12345678"
	messageTS := "1234567890.123456"

	mock := &mockClientAPI{
		listPinsFn: func(_ context.Context, _ string) ([]slack.Pin, error) {
			return nil, &slack.SlackError{Code: "channel_not_found", Message: "channel not found"}
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newPinCR(channelID, messageTS)
	meta.SetExternalName(cr, formatExternalName(channelID, messageTS))

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false for channel_not_found")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonChannelUnavailable {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonChannelUnavailable, cond.Reason)
	}
}

func TestObserve_IsArchived(t *testing.T) {
	channelID := "C12345678"
	messageTS := "1234567890.123456"

	mock := &mockClientAPI{
		listPinsFn: func(_ context.Context, _ string) ([]slack.Pin, error) {
			return nil, &slack.SlackError{Code: "is_archived", Message: "channel is archived"}
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newPinCR(channelID, messageTS)
	meta.SetExternalName(cr, formatExternalName(channelID, messageTS))

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false for is_archived")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonChannelUnavailable {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonChannelUnavailable, cond.Reason)
	}
}

func TestCreate_MessageNotFound(t *testing.T) {
	channelID := "C12345678"
	messageTS := "1234567890.123456"

	mock := &mockClientAPI{
		addPinFn: func(_ context.Context, _, _ string) error {
			return &slack.SlackError{Code: "message_not_found", Message: "message not found"}
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newPinCR(channelID, messageTS)

	_, err := ext.Create(context.Background(), cr)
	if err == nil {
		t.Fatal("expected error from Create with message_not_found")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonMessageNotFound {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonMessageNotFound, cond.Reason)
	}
}

func TestCreate_ChannelNotFound(t *testing.T) {
	channelID := "C12345678"
	messageTS := "1234567890.123456"

	mock := &mockClientAPI{
		addPinFn: func(_ context.Context, _, _ string) error {
			return &slack.SlackError{Code: "channel_not_found", Message: "channel not found"}
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newPinCR(channelID, messageTS)

	_, err := ext.Create(context.Background(), cr)
	if err == nil {
		t.Fatal("expected error from Create with channel_not_found")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonChannelUnavailable {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonChannelUnavailable, cond.Reason)
	}
}

func TestDelete_ChannelNotFound(t *testing.T) {
	channelID := "C12345678"
	messageTS := "1234567890.123456"

	mock := &mockClientAPI{
		removePinFn: func(_ context.Context, _, _ string) error {
			return &slack.SlackError{Code: "channel_not_found", Message: "channel not found"}
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newPinCR(channelID, messageTS)
	meta.SetExternalName(cr, formatExternalName(channelID, messageTS))

	_, err := ext.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete should not return error for channel_not_found, got: %v", err)
	}
}
