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

// ProviderConfig interface implementations.
// These methods satisfy the resource.ProviderConfig interface from
// crossplane-runtime, enabling the standard providerconfig reconciler.

var _ resource.ProviderConfig = &ProviderConfig{}
var _ resource.LegacyProviderConfigUsage = &ProviderConfigUsage{}
var _ resource.ProviderConfigUsageList = &ProviderConfigUsageList{}

// SetConditions sets the supplied conditions on the ProviderConfig.
func (p *ProviderConfig) SetConditions(c ...xpv1.Condition) {
	p.Status.SetConditions(c...)
}

// GetCondition returns the condition for the given ConditionType.
func (p *ProviderConfig) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return p.Status.GetCondition(ct)
}

// SetUsers sets the number of users of this ProviderConfig.
func (p *ProviderConfig) SetUsers(i int64) {
	p.Status.Users = i
}

// GetUsers returns the number of users of this ProviderConfig.
func (p *ProviderConfig) GetUsers() int64 {
	return p.Status.Users
}

// GetProviderConfigReference returns the provider config reference.
func (p *ProviderConfigUsage) GetProviderConfigReference() xpv1.Reference {
	return p.ProviderConfigUsage.ProviderConfigReference
}

// SetProviderConfigReference sets the provider config reference.
func (p *ProviderConfigUsage) SetProviderConfigReference(ref xpv1.Reference) {
	p.ProviderConfigUsage.ProviderConfigReference = ref
}

// GetResourceReference returns the resource reference.
func (p *ProviderConfigUsage) GetResourceReference() xpv1.TypedReference {
	return p.ProviderConfigUsage.ResourceReference
}

// SetResourceReference sets the resource reference.
func (p *ProviderConfigUsage) SetResourceReference(ref xpv1.TypedReference) {
	p.ProviderConfigUsage.ResourceReference = ref
}

// GetItems returns the list of ProviderConfigUsage items as the interface type.
func (p *ProviderConfigUsageList) GetItems() []resource.ProviderConfigUsage {
	items := make([]resource.ProviderConfigUsage, len(p.Items))
	for i := range p.Items {
		items[i] = &p.Items[i]
	}
	return items
}
