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
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Conversation{}, &ConversationList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,slack}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status

// Conversation is the Schema for the Conversations API.
type Conversation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConversationSpec   `json:"spec"`
	Status ConversationStatus `json:"status,omitempty"`
}

// ConversationSpec defines the desired state of Conversation.
type ConversationSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ConversationParameters `json:"forProvider"`
}

// ConversationParameters defines the desired Slack conversation settings.
type ConversationParameters struct {
	// Name is the channel name. Required.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=80
	Name string `json:"name"`

	// IsPrivate determines if the channel is private. Default: false.
	// +optional
	// +kubebuilder:default=false
	IsPrivate *bool `json:"isPrivate,omitempty"`

	// Topic is the channel topic.
	// +optional
	// +kubebuilder:validation:MaxLength=250
	Topic *string `json:"topic,omitempty"`

	// Purpose is the channel purpose.
	// +optional
	// +kubebuilder:validation:MaxLength=250
	Purpose *string `json:"purpose,omitempty"`
}

// ConversationStatus defines the observed state of Conversation.
type ConversationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ConversationObservation `json:"atProvider,omitempty"`
}

// ConversationObservation contains the observed state from Slack.
type ConversationObservation struct {
	// ID is the Slack channel ID.
	ID string `json:"id,omitempty"`

	// IsArchived indicates whether the channel is archived.
	IsArchived bool `json:"isArchived,omitempty"`

	// NumMembers is the number of members in the channel.
	NumMembers int `json:"numMembers,omitempty"`

	// Created is the Unix timestamp of channel creation.
	Created int64 `json:"created,omitempty"`
}

// +kubebuilder:object:root=true

// ConversationList contains a list of Conversation.
type ConversationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Conversation `json:"items"`
}
