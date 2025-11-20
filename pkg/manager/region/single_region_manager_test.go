package region

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils/mocks"
)

func TestNewSingleRegionManager(t *testing.T) {
	t.Run("creates new single region manager successfully", func(t *testing.T) {
		mockInstanceProvider := &mocks.MockInstanceProvider{}
		mockMetricProvider := &mocks.MockMetricProvider{}
		region := "us-west-2"

		manager := NewSingleRegionManager(region, mockInstanceProvider, mockMetricProvider)

		assert.NotNil(t, manager)
		assert.Equal(t, region, manager.region)
		assert.Equal(t, mockInstanceProvider, manager.instanceManager)
		assert.Equal(t, mockMetricProvider, manager.metricManager)
	})
}

func TestCollectMetrics(t *testing.T) {
	testCases := []struct {
		name                   string
		instances              []models.Instance
		getInstancesError      error
		collectMetricsErrors   []error
		expectedError          error
		expectedMetricCalls    int
		shouldCallGetInstances bool
	}{
		{
			name:                   "collect metrics success with multiple instances",
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{nil, nil},
			expectedError:          nil,
			expectedMetricCalls:    2,
			shouldCallGetInstances: true,
		},
		{
			name:                   "collect metrics success with single instance",
			instances:              []models.Instance{testutils.TestInstancePostgreSQL},
			getInstancesError:      nil,
			collectMetricsErrors:   []error{nil},
			expectedError:          nil,
			expectedMetricCalls:    1,
			shouldCallGetInstances: true,
		},
		{
			name:                   "collect metrics success with no instances",
			instances:              []models.Instance{},
			getInstancesError:      nil,
			collectMetricsErrors:   []error{},
			expectedError:          nil,
			expectedMetricCalls:    0,
			shouldCallGetInstances: true,
		},
		{
			name:                   "collect metrics with get instances error",
			instances:              nil,
			getInstancesError:      errors.New("failed to get instances"),
			collectMetricsErrors:   []error{},
			expectedError:          errors.New("failed to get instances"),
			expectedMetricCalls:    0,
			shouldCallGetInstances: true,
		},
		{
			name:                   "collect metrics with first instance error (fail fast)",
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{errors.New("metric collection failed"), nil},
			expectedError:          errors.New("metric collection failed"),
			expectedMetricCalls:    1,
			shouldCallGetInstances: true,
		},
		{
			name:                   "collect metrics with second instance error",
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{nil, errors.New("second instance failed")},
			expectedError:          errors.New("second instance failed"),
			expectedMetricCalls:    2,
			shouldCallGetInstances: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockIP := &mocks.MockInstanceProvider{}
			mockMP := &mocks.MockMetricProvider{}
			manager := NewSingleRegionManager("us-west-2", mockIP, mockMP)

			if tc.shouldCallGetInstances {
				mockIP.On("GetInstances", mock.Anything).
					Return(tc.instances, tc.getInstancesError)
			}

			if tc.getInstancesError == nil && tc.instances != nil {
				// For parallel execution, we can't guarantee order, so set up expectations differently
				if tc.expectedError != nil {
					// If we expect an error, at least one call should return an error
					// Use Maybe() to allow any instance to be called
					for i, instance := range tc.instances {
						if i < len(tc.collectMetricsErrors) {
							mockMP.On("CollectMetrics", mock.Anything, instance, mock.Anything).
								Return(tc.collectMetricsErrors[i]).Maybe()
						}
					}
				} else {
					// For success cases, all instances should be called exactly once
					for _, instance := range tc.instances {
						mockMP.On("CollectMetrics", mock.Anything, instance, mock.Anything).
							Return(nil).Once()
					}
				}
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

			mockIP.AssertExpectations(t)
			mockMP.AssertExpectations(t)
		})
	}
}

