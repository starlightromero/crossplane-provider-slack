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
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
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
func ResolveConversationID(ctx context.Context, reader client.Reader, params *ConversationMemberParameters) (string, error) {
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
