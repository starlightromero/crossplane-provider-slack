package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("xoxb-test-token")
	if c.token != "xoxb-test-token" {
		t.Errorf("expected token %q, got %q", "xoxb-test-token", c.token)
	}
	if c.baseURL != DefaultBaseURL {
		t.Errorf("expected baseURL %q, got %q", DefaultBaseURL, c.baseURL)
	}
	if c.httpClient == nil {
		t.Error("expected non-nil httpClient")
	}
}

func TestNewClientWithOptions(t *testing.T) {
	customHTTP := &http.Client{}
	c := NewClient("xoxb-test", WithBaseURL("http://localhost"), WithHTTPClient(customHTTP))
	if c.baseURL != "http://localhost" {
		t.Errorf("expected baseURL %q, got %q", "http://localhost", c.baseURL)
	}
	if c.httpClient != customHTTP {
		t.Error("expected custom HTTP client")
	}
}

func TestDoSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer xoxb-test" {
			t.Errorf("expected Authorization header %q, got %q", "Bearer xoxb-test", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type %q, got %q", "application/x-www-form-urlencoded", got)
		}
		if r.URL.Path != "/conversations.info" {
			t.Errorf("expected path /conversations.info, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channel": map[string]any{
				"id":   "C123",
				"name": "general",
			},
		})
	}))
	defer server.Close()

	c := NewClient("xoxb-test", WithBaseURL(server.URL))
	params := url.Values{"channel": {"C123"}}
	raw, err := c.Do(context.Background(), "conversations.info", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil response body")
	}
}

func TestDoSlackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "channel_not_found",
		})
	}))
	defer server.Close()

	c := NewClient("xoxb-test", WithBaseURL(server.URL))
	_, err := c.Do(context.Background(), "conversations.info", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	slackErr, ok := err.(*SlackError)
	if !ok {
		t.Fatalf("expected *SlackError, got %T", err)
	}
	if slackErr.Code != "channel_not_found" {
		t.Errorf("expected error code %q, got %q", "channel_not_found", slackErr.Code)
	}
}

func TestDoFormParams(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		receivedBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	c := NewClient("xoxb-test", WithBaseURL(server.URL))
	params := url.Values{
		"channel": {"C123"},
		"name":    {"new-name"},
	}
	_, err := c.Do(context.Background(), "conversations.rename", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := url.ParseQuery(receivedBody)
	if err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if parsed.Get("channel") != "C123" {
		t.Errorf("expected channel=C123, got %q", parsed.Get("channel"))
	}
	if parsed.Get("name") != "new-name" {
		t.Errorf("expected name=new-name, got %q", parsed.Get("name"))
	}
}

func TestSlackErrorIsRetriable(t *testing.T) {
	tests := []struct {
		code      string
		retriable bool
	}{
		{"internal_error", true},
		{"fatal_error", true},
		{"request_timeout", true},
		{"channel_not_found", false},
		{"name_taken", false},
		{"invalid_auth", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := &SlackError{Code: tt.code}
			if got := err.IsRetriable(); got != tt.retriable {
				t.Errorf("IsRetriable() = %v, want %v", got, tt.retriable)
			}
		})
	}
}
