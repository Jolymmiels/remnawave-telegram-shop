package utils

import (
	"context"
	"log/slog"
	"math"
	"net"
	"strings"
	"time"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	RetryOn5xx  bool
	RetryOnNet  bool
}

// DefaultRetryConfig returns sensible defaults for API calls
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:  3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		RetryOn5xx:  true,
		RetryOnNet:  true,
	}
}

// IsRetryableError checks if error is worth retrying
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors
	if strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "broken pipe") {
		return true
	}

	// Check for net.Error (timeout, temporary)
	var netErr net.Error
	if ok := isNetError(err, &netErr); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	// HTTP 5xx errors
	if strings.Contains(errStr, "status code: 5") ||
		strings.Contains(errStr, "Status: 5") ||
		strings.Contains(errStr, "unexpected status code: 5") {
		return true
	}

	// Rate limiting
	if strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "Too Many Requests") {
		return true
	}

	return false
}

func isNetError(err error, target *net.Error) bool {
	if e, ok := err.(net.Error); ok {
		*target = e
		return true
	}
	return false
}

// WithRetry executes function with exponential backoff retry
func WithRetry[T any](ctx context.Context, cfg RetryConfig, operation string, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		result, lastErr = fn()
		if lastErr == nil {
			return result, nil
		}

		if !IsRetryableError(lastErr) {
			return result, lastErr
		}

		if attempt < cfg.MaxRetries {
			delay := time.Duration(float64(cfg.BaseDelay) * math.Pow(2, float64(attempt)))
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}

			slog.Warn("Retrying operation",
				"operation", operation,
				"attempt", attempt+1,
				"max_retries", cfg.MaxRetries,
				"delay", delay,
				"error", lastErr.Error(),
			)

			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return result, lastErr
}

// WithRetryNoResult executes function without return value with retry
func WithRetryNoResult(ctx context.Context, cfg RetryConfig, operation string, fn func() error) error {
	_, err := WithRetry(ctx, cfg, operation, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
