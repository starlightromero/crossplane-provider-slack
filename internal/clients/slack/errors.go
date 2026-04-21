package slack

import (
	"errors"
	"fmt"
	"net"
	"os"
)

// SlackError represents a structured error from the Slack API.
type SlackError struct {
	Code    string // e.g. "name_taken", "channel_not_found"
	Message string // Human-readable message
}

func (e *SlackError) Error() string { return fmt.Sprintf("slack: %s: %s", e.Code, e.Message) }

// IsRetriable returns true if the error is transient and the operation should be retried.
func (e *SlackError) IsRetriable() bool {
	switch e.Code {
	case "internal_error", "fatal_error", "request_timeout":
		return true
	default:
		return false
	}
}

// RetriableError wraps an error to indicate the operation should be retried
// by the reconciler (requeue).
type RetriableError struct {
	Err error
}

func (e *RetriableError) Error() string {
	return fmt.Sprintf("retriable: %v", e.Err)
}

func (e *RetriableError) Unwrap() error {
	return e.Err
}

// IsRetriableError returns true if the error is a RetriableError or a
// SlackError with a retriable code.
func IsRetriableError(err error) bool {
	if err == nil {
		return false
	}
	var re *RetriableError
	if errors.As(err, &re) {
		return true
	}
	var se *SlackError
	if errors.As(err, &se) {
		return se.IsRetriable()
	}
	return false
}

// IsNetworkError returns true if the error is a network-level error
// (connection refused, DNS failure, timeout) that should be retried.
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	// Check for net.Error (includes timeouts)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	// Check for OS-level errors (connection refused, etc.)
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	// Check for syscall errors (ECONNREFUSED, etc.)
	var sysErr *os.SyscallError
	if errors.As(err, &sysErr) {
		return true
	}
	return false
}
