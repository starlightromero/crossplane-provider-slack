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
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: crossplane-provider-slack, Property 1: Bot token validation accepts only xoxb- prefixed strings
// **Validates: Requirements 1.4, 1.5**

func TestProperty_ValidTokensAccepted(t *testing.T) {
	// Any string starting with "xoxb-" should be accepted by ValidateToken.
	rapid.Check(t, func(t *rapid.T) {
		suffix := rapid.String().Draw(t, "suffix")
		token := "xoxb-" + suffix

		err := ValidateToken(token)
		if err != nil {
			t.Fatalf("ValidateToken(%q) returned error %v, expected nil for xoxb- prefixed token", token, err)
		}
	})
}

func TestProperty_InvalidTokensRejected(t *testing.T) {
	// Any string NOT starting with "xoxb-" should be rejected by ValidateToken.
	rapid.Check(t, func(t *rapid.T) {
		token := rapid.String().Draw(t, "token")

		// Skip tokens that happen to start with "xoxb-"
		if strings.HasPrefix(token, "xoxb-") {
			t.Skip("generated token has xoxb- prefix, skipping")
		}

		err := ValidateToken(token)
		if err == nil {
			t.Fatalf("ValidateToken(%q) returned nil, expected error for token without xoxb- prefix", token)
		}
	})
}
