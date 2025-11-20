package collector

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils/mocks"
)

func TestNewCollector(t *testing.T) {
	t.Run("creates new collector successfully", func(t *testing.T) {
		mockRegionManager := &mocks.MockRegionManager{}
		collector := NewCollector(mockRegionManager)

		assert.NotNil(t, collector)
		assert.Equal(t, mockRegionManager, collector.regionManager)
	})
}

func TestCollect(t *testing.T) {
	testCases := []struct {
		name                    string
		regionManagerError      error
		expectedLogOutput       bool
		shouldCallRegionManager bool
	}{
		{
			name:                    "collect metrics success",
			regionManagerError:      nil,
			expectedLogOutput:       true,
			shouldCallRegionManager: true,
		},
		{
			name:                    "collect metrics with region manager error",
			regionManagerError:      errors.New("region manager failed"),
			expectedLogOutput:       true,
			shouldCallRegionManager: true,
		},
		{
			name:                    "collect metrics with context error",
			regionManagerError:      context.Canceled,
			expectedLogOutput:       true,
			shouldCallRegionManager: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRegionManager := &mocks.MockRegionManager{}
			collector := NewCollector(mockRegionManager)

			if tc.shouldCallRegionManager {
				mockRegionManager.On("CollectMetrics", mock.Anything, mock.Anything).
					Return(tc.regionManagerError)
			}

			ch := make(chan prometheus.Metric, 100)

			collector.Collect(ch)

			close(ch)

			mockRegionManager.AssertExpectations(t)
		})
	}
}
