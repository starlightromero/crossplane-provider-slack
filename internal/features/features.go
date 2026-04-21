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

// Package features defines feature flag constants for the crossplane-provider-slack controllers.
package features

import "github.com/crossplane/crossplane-runtime/pkg/feature"

// Feature flags for managed resource controllers. Each flag gates the
// registration of the corresponding controller, allowing operators to
// enable only the resource types they need.
const (
	// EnableAlphaConversation enables the Conversation managed resource controller.
	EnableAlphaConversation feature.Flag = "EnableAlphaConversation"

	// EnableAlphaConversationBookmark enables the ConversationBookmark managed resource controller.
	EnableAlphaConversationBookmark feature.Flag = "EnableAlphaConversationBookmark"

	// EnableAlphaConversationPin enables the ConversationPin managed resource controller.
	EnableAlphaConversationPin feature.Flag = "EnableAlphaConversationPin"

	// EnableAlphaUserGroup enables the UserGroup managed resource controller.
	EnableAlphaUserGroup feature.Flag = "EnableAlphaUserGroup"

	// EnableAlphaUserGroupMembers enables the UserGroupMembers managed resource controller.
	EnableAlphaUserGroupMembers feature.Flag = "EnableAlphaUserGroupMembers"
)
