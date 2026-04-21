package slack

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoWithRetry_RetryAfterHeader(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	// Override sleep to avoid real delays in tests.
	origSleep := sleepFunc
	var sleptDuration time.Duration
	sleepFunc = func(_ context.Context, d time.Duration) error {
		sleptDuration = d
		return nil
	}
	defer func() { sleepFunc = origSleep }()

	c := NewClient("xoxb-test", WithBaseURL(server.URL))
	_, err := c.Do(context.Background(), "test.method", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
	if sleptDuration != 1*time.Second {
		t.Errorf("expected sleep of 1s, got %v", sleptDuration)
	}
}

func TestDoWithRetry_ExponentialBackoff(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= 2 {
			// No Retry-After header — triggers backoff.
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	origSleep := sleepFunc
	var sleepCalls int
	sleepFunc = func(_ context.Context, d time.Duration) error {
		sleepCalls++
		return nil
	}
	defer func() { sleepFunc = origSleep }()

	c := NewClient("xoxb-test", WithBaseURL(server.URL))
	_, err := c.Do(context.Background(), "test.method", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
	if sleepCalls != 2 {
		t.Errorf("expected 2 sleep calls, got %d", sleepCalls)
	}
}

func TestDoWithRetry_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	origSleep := sleepFunc
	sleepFunc = func(_ context.Context, d time.Duration) error { return nil }
	defer func() { sleepFunc = origSleep }()

	c := NewClient("xoxb-test", WithBaseURL(server.URL))
	_, err := c.Do(context.Background(), "test.method", nil)
	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}
	if !IsRetriableError(err) {
		t.Errorf("expected retriable error, got: %v", err)
	}
}

func TestDoWithRetry_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	origSleep := sleepFunc
	sleepFunc = func(ctx context.Context, d time.Duration) error {
		return ctx.Err()
	}
	defer func() { sleepFunc = origSleep }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	c := NewClient("xoxb-test", WithBaseURL(server.URL))
	_, err := c.Do(ctx, "test.method", nil)
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}

func TestCalculateBackoff_Bounds(t *testing.T) {
	for attempt := 0; attempt < 10; attempt++ {
		for i := 0; i < 100; i++ {
			d := CalculateBackoff(attempt)
			expectedMax := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
			if expectedMax > maxDelay {
				expectedMax = maxDelay
			}
			if d < 0 {
				t.Errorf("attempt %d: backoff %v is negative", attempt, d)
			}
			if d >= expectedMax {
				t.Errorf("attempt %d: backoff %v >= max %v", attempt, d, expectedMax)
			}
		}
	}
}
