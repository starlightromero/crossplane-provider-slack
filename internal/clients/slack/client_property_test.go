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

package slack

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: crossplane-provider-slack
// Property 2: Every Slack API request includes the Authorization header
// **Validates: Requirements 2.1**

func TestProperty_AuthorizationHeaderIncluded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate an arbitrary token with xoxb- prefix
		suffix := rapid.StringMatching(`[a-zA-Z0-9\-]{5,40}`).Draw(t, "tokenSuffix")
		token := "xoxb-" + suffix

		// Generate an arbitrary API method name
		method := rapid.StringMatching(`[a-z]+\.[a-z]+`).Draw(t, "method")

		var capturedAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}))
		defer server.Close()

		c := NewClient(token, WithBaseURL(server.URL))
		_, _ = c.Do(context.Background(), method, nil)

		expected := "Bearer " + token
		if capturedAuth != expected {
			t.Fatalf("expected Authorization header %q, got %q", expected, capturedAuth)
		}
	})
}

// Feature: crossplane-provider-slack
// Property 3: Rate-limited responses with Retry-After are retried after the specified duration
// **Validates: Requirements 2.2**

func TestProperty_RetryAfterCompliance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate arbitrary Retry-After value between 1 and 120 seconds
		retryAfterSeconds := rapid.IntRange(1, 120).Draw(t, "retryAfterSeconds")

		var requestCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount == 1 {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}))
		defer server.Close()

		// Override sleepFunc to capture the wait duration
		origSleep := sleepFunc
		var capturedDuration time.Duration
		sleepFunc = func(_ context.Context, d time.Duration) error {
			capturedDuration = d
			return nil
		}
		defer func() { sleepFunc = origSleep }()

		c := NewClient("xoxb-test-token", WithBaseURL(server.URL))
		_, err := c.Do(context.Background(), "test.method", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedDuration := time.Duration(retryAfterSeconds) * time.Second
		if capturedDuration < expectedDuration {
			t.Fatalf("expected sleep >= %v, got %v", expectedDuration, capturedDuration)
		}
	})
}

// Feature: crossplane-provider-slack
// Property 4: Exponential backoff with jitter produces bounded delays
// **Validates: Requirements 2.3**

func TestProperty_ExponentialBackoffBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate arbitrary retry attempt numbers (0-10)
		attempt := rapid.IntRange(0, 10).Draw(t, "attempt")

		result := CalculateBackoff(attempt)

		// Compute the expected upper bound: min(2^attempt * 1s, 60s)
		upperBound := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
		if upperBound > maxDelay {
			upperBound = maxDelay
		}

		// Result must be in [0, upperBound)
		if result < 0 {
			t.Fatalf("CalculateBackoff(%d) = %v, expected non-negative", attempt, result)
		}
		if result >= upperBound {
			t.Fatalf("CalculateBackoff(%d) = %v, expected < %v", attempt, result, upperBound)
		}
	})
}

// Feature: crossplane-provider-slack
// Property 5: Slack API error responses are parsed into structured errors
// **Validates: Requirements 2.4**

func TestProperty_SlackErrorParsing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate arbitrary error code strings (alphanumeric with underscores)
		errorCode := rapid.StringMatching(`[a-z][a-z0-9_]{2,30}`).Draw(t, "errorCode")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"ok":    false,
				"error": errorCode,
			})
		}))
		defer server.Close()

		c := NewClient("xoxb-test-token", WithBaseURL(server.URL))
		_, err := c.Do(context.Background(), "test.method", nil)

		if err == nil {
			t.Fatalf("expected error for error code %q, got nil", errorCode)
		}

		var slackErr *SlackError
		if !errors.As(err, &slackErr) {
			t.Fatalf("expected *SlackError, got %T: %v", err, err)
		}

		if slackErr.Code != errorCode {
			t.Fatalf("expected SlackError.Code = %q, got %q", errorCode, slackErr.Code)
		}
	})
}
