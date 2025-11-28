package utils

import (
	"context"
	"time"
)

func WithRetry[T any](ctx context.Context, operation func() (T, error), maxRetries int, baseDelay time.Duration) (T, error) {
	var result T
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err = operation()
		if err == nil {
			return result, nil
		}

		if attempt == maxRetries {
			return result, err
		}

		nextDelay := min(1<<attempt, 5)
		delay := baseDelay * time.Duration(nextDelay)
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}
	}

	return result, err
}
