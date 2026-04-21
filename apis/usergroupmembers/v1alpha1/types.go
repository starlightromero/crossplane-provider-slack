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
	SchemeBuilder.Register(&UserGroupMembers{}, &UserGroupMembersList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,slack}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status

// UserGroupMembers is the Schema for the UserGroupMembers API.
type UserGroupMembers struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserGroupMembersSpec   `json:"spec"`
	Status UserGroupMembersStatus `json:"status,omitempty"`
}

// UserGroupMembersSpec defines the desired state of UserGroupMembers.
type UserGroupMembersSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       UserGroupMembersParameters `json:"forProvider"`
}

// UserGroupMembersParameters defines the desired membership settings.
type UserGroupMembersParameters struct {
	// UserGroupID is the raw Slack user group ID. One of UserGroupID or
	// UserGroupRef is required.
	// +optional
	UserGroupID *string `json:"userGroupId,omitempty"`

	// UserGroupRef references a UserGroup resource to resolve the group ID.
	// +optional
	UserGroupRef *xpv1.Reference `json:"userGroupRef,omitempty"`

	// UserGroupSelector selects a UserGroup resource.
	// +optional
	UserGroupSelector *xpv1.Selector `json:"userGroupSelector,omitempty"`

	// UserEmails is the list of user email addresses that should be members
	// of the user group. Required, minimum 1 item.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	UserEmails []string `json:"userEmails"`
}

// UserGroupMembersStatus defines the observed state of UserGroupMembers.
type UserGroupMembersStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserGroupMembersObservation `json:"atProvider,omitempty"`
}

// UserGroupMembersObservation contains the observed state from Slack.
type UserGroupMembersObservation struct {
	// ResolvedUserIds is the list of Slack user IDs resolved from the emails.
	ResolvedUserIds []string `json:"resolvedUserIds,omitempty"`

	// MemberCount is the number of members currently in the user group.
	MemberCount int `json:"memberCount,omitempty"`
}

// +kubebuilder:object:root=true

// UserGroupMembersList contains a list of UserGroupMembers.
type UserGroupMembersList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserGroupMembers `json:"items"`
}
