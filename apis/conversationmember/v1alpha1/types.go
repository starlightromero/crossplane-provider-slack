/*
Copyright 2026 Starlight Romero.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&ConversationMember{}, &ConversationMemberList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,slack}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status

// ConversationMember is the Schema for the ConversationMembers API.
type ConversationMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConversationMemberSpec   `json:"spec"`
	Status ConversationMemberStatus `json:"status,omitempty"`
}

// ConversationMemberSpec defines the desired state of ConversationMember.
type ConversationMemberSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ConversationMemberParameters `json:"forProvider"`
}

// ConversationMemberParameters defines the desired membership settings.
type ConversationMemberParameters struct {
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

	// UserEmail is the email address of the user to invite. One of UserEmail
	// or UserID is required.
	// +optional
	UserEmail *string `json:"userEmail,omitempty"`

	// UserID is the raw Slack user ID. One of UserEmail or UserID is required.
	// +optional
	UserID *string `json:"userId,omitempty"`
}

// ConversationMemberStatus defines the observed state of ConversationMember.
type ConversationMemberStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ConversationMemberObservation `json:"atProvider,omitempty"`
}

// ConversationMemberObservation contains the observed state from Slack.
type ConversationMemberObservation struct {
	// ResolvedUserID is the Slack user ID resolved from the email.
	ResolvedUserID string `json:"resolvedUserId,omitempty"`

	// ChannelID is the Slack channel ID the user is a member of.
	ChannelID string `json:"channelId,omitempty"`
}

// +kubebuilder:object:root=true

// ConversationMemberList contains a list of ConversationMember.
type ConversationMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConversationMember `json:"items"`
}
