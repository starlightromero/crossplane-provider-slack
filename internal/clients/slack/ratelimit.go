package slack

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const (
	// maxRetries is the maximum number of retries for rate-limited requests.
	maxRetries = 3

	// baseDelay is the base delay for exponential backoff.
	baseDelay = 1 * time.Second

	// maxDelay is the maximum backoff delay cap.
	maxDelay = 60 * time.Second
)

// sleepFunc is a function that sleeps for the given duration. It can be
// overridden in tests to avoid real sleeps.
var sleepFunc = func(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CalculateBackoff computes the backoff duration for a given retry attempt.
// The formula is: min(2^attempt * baseDelay, maxDelay) with random jitter
// in [0, backoff). This function is exported for property-based testing.
func CalculateBackoff(attempt int) time.Duration {
	backoff := float64(baseDelay) * math.Pow(2, float64(attempt))
	if backoff > float64(maxDelay) {
		backoff = float64(maxDelay)
	}
	// Apply jitter: random value in [0, backoff)
	jittered := time.Duration(rand.Int63n(int64(backoff)))
	return jittered
}

// doWithRetry executes an HTTP request and handles rate limiting (HTTP 429).
// It retries up to maxRetries times, respecting Retry-After headers when
// present, or using exponential backoff with jitter otherwise.
func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Clone the request for retries (body already consumed on first attempt
		// is handled by the caller rebuilding the request).
		resp, err = c.httpClient.Do(req)
		if err != nil {
			// Network error — wrap as retriable for the reconciler.
			if IsNetworkError(err) {
				return nil, &RetriableError{Err: fmt.Errorf("network error on attempt %d: %w", attempt+1, err)}
			}
			return nil, err
		}

		// Not rate-limited — return immediately.
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Close the 429 response body before retrying.
		resp.Body.Close()

		// If we've exhausted retries, return an error.
		if attempt == maxRetries {
			return nil, &RetriableError{
				Err: fmt.Errorf("rate limited after %d retries", maxRetries),
			}
		}

		// Determine wait duration.
		var waitDuration time.Duration
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			seconds, parseErr := strconv.Atoi(retryAfter)
			if parseErr == nil && seconds > 0 {
				waitDuration = time.Duration(seconds) * time.Second
			} else {
				// Malformed Retry-After — fall back to backoff.
				waitDuration = CalculateBackoff(attempt)
			}
		} else {
			// No Retry-After header — use exponential backoff with jitter.
			waitDuration = CalculateBackoff(attempt)
		}

		// Wait before retrying.
		if err := sleepFunc(ctx, waitDuration); err != nil {
			return nil, fmt.Errorf("context cancelled during rate-limit wait: %w", err)
		}

		// Re-clone the request for the next attempt.
		req = req.Clone(ctx)
	}

	// Should not reach here, but just in case.
	return resp, err
}
