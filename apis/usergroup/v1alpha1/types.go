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
	SchemeBuilder.Register(&UserGroup{}, &UserGroupList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,slack}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status

// UserGroup is the Schema for the UserGroups API.
type UserGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserGroupSpec   `json:"spec"`
	Status UserGroupStatus `json:"status,omitempty"`
}

// UserGroupSpec defines the desired state of UserGroup.
type UserGroupSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       UserGroupParameters `json:"forProvider"`
}

// UserGroupParameters defines the desired Slack user group settings.
type UserGroupParameters struct {
	// Name is the user group name. Required.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Handle is the user group mention handle. Required.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9][a-z0-9._-]*$`
	Handle string `json:"handle"`

	// Description is an optional description of the user group.
	// +optional
	Description *string `json:"description,omitempty"`
}

// UserGroupStatus defines the observed state of UserGroup.
type UserGroupStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserGroupObservation `json:"atProvider,omitempty"`
}

// UserGroupObservation contains the observed state from Slack.
type UserGroupObservation struct {
	// ID is the Slack user group ID.
	ID string `json:"id,omitempty"`

	// IsEnabled indicates whether the user group is enabled.
	IsEnabled bool `json:"isEnabled,omitempty"`

	// CreatedBy is the user ID of the creator.
	CreatedBy string `json:"createdBy,omitempty"`

	// DateCreate is the Unix timestamp of user group creation.
	DateCreate int64 `json:"dateCreate,omitempty"`
}

// +kubebuilder:object:root=true

// UserGroupList contains a list of UserGroup.
type UserGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserGroup `json:"items"`
}
