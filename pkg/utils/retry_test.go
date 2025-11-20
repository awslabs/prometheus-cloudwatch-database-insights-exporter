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
