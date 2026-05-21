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
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
)

// Ensure ConversationMember satisfies the resource.Managed interface.
var _ resource.Managed = &ConversationMember{}

// GetCondition returns the condition for the given ConditionType.
func (c *ConversationMember) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return c.Status.GetCondition(ct)
}

// SetConditions sets the supplied conditions on the resource.
func (c *ConversationMember) SetConditions(conditions ...xpv1.Condition) {
	c.Status.SetConditions(conditions...)
}

// GetProviderConfigReference returns the provider config reference.
func (c *ConversationMember) GetProviderConfigReference() *xpv1.Reference {
	return c.Spec.ProviderConfigReference
}

// SetProviderConfigReference sets the provider config reference.
func (c *ConversationMember) SetProviderConfigReference(ref *xpv1.Reference) {
	c.Spec.ProviderConfigReference = ref
}

// GetWriteConnectionSecretToReference returns the connection secret reference.
func (c *ConversationMember) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return c.Spec.WriteConnectionSecretToReference
}

// SetWriteConnectionSecretToReference sets the connection secret reference.
func (c *ConversationMember) SetWriteConnectionSecretToReference(ref *xpv1.SecretReference) {
	c.Spec.WriteConnectionSecretToReference = ref
}

// GetManagementPolicies returns the management policies.
func (c *ConversationMember) GetManagementPolicies() xpv1.ManagementPolicies {
	return c.Spec.ManagementPolicies
}

// SetManagementPolicies sets the management policies.
func (c *ConversationMember) SetManagementPolicies(p xpv1.ManagementPolicies) {
	c.Spec.ManagementPolicies = p
}

// GetDeletionPolicy returns the deletion policy.
func (c *ConversationMember) GetDeletionPolicy() xpv1.DeletionPolicy {
	return c.Spec.DeletionPolicy
}

// SetDeletionPolicy sets the deletion policy.
func (c *ConversationMember) SetDeletionPolicy(p xpv1.DeletionPolicy) {
	c.Spec.DeletionPolicy = p
}
