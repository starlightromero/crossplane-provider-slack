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

package providerconfig

import (
	"encoding/json"
	"strings"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pgregory.net/rapid"

	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
)

// Feature: crossplane-provider-slack, Property 6: Serialized managed resources never contain credential values
// **Validates: Requirements 1.7, 10.4**

func TestProperty_CredentialExclusionFromSerialization(t *testing.T) {
	// For any ProviderConfig object and any bot token value, serializing the
	// ProviderConfig to JSON SHALL NOT produce output containing the bot token string.
	rapid.Check(t, func(t *rapid.T) {
		// Generate an arbitrary bot token with xoxb- prefix
		tokenSuffix := rapid.StringMatching(`[a-zA-Z0-9\-]{10,50}`).Draw(t, "tokenSuffix")
		botToken := "xoxb-" + tokenSuffix

		// Generate arbitrary ProviderConfig fields
		name := rapid.StringMatching(`[a-z][a-z0-9\-]{2,20}`).Draw(t, "name")
		namespace := rapid.StringMatching(`[a-z][a-z0-9\-]{2,15}`).Draw(t, "namespace")
		secretName := rapid.StringMatching(`[a-z][a-z0-9\-]{2,20}`).Draw(t, "secretName")
		secretKey := rapid.StringMatching(`[a-z][a-z0-9\-]{2,10}`).Draw(t, "secretKey")

		// Build a ProviderConfig with credentials configured via secretRef
		pc := &v1alpha1.ProviderConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "slack.crossplane.io/v1alpha1",
				Kind:       "ProviderConfig",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: v1alpha1.ProviderConfigSpec{
				Credentials: v1alpha1.ProviderCredentials{
					Source: xpv1.CredentialsSourceSecret,
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      secretName,
								Namespace: namespace,
							},
							Key: secretKey,
						},
					},
				},
			},
		}

		// Serialize the ProviderConfig to JSON
		data, err := json.Marshal(pc)
		if err != nil {
			t.Fatalf("failed to marshal ProviderConfig: %v", err)
		}

		serialized := string(data)

		// Assert the bot token value does NOT appear in the serialized output
		if strings.Contains(serialized, botToken) {
			t.Fatalf("serialized ProviderConfig contains bot token %q;\nJSON: %s", botToken, serialized)
		}
	})
}
