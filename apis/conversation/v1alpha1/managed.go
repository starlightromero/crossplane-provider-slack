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
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
)

// Ensure Conversation satisfies the resource.Managed interface.
var _ resource.Managed = &Conversation{}

// GetCondition returns the condition for the given ConditionType.
func (c *Conversation) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return c.Status.GetCondition(ct)
}

// SetConditions sets the supplied conditions on the resource.
func (c *Conversation) SetConditions(conditions ...xpv1.Condition) {
	c.Status.SetConditions(conditions...)
}

// GetProviderConfigReference returns the provider config reference.
func (c *Conversation) GetProviderConfigReference() *xpv1.Reference {
	return c.Spec.ProviderConfigReference
}

// SetProviderConfigReference sets the provider config reference.
func (c *Conversation) SetProviderConfigReference(ref *xpv1.Reference) {
	c.Spec.ProviderConfigReference = ref
}

// GetWriteConnectionSecretToReference returns the connection secret reference.
func (c *Conversation) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return c.Spec.WriteConnectionSecretToReference
}

// SetWriteConnectionSecretToReference sets the connection secret reference.
func (c *Conversation) SetWriteConnectionSecretToReference(ref *xpv1.SecretReference) {
	c.Spec.WriteConnectionSecretToReference = ref
}

// GetManagementPolicies returns the management policies.
func (c *Conversation) GetManagementPolicies() xpv1.ManagementPolicies {
	return c.Spec.ManagementPolicies
}

// SetManagementPolicies sets the management policies.
func (c *Conversation) SetManagementPolicies(p xpv1.ManagementPolicies) {
	c.Spec.ManagementPolicies = p
}

// GetDeletionPolicy returns the deletion policy.
func (c *Conversation) GetDeletionPolicy() xpv1.DeletionPolicy {
	return c.Spec.DeletionPolicy
}

// SetDeletionPolicy sets the deletion policy.
func (c *Conversation) SetDeletionPolicy(p xpv1.DeletionPolicy) {
	c.Spec.DeletionPolicy = p
}
