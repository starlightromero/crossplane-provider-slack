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

// Package v1alpha1 contains the v1alpha1 group ConversationMember resources of the
// Slack provider.
// +kubebuilder:object:generate=true
// +groupName=conversationmember.slack.crossplane.io
// +versionName=v1alpha1
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "conversationmember.slack.crossplane.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionResource scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// ConversationMember type metadata.
var (
	// ConversationMemberKind is the kind of the ConversationMember resource.
	ConversationMemberKind = "ConversationMember"

	// ConversationMemberGroupKind is the group-kind of the ConversationMember resource.
	ConversationMemberGroupKind = schema.GroupKind{Group: Group, Kind: ConversationMemberKind}.String()

	// ConversationMemberGroupVersionKind is the GVK of the ConversationMember resource.
	ConversationMemberGroupVersionKind = SchemeGroupVersion.WithKind(ConversationMemberKind)
)
