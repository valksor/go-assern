package aggregator

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/valksor/go-assern/internal/config"
)

// RetryableError wraps an error to indicate it can be retried.
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error should trigger a retry.
// Explicit RetryableError wrapper takes precedence.
// Context cancellation and deadline exceeded are not retryable.
// Connection errors and transient failures are retryable by default.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Explicit retryable errors take precedence
	var retryableErr *RetryableError
	if errors.As(err, &retryableErr) {
		return true
	}

	// Context errors are never retryable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Server not started is not retryable
	if errors.Is(err, ErrServerNotStarted) {
		return false
	}

	// By default, assume transient failures are retryable
	return true
}

// RetryFunc is the function signature for operations that can be retried.
type RetryFunc[T any] func(ctx context.Context, attempt int) (T, error)

// WithRetry executes the given function with retry logic based on config.
// If retryCfg is nil, the function is executed once without retries.
func WithRetry[T any](ctx context.Context, retryCfg *config.RetryConfig, fn RetryFunc[T]) (T, error) {
	var zero T

	// No retry config means single attempt
	if retryCfg == nil || retryCfg.MaxAttempts <= 1 {
		return fn(ctx, 1)
	}

	var lastErr error
	delay := retryCfg.InitialDelay

	for attempt := 1; attempt <= retryCfg.MaxAttempts; attempt++ {
		result, err := fn(ctx, attempt)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if !IsRetryable(err) {
			return zero, err
		}

		// Don't sleep after the last attempt
		if attempt == retryCfg.MaxAttempts {
			break
		}

		// Check context before sleeping
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}

		// Sleep with exponential backoff
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}

		// Calculate next delay with backoff
		delay = time.Duration(float64(delay) * retryCfg.BackoffFactor)
		if delay > retryCfg.MaxDelay {
			delay = retryCfg.MaxDelay
		}
	}

	return zero, &MaxRetriesExceededError{
		Attempts: retryCfg.MaxAttempts,
		LastErr:  lastErr,
	}
}

// MaxRetriesExceededError indicates all retry attempts have been exhausted.
type MaxRetriesExceededError struct {
	Attempts int
	LastErr  error
}

func (e *MaxRetriesExceededError) Error() string {
	return "max retries exceeded after " + itoa(e.Attempts) + " attempts: " + e.LastErr.Error()
}

func (e *MaxRetriesExceededError) Unwrap() error {
	return e.LastErr
}

// itoa is a simple int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)
	negative := n < 0

	if negative {
		n = -n
	}

	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	if negative {
		i--
		buf[i] = '-'
	}

	return string(buf[i:])
}

// CalculateBackoffDelay computes the delay before a given attempt using exponential backoff.
// Attempt 1 has no delay, attempt 2 gets initial_delay, attempt 3 gets initial_delay * factor, etc.
// This is useful for logging or metrics.
func CalculateBackoffDelay(cfg *config.RetryConfig, attempt int) time.Duration {
	if cfg == nil || attempt <= 1 {
		return 0
	}

	// attempt 2 -> factor^0, attempt 3 -> factor^1, etc.
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.BackoffFactor, float64(attempt-2))
	if time.Duration(delay) > cfg.MaxDelay {
		return cfg.MaxDelay
	}

	return time.Duration(delay)
}
