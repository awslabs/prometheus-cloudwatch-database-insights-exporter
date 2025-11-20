package region

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils/mocks"
)

func TestNewMultiRegionManager(t *testing.T) {
	t.Run("Creates new multi region manager successfully", func(t *testing.T) {
		manager := NewMultiRegionManager()

		assert.NotNil(t, manager)
		assert.NotNil(t, manager.RegionManagers)
		assert.Empty(t, manager.RegionManagers)
	})
}

func TestAddRegionManager(t *testing.T) {
	testCases := []struct {
		name          string
		regions       []string
		expectedCount int
	}{
		{
			name:          "Add single region manager",
			regions:       []string{"us-west-2"},
			expectedCount: 1,
		},
		{
			name:          "Add multiple region managers",
			regions:       []string{"us-west-2", "us-east-1", "eu-west-1"},
			expectedCount: 3,
		},
		{
			name:          "Add no region managers",
			regions:       []string{},
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewMultiRegionManager()

			for _, region := range tc.regions {
				mockRM := &mocks.MockRegionManager{}
				manager.AddRegionManager(region, mockRM)
			}

			assert.Len(t, manager.RegionManagers, tc.expectedCount)

			for _, region := range tc.regions {
				assert.Contains(t, manager.RegionManagers, region)
			}
		})
	}
}

func TestMultiRegionManagerCollectMetrics(t *testing.T) {
	testCases := []struct {
		name                string
		regions             []string
		regionManagerErrors []error
		expectedError       error
		expectedMetricCalls int
	}{
		{
			name:                "Collect metrics success with single region",
			regions:             []string{"us-west-2"},
			regionManagerErrors: []error{nil},
			expectedError:       nil,
			expectedMetricCalls: 1,
		},
		{
			name:                "Collect metrics success with multiple regions",
			regions:             []string{"us-west-2", "us-east-1", "eu-west-1"},
			regionManagerErrors: []error{nil, nil, nil},
			expectedError:       nil,
			expectedMetricCalls: 3,
		},
		{
			name:                "Collect metrics with no regions",
			regions:             []string{},
			regionManagerErrors: []error{},
			expectedError:       nil,
			expectedMetricCalls: 0,
		},
		{
			name:                "Collect metrics with first region error (fail fast)",
			regions:             []string{"us-west-2", "us-east-1"},
			regionManagerErrors: []error{errors.New("first region failed")},
			expectedError:       errors.New("first region failed"),
			expectedMetricCalls: 1,
		},
		{
			name:                "Collect metrics with second region error",
			regions:             []string{"us-west-2", "us-east-1"},
			regionManagerErrors: []error{nil, errors.New("second region failed")},
			expectedError:       errors.New("second region failed"),
			expectedMetricCalls: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewMultiRegionManager()

			var mockRMs []*mocks.MockRegionManager
			for i, region := range tc.regions {
				mockRM := &mocks.MockRegionManager{}

				// Set up mock expectation if we have an error defined for this region
				if i < len(tc.regionManagerErrors) {
					if tc.expectedError != nil {
						// For error cases, use Maybe() since map iteration order is non-deterministic
						mockRM.On("CollectMetrics", mock.Anything, mock.Anything).
							Return(tc.regionManagerErrors[i]).Maybe()
					} else {
						mockRM.On("CollectMetrics", mock.Anything, mock.Anything).
							Return(tc.regionManagerErrors[i]).Once()
					}
				} else {
					// For regions beyond the error list, set up Maybe() expectation
					mockRM.On("CollectMetrics", mock.Anything, mock.Anything).
						Return(nil).Maybe()
				}

				manager.AddRegionManager(region, mockRM)
				mockRMs = append(mockRMs, mockRM)
			}

			ch := make(chan prometheus.Metric, 100)
			err := manager.CollectMetrics(context.Background(), ch)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			close(ch)

			for _, mockRM := range mockRMs {
				mockRM.AssertExpectations(t)
			}
		})
	}
}

func TestMultiRegionManagerCollectMetricsForInstances(t *testing.T) {
	testCases := []struct {
		name                string
		regions             []string
		instanceIdentifiers []string
		regionManagerErrors []error
		expectedError       error
		expectedMetricCalls int
	}{
		{
			name:                "Collect filtered metrics success with single region",
			regions:             []string{"us-west-2"},
			instanceIdentifiers: []string{"test-db-1"},
			regionManagerErrors: []error{nil},
			expectedError:       nil,
			expectedMetricCalls: 1,
		},
		{
			name:                "Collect filtered metrics success with multiple regions",
			regions:             []string{"us-west-2", "us-east-1", "eu-west-1"},
			instanceIdentifiers: []string{"test-db-1", "test-db-2"},
			regionManagerErrors: []error{nil, nil, nil},
			expectedError:       nil,
			expectedMetricCalls: 3,
		},
		{
			name:                "Collect filtered metrics with no regions",
			regions:             []string{},
			instanceIdentifiers: []string{"test-db-1"},
			regionManagerErrors: []error{},
			expectedError:       nil,
			expectedMetricCalls: 0,
		},
		{
			name:                "Collect filtered metrics with empty instance identifiers",
			regions:             []string{"us-west-2"},
			instanceIdentifiers: []string{},
			regionManagerErrors: []error{nil},
			expectedError:       nil,
			expectedMetricCalls: 1,
		},
		{
			name:                "Collect filtered metrics with first region error (fail fast)",
			regions:             []string{"us-west-2", "us-east-1"},
			instanceIdentifiers: []string{"test-db-1"},
			regionManagerErrors: []error{errors.New("first region failed")},
			expectedError:       errors.New("first region failed"),
			expectedMetricCalls: 1,
		},
		{
			name:                "collect filtered metrics with second region error",
			regions:             []string{"us-west-2", "us-east-1"},
			instanceIdentifiers: []string{"test-db-1"},
			regionManagerErrors: []error{nil, errors.New("second region failed")},
			expectedError:       errors.New("second region failed"),
			expectedMetricCalls: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewMultiRegionManager()

			var mockRMs []*mocks.MockRegionManager
			for i, region := range tc.regions {
				mockRM := &mocks.MockRegionManager{}

				if i < len(tc.regionManagerErrors) {
					if tc.expectedError != nil {
						// For error cases, use Maybe() since map iteration order is non-deterministic
						mockRM.On("CollectMetricsForInstances", mock.Anything, mock.Anything, mock.Anything).
							Return(tc.regionManagerErrors[i]).Maybe()
					} else {
						mockRM.On("CollectMetricsForInstances", mock.Anything, mock.Anything, mock.Anything).
							Return(tc.regionManagerErrors[i]).Once()
					}
				} else {
					// For regions beyond the error list, set up Maybe() expectation
					mockRM.On("CollectMetricsForInstances", mock.Anything, mock.Anything, mock.Anything).
						Return(nil).Maybe()
				}

				manager.AddRegionManager(region, mockRM)
				mockRMs = append(mockRMs, mockRM)
			}

			ch := make(chan prometheus.Metric, 100)
			err := manager.CollectMetricsForInstances(context.Background(), tc.instanceIdentifiers, ch)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			close(ch)

			for _, mockRM := range mockRMs {
				mockRM.AssertExpectations(t)
			}
		})
	}
}
