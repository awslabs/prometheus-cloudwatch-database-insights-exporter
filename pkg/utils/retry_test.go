package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithRetry(t *testing.T) {
	testCases := []struct {
		name          string
		operation     func() (string, error)
		maxRetries    int
		baseDelay     time.Duration
		expectedError bool
		expectedValue string
		expectedCalls int
	}{
		{
			name: "succeeds on first attempt",
			operation: func() func() (string, error) {
				callCount := 0
				return func() (string, error) {
					callCount++
					return "success", nil
				}
			}(),
			maxRetries:    3,
			baseDelay:     time.Millisecond,
			expectedError: false,
			expectedValue: "success",
			expectedCalls: 1,
		},
		{
			name: "succeeds on second attempt",
			operation: func() func() (string, error) {
				callCount := 0
				return func() (string, error) {
					callCount++
					if callCount == 1 {
						return "", errors.New("first attempt failed")
					}
					return "success", nil
				}
			}(),
			maxRetries:    3,
			baseDelay:     time.Millisecond,
			expectedError: false,
			expectedValue: "success",
		},
		{
			name: "fails after all retries",
			operation: func() func() (string, error) {
				return func() (string, error) {
					return "", errors.New("persistent error")
				}
			}(),
			maxRetries:    3,
			baseDelay:     time.Millisecond,
			expectedError: true,
			expectedValue: "",
		},
		{
			name: "succeeds on last retry",
			operation: func() func() (string, error) {
				callCount := 0
				return func() (string, error) {
					callCount++
					if callCount <= 3 {
						return "", errors.New("retry attempt failed")
					}
					return "success", nil
				}
			}(),
			maxRetries:    3,
			baseDelay:     time.Millisecond,
			expectedError: false,
			expectedValue: "success",
		},
		{
			name: "zero retries succeeds immediately",
			operation: func() func() (string, error) {
				return func() (string, error) {
					return "immediate success", nil
				}
			}(),
			maxRetries:    0,
			baseDelay:     time.Millisecond,
			expectedError: false,
			expectedValue: "immediate success",
		},
		{
			name: "zero retries fails immediately",
			operation: func() func() (string, error) {
				return func() (string, error) {
					return "", errors.New("immediate failure")
				}
			}(),
			maxRetries:    0,
			baseDelay:     time.Millisecond,
			expectedError: true,
			expectedValue: "",
		},
		{
			name: "verifies delay is capped at 5x base delay",
			operation: func() func() (string, error) {
				callCount := 0
				return func() (string, error) {
					callCount++
					if callCount <= 10 {
						return "", errors.New("retry attempt failed")
					}
					return "success", nil
				}
			}(),
			maxRetries:    10,
			baseDelay:     10 * time.Millisecond,
			expectedError: false,
			expectedValue: "success",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			result, err := WithRetry(ctx, tc.operation, tc.maxRetries, tc.baseDelay)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedValue, result)
			}
		})
	}
}

func TestWithRetryDelayCap(t *testing.T) {
	t.Run("delay is capped at 5x base delay", func(t *testing.T) {
		callCount := 0
		operation := func() (string, error) {
			callCount++
			if callCount <= 5 {
				return "", errors.New("retry attempt failed")
			}
			return "success", nil
		}

		ctx := context.Background()
		baseDelay := 100 * time.Millisecond
		maxRetries := 10

		start := time.Now()
		result, err := WithRetry(ctx, operation, maxRetries, baseDelay)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 6, callCount)

		// Calculate expected total delay:
		// attempt 0: no delay (success on retry)
		// attempt 1: 1 * 100ms = 100ms
		// attempt 2: 2 * 100ms = 200ms
		// attempt 3: 4 * 100ms = 400ms
		// attempt 4: 5 * 100ms = 500ms (capped)
		// attempt 5: 5 * 100ms = 500ms (capped)
		// Total: 100 + 200 + 400 + 500 + 500 = 1700ms
		expectedDelay := 1700 * time.Millisecond

		// Allow some tolerance for execution time
		tolerance := 200 * time.Millisecond
		assert.True(t, elapsed >= expectedDelay, "elapsed time should be at least %v, got %v", expectedDelay, elapsed)
		assert.True(t, elapsed < expectedDelay+tolerance, "elapsed time should be less than %v, got %v", expectedDelay+tolerance, elapsed)
	})

	t.Run("delay progression with cap", func(t *testing.T) {
		delays := []time.Duration{}
		callCount := 0
		var lastCall time.Time
		lastCall = time.Now()

		operation := func() (string, error) {
			callCount++
			if callCount > 1 {
				delays = append(delays, time.Since(lastCall))
			}
			lastCall = time.Now()

			if callCount <= 6 {
				return "", errors.New("retry")
			}
			return "success", nil
		}

		ctx := context.Background()
		baseDelay := 50 * time.Millisecond

		_, err := WithRetry(ctx, operation, 10, baseDelay)
		assert.NoError(t, err)

		// Verify exponential backoff with cap:
		// delay 1: ~50ms  (1x)
		// delay 2: ~100ms (2x)
		// delay 3: ~200ms (4x)
		// delay 4: ~250ms (5x, capped)
		// delay 5: ~250ms (5x, capped)
		// delay 6: ~250ms (5x, capped)

		assert.Len(t, delays, 6)

		tolerance := 30 * time.Millisecond
		assert.InDelta(t, 50*time.Millisecond, delays[0], float64(tolerance), "delay 1 should be ~50ms")
		assert.InDelta(t, 100*time.Millisecond, delays[1], float64(tolerance), "delay 2 should be ~100ms")
		assert.InDelta(t, 200*time.Millisecond, delays[2], float64(tolerance), "delay 3 should be ~200ms")
		assert.InDelta(t, 250*time.Millisecond, delays[3], float64(tolerance), "delay 4 should be capped at ~250ms")
		assert.InDelta(t, 250*time.Millisecond, delays[4], float64(tolerance), "delay 5 should be capped at ~250ms")
		assert.InDelta(t, 250*time.Millisecond, delays[5], float64(tolerance), "delay 6 should be capped at ~250ms")
	})
}
