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

package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&ConversationBookmark{}, &ConversationBookmarkList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,slack}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status

// ConversationBookmark is the Schema for the ConversationBookmarks API.
type ConversationBookmark struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConversationBookmarkSpec   `json:"spec"`
	Status ConversationBookmarkStatus `json:"status,omitempty"`
}

// ConversationBookmarkSpec defines the desired state of ConversationBookmark.
type ConversationBookmarkSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ConversationBookmarkParameters `json:"forProvider"`
}

// ConversationBookmarkParameters defines the desired bookmark settings.
type ConversationBookmarkParameters struct {
	// ConversationID is the raw Slack channel ID. One of ConversationID or
	// ConversationRef is required.
	// +optional
	ConversationID *string `json:"conversationId,omitempty"`

	// ConversationRef references a Conversation resource to resolve the channel ID.
	// +optional
	ConversationRef *xpv1.Reference `json:"conversationRef,omitempty"`

	// ConversationSelector selects a Conversation resource.
	// +optional
	ConversationSelector *xpv1.Selector `json:"conversationSelector,omitempty"`

	// Title is the bookmark display title. Required.
	// +kubebuilder:validation:Required
	Title string `json:"title"`

	// Type is the bookmark type. Currently only "link" is supported.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=link
	Type string `json:"type"`

	// Link is the bookmark URL. Required.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	Link string `json:"link"`
}

// ConversationBookmarkStatus defines the observed state of ConversationBookmark.
type ConversationBookmarkStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ConversationBookmarkObservation `json:"atProvider,omitempty"`
}

// ConversationBookmarkObservation contains the observed state from Slack.
type ConversationBookmarkObservation struct {
	// ID is the Slack bookmark ID.
	ID string `json:"id,omitempty"`

	// ChannelID is the Slack channel ID the bookmark belongs to.
	ChannelID string `json:"channelId,omitempty"`

	// DateCreated is the Unix timestamp of bookmark creation.
	DateCreated int64 `json:"dateCreated,omitempty"`
}

// +kubebuilder:object:root=true

// ConversationBookmarkList contains a list of ConversationBookmark.
type ConversationBookmarkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConversationBookmark `json:"items"`
}
