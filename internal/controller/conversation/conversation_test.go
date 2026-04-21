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

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	conversationv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/conversation/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
)

func strPtr(s string) *string { return &s }

func TestObserve_ExistingChannel(t *testing.T) {
	mock := &mockClientAPI{
		getConversationInfoFn: func(_ context.Context, channelID string) (*slack.Conversation, error) {
			return &slack.Conversation{
				ID:      "C12345678",
				Name:    "my-channel",
				Topic:   slack.Topic{Value: "the topic"},
				Purpose: slack.Topic{Value: "the purpose"},
			}, nil
		},
	}

	ext := &external{client: mock}
	cr := newConversationCR("my-channel", strPtr("the topic"), strPtr("the purpose"))
	meta.SetExternalName(cr, "C12345678")

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
	if cr.Status.AtProvider.ID != "C12345678" {
		t.Fatalf("expected AtProvider.ID=C12345678, got %q", cr.Status.AtProvider.ID)
	}
}

func TestObserve_NonExistentChannel(t *testing.T) {
	mock := &mockClientAPI{
		getConversationInfoFn: func(_ context.Context, _ string) (*slack.Conversation, error) {
			return nil, &slack.SlackError{Code: "channel_not_found", Message: "channel not found"}
		},
	}

	ext := &external{client: mock}
	cr := newConversationCR("missing-channel", nil, nil)
	meta.SetExternalName(cr, "C99999999")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonNotFound {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonNotFound, cond.Reason)
	}
}

func TestObserve_NoExternalName(t *testing.T) {
	mock := &mockClientAPI{}
	ext := &external{client: mock}
	cr := newConversationCR("new-channel", nil, nil)

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false when no external-name")
	}
}

func TestCreate_Success(t *testing.T) {
	mock := &mockClientAPI{
		createConversationFn: func(_ context.Context, name string, _ bool) (*slack.Conversation, error) {
			return &slack.Conversation{ID: "CNEWCHAN01", Name: name}, nil
		},
	}

	ext := &external{client: mock}
	cr := newConversationCR("new-channel", nil, nil)

	_, err := ext.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	got := meta.GetExternalName(cr)
	if got != "CNEWCHAN01" {
		t.Fatalf("external-name = %q, want CNEWCHAN01", got)
	}
}

func TestCreate_NameTakenError(t *testing.T) {
	mock := &mockClientAPI{
		createConversationFn: func(_ context.Context, _ string, _ bool) (*slack.Conversation, error) {
			return nil, &slack.SlackError{Code: "name_taken", Message: "channel name is already taken"}
		},
	}

	ext := &external{client: mock}
	cr := newConversationCR("taken-channel", nil, nil)

	_, err := ext.Create(context.Background(), cr)
	if err == nil {
		t.Fatal("expected error from Create with name_taken")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonNameConflict {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonNameConflict, cond.Reason)
	}
	if cond.Status != "False" {
		t.Fatalf("expected Synced status=False, got %s", cond.Status)
	}
}

func TestUpdate_ChangedName(t *testing.T) {
	var renameCalled bool
	mock := &mockClientAPI{
		getConversationInfoFn: func(_ context.Context, _ string) (*slack.Conversation, error) {
			return &slack.Conversation{
				ID:      "C12345678",
				Name:    "old-name",
				Topic:   slack.Topic{Value: "topic"},
				Purpose: slack.Topic{Value: "purpose"},
			}, nil
		},
		renameConversationFn: func(_ context.Context, _, name string) error {
			renameCalled = true
			if name != "new-name" {
				t.Fatalf("rename called with %q, want new-name", name)
			}
			return nil
		},
	}

	ext := &external{client: mock}
	cr := newConversationCR("new-name", strPtr("topic"), strPtr("purpose"))
	meta.SetExternalName(cr, "C12345678")

	_, err := ext.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !renameCalled {
		t.Fatal("expected RenameConversation to be called")
	}
}

