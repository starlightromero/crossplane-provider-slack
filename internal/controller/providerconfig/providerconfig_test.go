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
	"testing"

	corev1 "k8s.io/api/core/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"

	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
)

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{name: "valid token", token: "xoxb-123-456-abc", wantErr: false},
		{name: "valid token minimal", token: "xoxb-x", wantErr: false},
		{name: "empty token", token: "", wantErr: true},
		{name: "wrong prefix xoxp", token: "xoxp-123-456", wantErr: true},
		{name: "wrong prefix xoxa", token: "xoxa-123-456", wantErr: true},
		{name: "no prefix", token: "some-random-token", wantErr: true},
		{name: "partial prefix", token: "xoxb", wantErr: true},
		{name: "exact prefix only", token: "xoxb-", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken(%q) error = %v, wantErr %v", tt.token, err, tt.wantErr)
			}
		})
	}
}

func TestSetCredentialCondition(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		extractErr error
		wantStatus corev1.ConditionStatus
		wantReason xpv1.ConditionReason
	}{
		{
			name:       "valid token sets Ready=True",
			token:      "xoxb-123-456-abc",
			extractErr: nil,
			wantStatus: corev1.ConditionTrue,
			wantReason: "",
		},
		{
			name:       "extract error sets Ready=False with SecretNotFound",
			token:      "",
			extractErr: errorf("secret not found"),
			wantStatus: corev1.ConditionFalse,
			wantReason: ReasonSecretNotFound,
		},
		{
			name:       "invalid token sets Ready=False with InvalidCredentials",
			token:      "xoxp-wrong-prefix",
			extractErr: nil,
			wantStatus: corev1.ConditionFalse,
			wantReason: ReasonInvalidCredentials,
		},
		{
			name:       "empty token sets Ready=False with InvalidCredentials",
			token:      "",
			extractErr: nil,
			wantStatus: corev1.ConditionFalse,
			wantReason: ReasonInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := &v1alpha1.ProviderConfig{}
			SetCredentialCondition(pc, tt.token, tt.extractErr)

			cond := pc.GetCondition(xpv1.TypeReady)
			if cond.Status != tt.wantStatus {
				t.Errorf("condition status = %v, want %v", cond.Status, tt.wantStatus)
			}
			if tt.wantReason != "" && cond.Reason != tt.wantReason {
				t.Errorf("condition reason = %v, want %v", cond.Reason, tt.wantReason)
			}
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

func errorf(msg string) error {
	return &testError{msg: msg}
}
