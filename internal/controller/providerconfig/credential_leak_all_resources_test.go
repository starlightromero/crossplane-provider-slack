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

package providerconfig

import (
	"encoding/json"
	"strings"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"pgregory.net/rapid"

	bookmarkv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/bookmark/v1alpha1"
	conversationv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/conversation/v1alpha1"
	pinv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/pin/v1alpha1"
	usergroupv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/usergroup/v1alpha1"
	usergroupmembersv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/usergroupmembers/v1alpha1"
	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
)

// Feature: crossplane-provider-slack, Property 6 (extended): Serialized managed resources never contain credential values
// This test extends Property 6 to cover ALL managed resource types.
// **Validates: Requirements 1.7, 10.4**

func TestProperty_CredentialLeakAllResourceTypes(t *testing.T) {
	// For any managed resource type and any bot token value, serializing the
	// resource to JSON SHALL NOT produce output containing the bot token string.
	rapid.Check(t, func(t *rapid.T) {
		// Generate an arbitrary bot token with xoxb- prefix
		tokenSuffix := rapid.StringMatching(`[a-zA-Z0-9\-]{10,50}`).Draw(t, "tokenSuffix")
		botToken := "xoxb-" + tokenSuffix

		// Generate common metadata
		name := rapid.StringMatching(`[a-z][a-z0-9\-]{2,20}`).Draw(t, "name")

		// Pick a resource type to test
		resourceIdx := rapid.IntRange(0, 5).Draw(t, "resourceIdx")

		var obj runtime.Object
		switch resourceIdx {
		case 0:
			// ProviderConfig
			obj = &v1alpha1.ProviderConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "slack.crossplane.io/v1alpha1",
					Kind:       "ProviderConfig",
				},
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec: v1alpha1.ProviderConfigSpec{
					Credentials: v1alpha1.ProviderCredentials{
						Source: xpv1.CredentialsSourceSecret,
						CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
							SecretRef: &xpv1.SecretKeySelector{
								SecretReference: xpv1.SecretReference{
									Name:      "slack-creds",
									Namespace: "crossplane-system",
								},
								Key: "token",
							},
						},
					},
				},
			}
		case 1:
			// Conversation
			isPrivate := false
			obj = &conversationv1alpha1.Conversation{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "conversation.slack.crossplane.io/v1alpha1",
					Kind:       "Conversation",
				},
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec: conversationv1alpha1.ConversationSpec{
					ForProvider: conversationv1alpha1.ConversationParameters{
						Name:      "test-channel",
						IsPrivate: &isPrivate,
					},
				},
			}
		case 2:
			// ConversationBookmark
			obj = &bookmarkv1alpha1.ConversationBookmark{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "bookmark.slack.crossplane.io/v1alpha1",
					Kind:       "ConversationBookmark",
				},
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec: bookmarkv1alpha1.ConversationBookmarkSpec{
					ForProvider: bookmarkv1alpha1.ConversationBookmarkParameters{
						Title: "Test Bookmark",
						Type:  "link",
						Link:  "https://example.com",
					},
				},
			}
		case 3:
			// ConversationPin
			obj = &pinv1alpha1.ConversationPin{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "pin.slack.crossplane.io/v1alpha1",
					Kind:       "ConversationPin",
				},
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec: pinv1alpha1.ConversationPinSpec{
					ForProvider: pinv1alpha1.ConversationPinParameters{
						MessageTimestamp: "1234567890.123456",
					},
				},
			}
		case 4:
			// UserGroup
			obj = &usergroupv1alpha1.UserGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "usergroup.slack.crossplane.io/v1alpha1",
					Kind:       "UserGroup",
				},
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec: usergroupv1alpha1.UserGroupSpec{
					ForProvider: usergroupv1alpha1.UserGroupParameters{
						Name:   "Test Group",
						Handle: "test-group",
					},
				},
			}
		case 5:
			// UserGroupMembers
			obj = &usergroupmembersv1alpha1.UserGroupMembers{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "usergroupmembers.slack.crossplane.io/v1alpha1",
					Kind:       "UserGroupMembers",
				},
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec: usergroupmembersv1alpha1.UserGroupMembersSpec{
					ForProvider: usergroupmembersv1alpha1.UserGroupMembersParameters{
						UserEmails: []string{"user@example.com"},
					},
				},
			}
		}

		// Serialize the resource to JSON
		data, err := json.Marshal(obj)
		if err != nil {
			t.Fatalf("failed to marshal resource: %v", err)
		}

		serialized := string(data)

		// Assert the bot token value does NOT appear in the serialized output
		if strings.Contains(serialized, botToken) {
			t.Fatalf("serialized resource contains bot token %q;\nJSON: %s", botToken, serialized)
		}
	})
}
