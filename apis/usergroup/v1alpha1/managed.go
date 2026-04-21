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

// Ensure UserGroup satisfies the resource.Managed interface.
var _ resource.Managed = &UserGroup{}

// GetCondition returns the condition for the given ConditionType.
func (u *UserGroup) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return u.Status.GetCondition(ct)
}

// SetConditions sets the supplied conditions on the resource.
func (u *UserGroup) SetConditions(conditions ...xpv1.Condition) {
	u.Status.SetConditions(conditions...)
}

// GetProviderConfigReference returns the provider config reference.
func (u *UserGroup) GetProviderConfigReference() *xpv1.Reference {
	return u.Spec.ProviderConfigReference
}

// SetProviderConfigReference sets the provider config reference.
func (u *UserGroup) SetProviderConfigReference(ref *xpv1.Reference) {
	u.Spec.ProviderConfigReference = ref
}

// GetWriteConnectionSecretToReference returns the connection secret reference.
func (u *UserGroup) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return u.Spec.WriteConnectionSecretToReference
}

// SetWriteConnectionSecretToReference sets the connection secret reference.
func (u *UserGroup) SetWriteConnectionSecretToReference(ref *xpv1.SecretReference) {
	u.Spec.WriteConnectionSecretToReference = ref
}

// GetManagementPolicies returns the management policies.
func (u *UserGroup) GetManagementPolicies() xpv1.ManagementPolicies {
	return u.Spec.ManagementPolicies
}

// SetManagementPolicies sets the management policies.
func (u *UserGroup) SetManagementPolicies(p xpv1.ManagementPolicies) {
	u.Spec.ManagementPolicies = p
}

// GetDeletionPolicy returns the deletion policy.
func (u *UserGroup) GetDeletionPolicy() xpv1.DeletionPolicy {
	return u.Spec.DeletionPolicy
}

// SetDeletionPolicy sets the deletion policy.
func (u *UserGroup) SetDeletionPolicy(p xpv1.DeletionPolicy) {
	u.Spec.DeletionPolicy = p
}