func TestUpdate_ChangedTopic(t *testing.T) {
	var topicCalled bool
	mock := &mockClientAPI{
		getConversationInfoFn: func(_ context.Context, _ string) (*slack.Conversation, error) {
			return &slack.Conversation{
				ID:      "C12345678",
				Name:    "my-channel",
				Topic:   slack.Topic{Value: "old-topic"},
				Purpose: slack.Topic{Value: "purpose"},
			}, nil
		},
		setConversationTopicFn: func(_ context.Context, _, topic string) error {
			topicCalled = true
			if topic != "new-topic" {
				t.Fatalf("setTopic called with %q, want new-topic", topic)
			}
			return nil
		},
	}

	ext := &external{client: mock}
	cr := newConversationCR("my-channel", strPtr("new-topic"), strPtr("purpose"))
	meta.SetExternalName(cr, "C12345678")

	_, err := ext.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !topicCalled {
		t.Fatal("expected SetConversationTopic to be called")
	}
}

func TestDelete_Success(t *testing.T) {
	var archiveCalled bool
	mock := &mockClientAPI{
		archiveConversationFn: func(_ context.Context, channelID string) error {
			archiveCalled = true
			if channelID != "C12345678" {
				t.Fatalf("archive called with %q, want C12345678", channelID)
			}
			return nil
		},
	}

	ext := &external{client: mock}
	cr := newConversationCR("my-channel", nil, nil)
	meta.SetExternalName(cr, "C12345678")

	_, err := ext.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !archiveCalled {
		t.Fatal("expected ArchiveConversation to be called")
	}
}

func TestDelete_ChannelNotFound(t *testing.T) {
	mock := &mockClientAPI{
		archiveConversationFn: func(_ context.Context, _ string) error {
			return &slack.SlackError{Code: "channel_not_found", Message: "channel not found"}
		},
	}

	ext := &external{client: mock}
	cr := newConversationCR("gone-channel", nil, nil)
	meta.SetExternalName(cr, "C99999999")

	_, err := ext.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete should not return error for channel_not_found, got: %v", err)
	}
}

func TestObserve_DriftDetection(t *testing.T) {
	tests := []struct {
		name         string
		desiredName  string
		desiredTopic *string
		remoteName   string
		remoteTopic  string
		wantUpToDate bool
	}{
		{
			name:         "all match",
			desiredName:  "channel",
			desiredTopic: strPtr("topic"),
			remoteName:   "channel",
			remoteTopic:  "topic",
			wantUpToDate: true,
		},
		{
			name:         "name differs",
			desiredName:  "new-name",
			desiredTopic: strPtr("topic"),
			remoteName:   "old-name",
			remoteTopic:  "topic",
			wantUpToDate: false,
		},
		{
			name:         "topic differs",
			desiredName:  "channel",
			desiredTopic: strPtr("new-topic"),
			remoteName:   "channel",
			remoteTopic:  "old-topic",
			wantUpToDate: false,
		},
		{
			name:         "nil topic matches empty remote",
			desiredName:  "channel",
			desiredTopic: nil,
			remoteName:   "channel",
			remoteTopic:  "",
			wantUpToDate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClientAPI{
				getConversationInfoFn: func(_ context.Context, _ string) (*slack.Conversation, error) {
					return &slack.Conversation{
						ID:      "C12345678",
						Name:    tt.remoteName,
						Topic:   slack.Topic{Value: tt.remoteTopic},
						Purpose: slack.Topic{Value: ""},
					}, nil
				},
			}

			ext := &external{client: mock}
			cr := &conversationv1alpha1.Conversation{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "conversation.slack.crossplane.io/v1alpha1",
					Kind:       "Conversation",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{},
				},
				Spec: conversationv1alpha1.ConversationSpec{
					ForProvider: conversationv1alpha1.ConversationParameters{
						Name:  tt.desiredName,
						Topic: tt.desiredTopic,
					},
				},
			}
			meta.SetExternalName(cr, "C12345678")

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
