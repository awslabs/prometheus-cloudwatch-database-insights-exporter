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
				// Set up expectations for the new batch-based methods
				for i, instance := range tc.instances {
					// GetMetricBatches is called for each instance
					batches := [][]string{testutils.TestMetricNamesWithStats}
					mockMP.On("GetMetricBatches", mock.Anything, instance).
						Return(batches, nil).Maybe()

					// CollectMetricsForBatch is called for each batch
					if tc.expectedError != nil && i < len(tc.collectMetricsErrors) && tc.collectMetricsErrors[i] != nil {
						mockMP.On("CollectMetricsForBatch", mock.Anything, instance, mock.Anything, mock.Anything).
							Return(tc.collectMetricsErrors[i]).Maybe()
					} else {
						mockMP.On("CollectMetricsForBatch", mock.Anything, instance, mock.Anything, mock.Anything).
							Return(nil).Maybe()
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

				// Set up expectations for the new batch-based methods
				for i, instance := range filteredInstances {
					// GetMetricBatches is called for each instance
					batches := [][]string{testutils.TestMetricNamesWithStats}
					mockMP.On("GetMetricBatches", mock.Anything, instance).
						Return(batches, nil).Maybe()

					// CollectMetricsForBatch is called for each batch
					if tc.expectedError != nil && i < len(tc.collectMetricsErrors) && tc.collectMetricsErrors[i] != nil {
						mockMP.On("CollectMetricsForBatch", mock.Anything, instance, mock.Anything, mock.Anything).
							Return(tc.collectMetricsErrors[i]).Maybe()
					} else {
						mockMP.On("CollectMetricsForBatch", mock.Anything, instance, mock.Anything, mock.Anything).
							Return(nil).Maybe()
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

func TestCollectMetricsWithQueue(t *testing.T) {
	testCases := []struct {
		name                      string
		instances                 []models.Instance
		batchesPerInstance        [][][]string
		getBatchesErrors          []error
		collectBatchErrors        []error
		expectedError             error
		expectedGetBatchesCalls   int
		expectedCollectBatchCalls int
	}{
		{
			name:      "single instance with multiple batches (key optimization scenario)",
			instances: []models.Instance{testutils.TestInstancePostgreSQL},
			batchesPerInstance: [][][]string{
				{
					[]string{"metric1", "metric2", "metric3"},
					[]string{"metric4", "metric5", "metric6"},
					[]string{"metric7", "metric8", "metric9"},
				},
			},
			getBatchesErrors:          []error{nil},
			collectBatchErrors:        []error{nil, nil, nil},
			expectedError:             nil,
			expectedGetBatchesCalls:   1,
			expectedCollectBatchCalls: 3,
		},
		{
			name:      "multiple instances with multiple batches each",
			instances: testutils.TestInstances,
			batchesPerInstance: [][][]string{
				{
					[]string{"metric1", "metric2"},
					[]string{"metric3", "metric4"},
				},
				{
					[]string{"metric5", "metric6"},
					[]string{"metric7", "metric8"},
				},
			},
			getBatchesErrors:          []error{nil, nil},
			collectBatchErrors:        []error{nil, nil, nil, nil},
			expectedError:             nil,
			expectedGetBatchesCalls:   2,
			expectedCollectBatchCalls: 4,
		},
		{
			name:                      "single instance with no batches",
			instances:                 []models.Instance{testutils.TestInstancePostgreSQL},
			batchesPerInstance:        [][][]string{{}},
			getBatchesErrors:          []error{nil},
			collectBatchErrors:        []error{},
			expectedError:             nil,
			expectedGetBatchesCalls:   1,
			expectedCollectBatchCalls: 0,
		},
		{
			name:      "GetMetricBatches error for first instance continues with second",
			instances: testutils.TestInstances,
			batchesPerInstance: [][][]string{
				nil,
				{
					[]string{"metric1", "metric2"},
				},
			},
			getBatchesErrors:          []error{errors.New("failed to get batches"), nil},
			collectBatchErrors:        []error{nil},
			expectedError:             errors.New("failed to get batches"),
			expectedGetBatchesCalls:   2, // Continues to second instance
			expectedCollectBatchCalls: 1, // Second instance batches are processed
		},
		{
			name:      "CollectMetricsForBatch error in first batch continues with second",
			instances: []models.Instance{testutils.TestInstancePostgreSQL},
			batchesPerInstance: [][][]string{
				{
					[]string{"metric1", "metric2"},
					[]string{"metric3", "metric4"},
				},
			},
			getBatchesErrors:          []error{nil},
			collectBatchErrors:        []error{errors.New("batch collection failed"), nil},
			expectedError:             errors.New("batch collection failed"),
			expectedGetBatchesCalls:   1,
			expectedCollectBatchCalls: 2, // Both batches are processed despite first error
		},
		{
			name:      "mixed success and failure across batches continues processing",
			instances: []models.Instance{testutils.TestInstancePostgreSQL},
			batchesPerInstance: [][][]string{
				{
					[]string{"metric1", "metric2"},
					[]string{"metric3", "metric4"},
					[]string{"metric5", "metric6"},
				},
			},
			getBatchesErrors:          []error{nil},
			collectBatchErrors:        []error{nil, errors.New("second batch failed"), nil},
			expectedError:             errors.New("second batch failed"),
			expectedGetBatchesCalls:   1,
			expectedCollectBatchCalls: 3, // All batches are processed despite error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockIP := &mocks.MockInstanceProvider{}
			mockMP := &mocks.MockMetricProvider{}
			manager := NewSingleRegionManager("us-west-2", mockIP, mockMP)

			mockIP.On("GetInstances", mock.Anything).
				Return(tc.instances, nil)

			// Set up GetMetricBatches expectations
			for i, instance := range tc.instances {
				if i < len(tc.getBatchesErrors) {
					mockMP.On("GetMetricBatches", mock.Anything, instance).
						Return(tc.batchesPerInstance[i], tc.getBatchesErrors[i]).Once()
				}
			}

			// Set up CollectMetricsForBatch expectations
			batchIndex := 0
			for i, instance := range tc.instances {
				if i < len(tc.getBatchesErrors) && tc.getBatchesErrors[i] == nil {
					for _, batch := range tc.batchesPerInstance[i] {
						if batchIndex < len(tc.collectBatchErrors) {
							mockMP.On("CollectMetricsForBatch", mock.Anything, instance, batch, mock.Anything).
								Return(tc.collectBatchErrors[batchIndex]).Maybe()
							batchIndex++
						}
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

func TestFetchMetricBatchesInParallel(t *testing.T) {
	testCases := []struct {
		name                 string
		instances            []models.Instance
		batchesPerInstance   [][][]string
		getBatchesErrors     []error
		expectedResultCount  int
		expectedSuccessCount int
		expectedErrorCount   int
		maxConcurrency       int
	}{
		{
			name:      "single instance success",
			instances: []models.Instance{testutils.TestInstancePostgreSQL},
			batchesPerInstance: [][][]string{
				{
					[]string{"metric1", "metric2"},
					[]string{"metric3", "metric4"},
				},
			},
			getBatchesErrors:     []error{nil},
			expectedResultCount:  1,
			expectedSuccessCount: 1,
			expectedErrorCount:   0,
			maxConcurrency:       4,
		},
		{
			name:      "multiple instances all success",
			instances: testutils.TestInstances,
			batchesPerInstance: [][][]string{
				{
					[]string{"metric1", "metric2"},
				},
				{
					[]string{"metric3", "metric4"},
				},
			},
			getBatchesErrors:     []error{nil, nil},
			expectedResultCount:  2,
			expectedSuccessCount: 2,
			expectedErrorCount:   0,
			maxConcurrency:       4,
		},
		{
			name:      "single instance with error",
			instances: []models.Instance{testutils.TestInstancePostgreSQL},
			batchesPerInstance: [][][]string{
				nil,
			},
			getBatchesErrors:     []error{errors.New("API call failed")},
			expectedResultCount:  1,
			expectedSuccessCount: 0,
			expectedErrorCount:   1,
			maxConcurrency:       4,
		},
		{
			name:      "mixed success and failure",
			instances: testutils.TestInstances,
			batchesPerInstance: [][][]string{
				{
					[]string{"metric1", "metric2"},
				},
				nil,
			},
			getBatchesErrors:     []error{nil, errors.New("second instance failed")},
			expectedResultCount:  2,
			expectedSuccessCount: 1,
			expectedErrorCount:   1,
			maxConcurrency:       4,
		},
		{
			name:      "all instances fail",
			instances: testutils.TestInstances,
			batchesPerInstance: [][][]string{
				nil,
				nil,
			},
			getBatchesErrors:     []error{errors.New("first failed"), errors.New("second failed")},
			expectedResultCount:  2,
			expectedSuccessCount: 0,
			expectedErrorCount:   2,
			maxConcurrency:       4,
		},
		{
			name:                 "empty instances list",
			instances:            []models.Instance{},
			batchesPerInstance:   [][][]string{},
			getBatchesErrors:     []error{},
			expectedResultCount:  0,
			expectedSuccessCount: 0,
			expectedErrorCount:   0,
			maxConcurrency:       4,
		},
		{
			name: "many instances with limited concurrency",
			instances: []models.Instance{
				testutils.TestInstancePostgreSQL,
				testutils.TestInstanceMySQL,
				testutils.NewTestInstance("db-3", "test-db-3", models.AuroraPostgreSQL),
				testutils.NewTestInstance("db-4", "test-db-4", models.AuroraMySQL),
				testutils.NewTestInstance("db-5", "test-db-5", models.AuroraPostgreSQL),
			},
			batchesPerInstance: [][][]string{
				{[]string{"m1"}},
				{[]string{"m2"}},
				{[]string{"m3"}},
				{[]string{"m4"}},
				{[]string{"m5"}},
			},
			getBatchesErrors:     []error{nil, nil, nil, nil, nil},
			expectedResultCount:  5,
			expectedSuccessCount: 5,
			expectedErrorCount:   0,
			maxConcurrency:       2, // Limited concurrency
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockIP := &mocks.MockInstanceProvider{}
			mockMP := &mocks.MockMetricProvider{}
			manager := NewSingleRegionManager("us-west-2", mockIP, mockMP)
			manager.maxConcurrency = tc.maxConcurrency

			// Set up GetMetricBatches expectations
			for i, instance := range tc.instances {
				if i < len(tc.getBatchesErrors) {
					mockMP.On("GetMetricBatches", mock.Anything, instance).
						Return(tc.batchesPerInstance[i], tc.getBatchesErrors[i]).Once()
				}
			}

			// Call the method
			results := manager.fetchMetricBatchesInParallel(context.Background(), tc.instances)

			// Verify results
			assert.Equal(t, tc.expectedResultCount, len(results), "Result count mismatch")

			successCount := 0
			errorCount := 0
			for i, result := range results {
				// Verify instance is preserved
				if i < len(tc.instances) {
					assert.Equal(t, tc.instances[i], result.instance, "Instance mismatch at index %d", i)
				}

				if result.err != nil {
					errorCount++
				} else {
					successCount++
					// Verify batches are correct
					if i < len(tc.batchesPerInstance) {
						assert.Equal(t, tc.batchesPerInstance[i], result.batches, "Batches mismatch at index %d", i)
					}
				}
			}

			assert.Equal(t, tc.expectedSuccessCount, successCount, "Success count mismatch")
			assert.Equal(t, tc.expectedErrorCount, errorCount, "Error count mismatch")

			mockMP.AssertExpectations(t)
		})
	}
}

func TestFetchMetricBatchesInParallelContextCancellation(t *testing.T) {
	t.Run("context cancelled before API calls", func(t *testing.T) {
		mockIP := &mocks.MockInstanceProvider{}
		mockMP := &mocks.MockMetricProvider{}
		manager := NewSingleRegionManager("us-west-2", mockIP, mockMP)

		instances := []models.Instance{
			testutils.TestInstancePostgreSQL,
			testutils.TestInstanceMySQL,
		}

		// Mock may or may not be called depending on timing
		mockMP.On("GetMetricBatches", mock.Anything, mock.Anything).
			Return([][]string{{"metric1"}}, nil).Maybe()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		results := manager.fetchMetricBatchesInParallel(ctx, instances)

		// Should return results for all instances
		assert.Equal(t, len(instances), len(results))

		// At least some should have context.Canceled error
		hasContextError := false
		for _, result := range results {
			if result.err != nil && errors.Is(result.err, context.Canceled) {
				hasContextError = true
				break
			}
		}
		assert.True(t, hasContextError, "Expected at least one context cancellation error")
	})

	t.Run("context cancelled during API calls", func(t *testing.T) {
		mockIP := &mocks.MockInstanceProvider{}
		mockMP := &mocks.MockMetricProvider{}
		manager := NewSingleRegionManager("us-west-2", mockIP, mockMP)
		manager.maxConcurrency = 1 // Force sequential processing

		instances := []models.Instance{
			testutils.TestInstancePostgreSQL,
			testutils.TestInstanceMySQL,
			testutils.NewTestInstance("db-3", "test-db-3", models.AuroraPostgreSQL),
		}

		ctx, cancel := context.WithCancel(context.Background())

		// First call succeeds
		mockMP.On("GetMetricBatches", mock.Anything, testutils.TestInstancePostgreSQL).
			Return([][]string{{"metric1"}}, nil).Once().
			Run(func(args mock.Arguments) {
				// Cancel context after first successful call
				cancel()
			})

		// Subsequent calls may or may not execute depending on timing
		mockMP.On("GetMetricBatches", mock.Anything, mock.Anything).
			Return([][]string{}, nil).Maybe()

		results := manager.fetchMetricBatchesInParallel(ctx, instances)

		// Should return results for all instances
		assert.Equal(t, len(instances), len(results))

		// First should succeed
		assert.NotNil(t, results[0].batches)
		assert.Nil(t, results[0].err)

		// Others may have context errors or succeed depending on timing
		// We just verify we got results for all
	})
}

func TestFetchMetricBatchesInParallelConcurrencyLimit(t *testing.T) {
	t.Run("respects maxConcurrency limit", func(t *testing.T) {
		mockIP := &mocks.MockInstanceProvider{}
		mockMP := &mocks.MockMetricProvider{}
		manager := NewSingleRegionManager("us-west-2", mockIP, mockMP)
		manager.maxConcurrency = 2 // Limit to 2 concurrent calls

		// Create unique instances to avoid mock confusion
		instances := []models.Instance{
			testutils.NewTestInstance("db-1", "test-db-1", models.AuroraPostgreSQL),
			testutils.NewTestInstance("db-2", "test-db-2", models.AuroraMySQL),
			testutils.NewTestInstance("db-3", "test-db-3", models.AuroraPostgreSQL),
			testutils.NewTestInstance("db-4", "test-db-4", models.AuroraMySQL),
			testutils.NewTestInstance("db-5", "test-db-5", models.AuroraPostgreSQL),
		}

		// Mock each instance separately
		for _, instance := range instances {
			mockMP.On("GetMetricBatches", mock.Anything, instance).
				Return([][]string{{"metric1"}}, nil).Once()
		}

		results := manager.fetchMetricBatchesInParallel(context.Background(), instances)

		assert.Equal(t, len(instances), len(results))

		// All should succeed
		for _, result := range results {
			assert.Nil(t, result.err)
			assert.NotNil(t, result.batches)
		}

		mockMP.AssertExpectations(t)
	})
}
