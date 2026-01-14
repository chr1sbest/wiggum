package resilience

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"
)

// PermanentError wraps an error to mark it as non-retryable.
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

// NewPermanentError wraps an error to indicate it should not be retried.
func NewPermanentError(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{Err: err}
}

// TransientError wraps an error to mark it as retryable.
type TransientError struct {
	Err error
}

func (e *TransientError) Error() string {
	return e.Err.Error()
}

func (e *TransientError) Unwrap() error {
	return e.Err
}

// NewTransientError wraps an error to explicitly indicate it should be retried.
func NewTransientError(err error) error {
	if err == nil {
		return nil
	}
	return &TransientError{Err: err}
}

// IsPermanentError checks if an error is marked as permanent (non-retryable).
func IsPermanentError(err error) bool {
	if err == nil {
		return false
	}

	// Check for explicit permanent error wrapper
	var permErr *PermanentError
	if errors.As(err, &permErr) {
		return true
	}

	// Check for explicitly transient errors (these are retryable)
	var transErr *TransientError
	if errors.As(err, &transErr) {
		return false
	}

	// Check for context errors - these are permanent
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Classify common error types
	return classifyError(err)
}

// IsTransientError checks if an error is transient (retryable).
func IsTransientError(err error) bool {
	return err != nil && !IsPermanentError(err)
}

// classifyError determines if an error is permanent based on its type.
func classifyError(err error) bool {
	// Network errors - check if they're connection-related (transient) vs DNS/lookup (permanent)
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Timeout errors are transient
		if netErr.Timeout() {
			return false
		}
	}

	// DNS lookup failures are typically permanent
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return true
		}
	}

	// File system errors
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		// Permission denied is permanent
		if errors.Is(pathErr.Err, syscall.EACCES) || errors.Is(pathErr.Err, syscall.EPERM) {
			return true
		}
		// File not found is permanent
		if errors.Is(pathErr.Err, syscall.ENOENT) {
			return true
		}
	}

	// Syscall errors
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		switch sysErr {
		case syscall.EACCES, syscall.EPERM, syscall.ENOENT, syscall.ENOTDIR:
			return true // Permanent
		case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ETIMEDOUT:
			return false // Transient
		}
	}

	// Default: assume transient (allow retry)
	return false
}
