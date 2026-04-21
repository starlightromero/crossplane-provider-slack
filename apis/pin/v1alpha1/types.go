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
	SchemeBuilder.Register(&ConversationPin{}, &ConversationPinList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,slack}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status

// ConversationPin is the Schema for the ConversationPins API.
type ConversationPin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConversationPinSpec   `json:"spec"`
	Status ConversationPinStatus `json:"status,omitempty"`
}

// ConversationPinSpec defines the desired state of ConversationPin.
type ConversationPinSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ConversationPinParameters `json:"forProvider"`
}

// ConversationPinParameters defines the desired pin settings.
type ConversationPinParameters struct {
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

	// MessageTimestamp is the Slack message timestamp to pin.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^\d+\.\d+$`
	MessageTimestamp string `json:"messageTimestamp"`
}

// ConversationPinStatus defines the observed state of ConversationPin.
type ConversationPinStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ConversationPinObservation `json:"atProvider,omitempty"`
}

// ConversationPinObservation contains the observed state from Slack.
type ConversationPinObservation struct {
	// ChannelID is the Slack channel ID the pin belongs to.
	ChannelID string `json:"channelId,omitempty"`

	// PinnedAt is the Unix timestamp of when the message was pinned.
	PinnedAt int64 `json:"pinnedAt,omitempty"`
}

// +kubebuilder:object:root=true

// ConversationPinList contains a list of ConversationPin.
type ConversationPinList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConversationPin `json:"items"`
}
