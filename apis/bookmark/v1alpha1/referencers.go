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
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	conversationv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/conversation/v1alpha1"
)

// ConversationExternalName resolves the external-name from a Conversation resource.
func ConversationExternalName() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		return meta.GetExternalName(mg)
	}
}

// ResolveConversationID resolves the conversation ID from the spec fields.
// It returns the raw ConversationID if set, otherwise resolves via the reference.
func ResolveConversationID(ctx context.Context, reader client.Reader, params *ConversationBookmarkParameters) (string, error) {
	if params.ConversationID != nil && *params.ConversationID != "" {
		return *params.ConversationID, nil
	}

	if params.ConversationRef != nil && params.ConversationRef.Name != "" {
		conv := &conversationv1alpha1.Conversation{}
		if err := reader.Get(ctx, client.ObjectKey{Name: params.ConversationRef.Name}, conv); err != nil {
			return "", err
		}
		return meta.GetExternalName(conv), nil
	}

	return "", nil
}
