package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/valksor/go-assern/internal/config"
)

func TestWithRetry_NoConfig(t *testing.T) {
	t.Parallel()

	calls := 0
	result, err := WithRetry[string](context.Background(), nil, func(ctx context.Context, attempt int) (string, error) {
		calls++

		return "ok", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestWithRetry_SucceedsOnFirstAttempt(t *testing.T) {
	t.Parallel()

	cfg := &config.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	calls := 0
	result, err := WithRetry[string](context.Background(), cfg, func(ctx context.Context, attempt int) (string, error) {
		calls++

		return "success", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("result = %q, want %q", result, "success")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestWithRetry_SucceedsOnSecondAttempt(t *testing.T) {
	t.Parallel()

	cfg := &config.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	calls := 0
	result, err := WithRetry[string](context.Background(), cfg, func(ctx context.Context, attempt int) (string, error) {
		calls++
		if attempt == 1 {
			return "", errors.New("transient error")
		}

		return "success", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("result = %q, want %q", result, "success")
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestWithRetry_ExhaustsAttempts(t *testing.T) {
	t.Parallel()

	cfg := &config.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	calls := 0
	_, err := WithRetry[string](context.Background(), cfg, func(ctx context.Context, attempt int) (string, error) {
		calls++

		return "", errors.New("persistent error")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}

	var maxRetryErr *MaxRetriesExceededError
	if !errors.As(err, &maxRetryErr) {
		t.Errorf("expected MaxRetriesExceededError, got %T", err)
	}
	if maxRetryErr.Attempts != 3 {
		t.Errorf("maxRetryErr.Attempts = %d, want 3", maxRetryErr.Attempts)
	}
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	t.Parallel()

	cfg := &config.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	calls := 0
	_, err := WithRetry[string](context.Background(), cfg, func(ctx context.Context, attempt int) (string, error) {
		calls++

		return "", context.Canceled
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (non-retryable should stop immediately)", calls)
	}
}

func TestWithRetry_ContextCancellation(t *testing.T) {
	t.Parallel()

	cfg := &config.RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := WithRetry[string](ctx, cfg, func(ctx context.Context, attempt int) (string, error) {
		calls++

		return "", errors.New("transient error")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestWithRetry_MaxDelayRespected(t *testing.T) {
	t.Parallel()

	cfg := &config.RetryConfig{
		MaxAttempts:   4,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      60 * time.Millisecond, // Lower than 50*2*2 = 200ms
		BackoffFactor: 2.0,
	}

	start := time.Now()
	calls := 0
	_, _ = WithRetry[string](context.Background(), cfg, func(ctx context.Context, attempt int) (string, error) {
		calls++

		return "", errors.New("error")
	})
	elapsed := time.Since(start)

	if calls != 4 {
		t.Errorf("calls = %d, want 4", calls)
	}

	// Expected: 50ms + 60ms + 60ms = 170ms (maxDelay caps 100ms and 200ms to 60ms)
	// Allow some tolerance
	expectedMin := 150 * time.Millisecond
	expectedMax := 250 * time.Millisecond
	if elapsed < expectedMin || elapsed > expectedMax {
		t.Errorf("elapsed = %v, want between %v and %v", elapsed, expectedMin, expectedMax)
	}
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"context canceled", context.Canceled, false},
		{"context deadline exceeded", context.DeadlineExceeded, false},
		{"server not started", ErrServerNotStarted, false},
		{"generic error", errors.New("some error"), true},
		{"retryable error", &RetryableError{Err: errors.New("retry me")}, true},
		{"wrapped retryable", &RetryableError{Err: context.Canceled}, true}, // Explicit retryable wrapping
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestCalculateBackoffDelay(t *testing.T) {
	t.Parallel()

	cfg := &config.RetryConfig{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 0},                      // First attempt has no delay
		{2, 100 * time.Millisecond}, // initial * 2^0
		{3, 200 * time.Millisecond}, // initial * 2^1
		{4, 400 * time.Millisecond}, // initial * 2^2
		{5, 800 * time.Millisecond}, // initial * 2^3
		{6, 1 * time.Second},        // capped at max
		{7, 1 * time.Second},        // still capped
	}

	for _, tt := range tests {
		result := CalculateBackoffDelay(cfg, tt.attempt)
		if result != tt.expected {
			t.Errorf("CalculateBackoffDelay(attempt=%d) = %v, want %v", tt.attempt, result, tt.expected)
		}
	}
}

func TestMaxRetriesExceededError(t *testing.T) {
	t.Parallel()

	underlying := errors.New("underlying error")
	err := &MaxRetriesExceededError{
		Attempts: 5,
		LastErr:  underlying,
	}

	if !errors.Is(err, underlying) {
		t.Error("MaxRetriesExceededError should unwrap to underlying error")
	}

	expected := "max retries exceeded after 5 attempts: underlying error"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestWithRetry_SingleAttemptConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.RetryConfig{
		MaxAttempts: 1, // Single attempt means no retries
	}

	calls := 0
	_, err := WithRetry[string](context.Background(), cfg, func(ctx context.Context, attempt int) (string, error) {
		calls++

		return "", errors.New("error")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestWithRetry_ZeroAttemptConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.RetryConfig{
		MaxAttempts: 0, // Zero means default to single attempt
	}

	calls := 0
	result, err := WithRetry[string](context.Background(), cfg, func(ctx context.Context, attempt int) (string, error) {
		calls++

		return "ok", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}
