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

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	usergroupv1alpha1 "github.com/avodah-inc/crossplane-provider-slack/apis/usergroup/v1alpha1"
)

// UserGroupExternalName resolves the external-name from a UserGroup resource.
func UserGroupExternalName() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		return meta.GetExternalName(mg)
	}
}

// ResolveUserGroupID resolves the user group ID from the spec fields.
// It returns the raw UserGroupID if set, otherwise resolves via the reference.
func ResolveUserGroupID(ctx context.Context, reader client.Reader, params *UserGroupMembersParameters) (string, error) {
	if params.UserGroupID != nil && *params.UserGroupID != "" {
		return *params.UserGroupID, nil
	}

	if params.UserGroupRef != nil && params.UserGroupRef.Name != "" {
		ug := &usergroupv1alpha1.UserGroup{}
		if err := reader.Get(ctx, client.ObjectKey{Name: params.UserGroupRef.Name}, ug); err != nil {
			return "", err
		}
		return meta.GetExternalName(ug), nil
	}

	return "", nil
}
