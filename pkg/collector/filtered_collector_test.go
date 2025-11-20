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

func TestNewFilteredCollector(t *testing.T) {
	testCases := []struct {
		name           string
		regionManager  *mocks.MockRegionManager
		instanceFilter []string
	}{
		{
			name:           "creates filtered collector with valid region manager and instance filter",
			regionManager:  &mocks.MockRegionManager{},
			instanceFilter: []string{"instance1", "instance2"},
		},
		{
			name:           "creates filtered collector with empty instance filter",
			regionManager:  &mocks.MockRegionManager{},
			instanceFilter: []string{},
		},
		{
			name:           "creates filtered collector with nil region manager",
			regionManager:  nil,
			instanceFilter: []string{"instance1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			collector := NewFilteredCollector(tc.regionManager, tc.instanceFilter)

			assert.NotNil(t, collector)
			assert.Equal(t, tc.regionManager, collector.regionManager)
			assert.Equal(t, tc.instanceFilter, collector.instanceFilter)
		})
	}
}

func TestFilteredCollectorDescribe(t *testing.T) {
	t.Run("describe does not panic", func(t *testing.T) {
		mockRegionManager := &mocks.MockRegionManager{}
		collector := NewFilteredCollector(mockRegionManager, []string{"instance1"})

		ch := make(chan *prometheus.Desc, 10)

		assert.NotPanics(t, func() {
			collector.Describe(ch)
		})

		close(ch)
	})
}

func TestFilteredCollectorCollect(t *testing.T) {
	testCases := []struct {
		name                    string
		instanceFilter          []string
		regionManagerError      error
		expectedLogOutput       bool
		shouldCallRegionManager bool
	}{
		{
			name:                    "collect with successful filtered metric collection",
			instanceFilter:          []string{"instance1", "instance2"},
			regionManagerError:      nil,
			expectedLogOutput:       true,
			shouldCallRegionManager: true,
		},
		{
			name:                    "collect with error from CollectMetricsForInstances",
			instanceFilter:          []string{"instance1"},
			regionManagerError:      errors.New("collection failed"),
			expectedLogOutput:       true,
			shouldCallRegionManager: true,
		},
		{
			name:                    "collect with context cancellation",
			instanceFilter:          []string{"instance1"},
			regionManagerError:      context.Canceled,
			expectedLogOutput:       true,
			shouldCallRegionManager: true,
		},
		{
			name:                    "collect with empty instance filter",
			instanceFilter:          []string{},
			regionManagerError:      nil,
			expectedLogOutput:       true,
			shouldCallRegionManager: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRegionManager := &mocks.MockRegionManager{}
			collector := NewFilteredCollector(mockRegionManager, tc.instanceFilter)

			if tc.shouldCallRegionManager {
				mockRegionManager.On("CollectMetricsForInstances", mock.Anything, tc.instanceFilter, mock.Anything).
					Return(tc.regionManagerError)
			}

			ch := make(chan prometheus.Metric, 100)

			collector.Collect(ch)

			close(ch)

			mockRegionManager.AssertExpectations(t)
		})
	}
}