func TestCollectMetricsForInstances(t *testing.T) {
	testCases := []struct {
		name                   string
		instanceIdentifiers    []string
		instances              []models.Instance
		getInstancesError      error
		collectMetricsErrors   []error
		expectedError          error
		expectedMetricCalls    int
		shouldCallGetInstances bool
	}{
		{
			name:                   "filter matches single instance",
			instanceIdentifiers:    []string{"test-postgres-db"},
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{nil},
			expectedError:          nil,
			expectedMetricCalls:    1,
			shouldCallGetInstances: true,
		},
		{
			name:                   "filter matches multiple instances",
			instanceIdentifiers:    []string{"test-postgres-db", "test-mysql-db"},
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{nil, nil},
			expectedError:          nil,
			expectedMetricCalls:    2,
			shouldCallGetInstances: true,
		},
		{
			name:                   "filter matches no instances (empty filtered list)",
			instanceIdentifiers:    []string{"non-existent-db"},
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{},
			expectedError:          nil,
			expectedMetricCalls:    0,
			shouldCallGetInstances: true,
		},
		{
			name:                   "empty instanceIdentifiers array",
			instanceIdentifiers:    []string{},
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{},
			expectedError:          nil,
			expectedMetricCalls:    0,
			shouldCallGetInstances: true,
		},
		{
			name:                   "instance identifiers with non-existent IDs",
			instanceIdentifiers:    []string{"test-postgres-db", "non-existent-db", "another-missing-db"},
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{nil},
			expectedError:          nil,
			expectedMetricCalls:    1,
			shouldCallGetInstances: true,
		},
		{
			name:                   "GetInstances returns error",
			instanceIdentifiers:    []string{"test-postgres-db"},
			instances:              nil,
			getInstancesError:      errors.New("failed to get instances"),
			collectMetricsErrors:   []error{},
			expectedError:          errors.New("failed to get instances"),
			expectedMetricCalls:    0,
			shouldCallGetInstances: true,
		},
		{
			name:                   "successful collection for all filtered instances",
			instanceIdentifiers:    []string{"test-mysql-db"},
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{nil},
			expectedError:          nil,
			expectedMetricCalls:    1,
			shouldCallGetInstances: true,
		},
		{
			name:                   "error during metric collection for first filtered instance (fail fast)",
			instanceIdentifiers:    []string{"test-postgres-db", "test-mysql-db"},
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{errors.New("metric collection failed"), nil},
			expectedError:          errors.New("metric collection failed"),
			expectedMetricCalls:    1,
			shouldCallGetInstances: true,
		},
		{
			name:                   "error during metric collection for second filtered instance",
			instanceIdentifiers:    []string{"test-postgres-db", "test-mysql-db"},
			instances:              testutils.TestInstances,
			getInstancesError:      nil,
			collectMetricsErrors:   []error{nil, errors.New("second instance failed")},
			expectedError:          errors.New("second instance failed"),
			expectedMetricCalls:    2,
			shouldCallGetInstances: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockIP := &mocks.MockInstanceProvider{}
			mockMP := &mocks.MockMetricProvider{}
			manager := NewSingleRegionManager("us-west-2", mockIP, mockMP)

			if tc.shouldCallGetInstances {
				mockIP.On("GetInstances", mock.Anything).
					Return(tc.instances, tc.getInstancesError)
			}

			if tc.getInstancesError == nil && tc.instances != nil {
				var filteredInstances []models.Instance
				for _, instance := range tc.instances {
					for _, identifier := range tc.instanceIdentifiers {
						if instance.Identifier == identifier {
							filteredInstances = append(filteredInstances, instance)
							break
						}
					}
				}

				// For parallel execution, we can't guarantee order, so set up expectations differently
				if tc.expectedError != nil {
					// If we expect an error, at least one call should return an error
					// Use Maybe() to allow any instance to be called
					for i, instance := range filteredInstances {
						if i < len(tc.collectMetricsErrors) {
							mockMP.On("CollectMetrics", mock.Anything, instance, mock.Anything).
								Return(tc.collectMetricsErrors[i]).Maybe()
						}
					}
				} else {
					// For success cases, all instances should be called exactly once
					for _, instance := range filteredInstances {
						mockMP.On("CollectMetrics", mock.Anything, instance, mock.Anything).
							Return(nil).Once()
					}
				}
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

			mockIP.AssertExpectations(t)
			mockMP.AssertExpectations(t)
		})
	}
}
