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

// Package providerconfig implements the ProviderConfig controller for the
// Slack provider. It validates bot token credentials and sets Ready conditions.
package providerconfig

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"

	"github.com/avodah-inc/crossplane-provider-slack/apis/v1alpha1"
)

// Condition reasons for ProviderConfig status.
const (
	// ReasonSecretNotFound indicates the referenced Secret does not exist.
	ReasonSecretNotFound xpv1.ConditionReason = "SecretNotFound"

	// ReasonInvalidCredentials indicates the bot token is empty or has an
	// invalid prefix.
	ReasonInvalidCredentials xpv1.ConditionReason = "InvalidCredentials"
)

// Error messages.
const (
	errGetSecret      = "cannot get credentials secret"
	errSecretKeyEmpty = "credentials secret key is empty"
	errInvalidToken   = "bot token must start with xoxb- prefix"
	errNoSecretRef    = "spec.credentials.secretRef is required when source is Secret"
)

// tokenPrefix is the required prefix for Slack bot tokens.
const tokenPrefix = "xoxb-"

// ValidateToken checks whether the given token is a valid Slack bot token.
// It returns an error if the token is empty or does not start with "xoxb-".
// This function is exported for reuse in property-based tests.
func ValidateToken(token string) error {
	if token == "" {
		return errors.New(errSecretKeyEmpty)
	}
	if !strings.HasPrefix(token, tokenPrefix) {
		return errors.New(errInvalidToken)
	}
	return nil
}

// ExtractToken reads the bot token from the Secret referenced by the
// ProviderConfig credentials. It returns the token string or an error.
func ExtractToken(ctx context.Context, kube client.Client, pc *v1alpha1.ProviderConfig) (string, error) {
	if pc.Spec.Credentials.Source != xpv1.CredentialsSourceSecret {
		return "", errors.New(errNoSecretRef)
	}

	ref := pc.Spec.Credentials.SecretRef
	if ref == nil {
		return "", errors.New(errNoSecretRef)
	}

	secret := &corev1.Secret{}
	nn := types.NamespacedName{
		Namespace: ref.Namespace,
		Name:      ref.Name,
	}
	if err := kube.Get(ctx, nn, secret); err != nil {
		return "", errors.Wrap(err, errGetSecret)
	}

	token := string(secret.Data[ref.Key])
	return token, nil
}

// SetCredentialCondition validates the token and sets the appropriate Ready
// condition on the ProviderConfig.
func SetCredentialCondition(pc *v1alpha1.ProviderConfig, token string, extractErr error) {
	if extractErr != nil {
		pc.SetConditions(xpv1.Condition{
			Type:    xpv1.TypeReady,
			Status:  corev1.ConditionFalse,
			Reason:  ReasonSecretNotFound,
			Message: extractErr.Error(),
		})
		return
	}

	if err := ValidateToken(token); err != nil {
		pc.SetConditions(xpv1.Condition{
			Type:    xpv1.TypeReady,
			Status:  corev1.ConditionFalse,
			Reason:  ReasonInvalidCredentials,
			Message: fmt.Sprintf("invalid bot token: %s", err.Error()),
		})
		return
	}

	pc.SetConditions(xpv1.Condition{
		Type:   xpv1.TypeReady,
		Status: corev1.ConditionTrue,
	})
}
