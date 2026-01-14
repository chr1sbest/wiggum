package resilience

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"
	"testing"
)

func TestPermanentError(t *testing.T) {
	originalErr := errors.New("original error")
	permErr := NewPermanentError(originalErr)

	if permErr.Error() != originalErr.Error() {
		t.Errorf("expected %q, got %q", originalErr.Error(), permErr.Error())
	}

	var unwrapped *PermanentError
	if !errors.As(permErr, &unwrapped) {
		t.Error("expected to unwrap as PermanentError")
	}

	if !errors.Is(permErr, originalErr) {
		t.Error("expected permanent error to unwrap to original")
	}
}

func TestTransientError(t *testing.T) {
	originalErr := errors.New("original error")
	transErr := NewTransientError(originalErr)

	if transErr.Error() != originalErr.Error() {
		t.Errorf("expected %q, got %q", originalErr.Error(), transErr.Error())
	}

	var unwrapped *TransientError
	if !errors.As(transErr, &unwrapped) {
		t.Error("expected to unwrap as TransientError")
	}

	if !errors.Is(transErr, originalErr) {
		t.Error("expected transient error to unwrap to original")
	}
}

func TestNewPermanentError_Nil(t *testing.T) {
	if NewPermanentError(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestNewTransientError_Nil(t *testing.T) {
	if NewTransientError(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestIsPermanentError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "explicit permanent error",
			err:      NewPermanentError(errors.New("fatal")),
			expected: true,
		},
		{
			name:     "explicit transient error",
			err:      NewTransientError(errors.New("temporary")),
			expected: false,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "generic error (default transient)",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPermanentError(tt.err)
			if result != tt.expected {
				t.Errorf("IsPermanentError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "explicit transient error",
			err:      NewTransientError(errors.New("temporary")),
			expected: true,
		},
		{
			name:     "generic error (default transient)",
			err:      errors.New("some error"),
			expected: true,
		},
		{
			name:     "context canceled (permanent)",
			err:      context.Canceled,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTransientError(tt.err)
			if result != tt.expected {
				t.Errorf("IsTransientError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestClassifyError_FileSystem(t *testing.T) {
	// Permission denied should be permanent
	permErr := &os.PathError{
		Op:   "open",
		Path: "/root/secret",
		Err:  syscall.EACCES,
	}
	if !IsPermanentError(permErr) {
		t.Error("permission denied should be permanent")
	}

	// File not found should be permanent
	notFoundErr := &os.PathError{
		Op:   "open",
		Path: "/nonexistent",
		Err:  syscall.ENOENT,
	}
	if !IsPermanentError(notFoundErr) {
		t.Error("file not found should be permanent")
	}
}

func TestClassifyError_Syscall(t *testing.T) {
	// Connection refused is transient
	if IsPermanentError(syscall.ECONNREFUSED) {
		t.Error("connection refused should be transient")
	}

	// Connection reset is transient
	if IsPermanentError(syscall.ECONNRESET) {
		t.Error("connection reset should be transient")
	}

	// Permission denied is permanent
	if !IsPermanentError(syscall.EACCES) {
		t.Error("EACCES should be permanent")
	}
}

func TestClassifyError_DNS(t *testing.T) {
	// DNS not found should be permanent
	dnsErr := &net.DNSError{
		Err:        "no such host",
		Name:       "nonexistent.invalid",
		IsNotFound: true,
	}
	if !IsPermanentError(dnsErr) {
		t.Error("DNS not found should be permanent")
	}

	// DNS temporary failure should be transient
	dnsTemp := &net.DNSError{
		Err:         "temporary failure",
		Name:        "example.com",
		IsTemporary: true,
		IsNotFound:  false,
	}
	if IsPermanentError(dnsTemp) {
		t.Error("DNS temporary failure should be transient")
	}
}
