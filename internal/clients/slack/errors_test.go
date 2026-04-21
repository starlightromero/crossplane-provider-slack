package slack

import (
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
)

func TestRetriableError(t *testing.T) {
	inner := fmt.Errorf("connection refused")
	re := &RetriableError{Err: inner}

	if re.Error() != "retriable: connection refused" {
		t.Errorf("unexpected error string: %s", re.Error())
	}
	if !errors.Is(re, inner) {
		t.Error("expected Unwrap to return inner error")
	}
}

func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"regular error", fmt.Errorf("something"), false},
		{"RetriableError", &RetriableError{Err: fmt.Errorf("timeout")}, true},
		{"SlackError retriable", &SlackError{Code: "internal_error"}, true},
		{"SlackError non-retriable", &SlackError{Code: "channel_not_found"}, false},
		{"wrapped RetriableError", fmt.Errorf("wrap: %w", &RetriableError{Err: fmt.Errorf("x")}), true},
		{"wrapped SlackError retriable", fmt.Errorf("wrap: %w", &SlackError{Code: "fatal_error"}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetriableError(tt.err); got != tt.expected {
				t.Errorf("IsRetriableError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"regular error", fmt.Errorf("something"), false},
		{"net.OpError", &net.OpError{Op: "dial", Err: fmt.Errorf("refused")}, true},
		{"net.DNSError", &net.DNSError{Err: "no such host"}, true},
		{"os.SyscallError", &os.SyscallError{Syscall: "connect", Err: fmt.Errorf("refused")}, true},
		{"wrapped net.OpError", fmt.Errorf("wrap: %w", &net.OpError{Op: "dial", Err: fmt.Errorf("x")}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNetworkError(tt.err); got != tt.expected {
				t.Errorf("IsNetworkError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
