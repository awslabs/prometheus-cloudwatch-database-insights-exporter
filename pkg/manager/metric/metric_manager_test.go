package metric

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awspi "github.com/aws/aws-sdk-go-v2/service/pi"
	pitypes "github.com/aws/aws-sdk-go-v2/service/pi/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/clients/pi"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils/mocks"
)

func TestNewMetricManager(t *testing.T) {
	testCases := []struct {
		name          string
		mockPiService pi.PIService
	}{
		{
			name:          "Valid PI service",
			mockPiService: &mocks.MockPIService{},
		},
		{
			name:          "Nil PI service",
			mockPiService: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager, _ := NewMetricManager(tc.mockPiService, testutils.CreateDefaultParsedTestConfig())

			assert.NotNil(t, manager)
			assert.Equal(t, tc.mockPiService, manager.piService)
		})
	}
}

func TestGetMetricBatches(t *testing.T) {
	testCases := []struct {
		name             string
		instanceFactory  func() models.Instance
		mockListResponse *awspi.ListAvailableResourceMetricsOutput
		listError        error
		expectedError    error
		expectedBatches  int
		shouldCallList   bool
	}{
		{
			name:             "Get metric batches within MetricsTTL",
			instanceFactory:  testutils.NewTestInstancePostgreSQL,
			mockListResponse: nil,
			listError:        nil,
			expectedError:    nil,
			expectedBatches:  1,
			shouldCallList:   false,
		},
		{
			name:             "Get metric batches with expired MetricsTTL",
			instanceFactory:  testutils.NewTestInstancePostgreSQLExpired,
			mockListResponse: mocks.NewMockPIListMetricsResponse(),
			listError:        nil,
			expectedError:    nil,
			expectedBatches:  1,
			shouldCallList:   true,
		},
		{
			name:             "Get metric batches for no MetricsDetails",
			instanceFactory:  testutils.NewTestInstanceNoMetrics,
			mockListResponse: mocks.NewMockPIListMetricsResponse(),
			listError:        nil,
			expectedError:    nil,
			expectedBatches:  1,
			shouldCallList:   true,
		},
		{
			name:             "Get metric batches with ListAvailableResourceMetrics error",
			instanceFactory:  testutils.NewTestInstanceNoMetrics,
			mockListResponse: nil,
			listError:        errors.New("ListAvailableResourceMetrics failed"),
			expectedError:    errors.New("ListAvailableResourceMetrics failed"),
			expectedBatches:  0,
			shouldCallList:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance := tc.instanceFactory()

			mockPI := &mocks.MockPIService{}
			manager, _ := NewMetricManager(mockPI, testutils.CreateDefaultParsedTestConfig())

			if tc.shouldCallList {
				mockPI.On("ListAvailableResourceMetrics", mock.Anything, instance.ResourceID).
					Return(tc.mockListResponse, tc.listError)
			}

			batches, err := manager.GetMetricBatches(context.Background(), instance)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
				assert.Nil(t, batches)
			} else {
				assert.NoError(t, err)
				assert.Len(t, batches, tc.expectedBatches)
			}

			mockPI.AssertExpectations(t)
		})
	}
}

func TestCollectMetricsForBatch(t *testing.T) {
	testCases := []struct {
		name                string
		instanceFactory     func() models.Instance
		metricsBatch        []string
		mockGetResponse     *awspi.GetResourceMetricsOutput
		getError            error
		expectedError       error
		expectedMetricCount int
	}{
		{
			name:                "Collect metrics for batch success",
			instanceFactory:     testutils.NewTestInstancePostgreSQL,
			metricsBatch:        testutils.TestMetricNamesWithStats,
			mockGetResponse:     mocks.NewMockPIGetResourceMetricsResponse(),
			getError:            nil,
			expectedError:       nil,
			expectedMetricCount: 5,
		},
		{
			name:                "Collect metrics for batch with GetResourceMetrics error",
			instanceFactory:     testutils.NewTestInstancePostgreSQL,
			metricsBatch:        testutils.TestMetricNamesWithStats,
			mockGetResponse:     nil,
			getError:            errors.New("GetResourceMetrics failed"),
			expectedError:       errors.New("GetResourceMetrics failed"),
			expectedMetricCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance := tc.instanceFactory()

			mockPI := &mocks.MockPIService{}
			manager, _ := NewMetricManager(mockPI, testutils.CreateDefaultParsedTestConfig())

			mockPI.On("GetResourceMetrics", mock.Anything, instance.ResourceID, tc.metricsBatch).
				Return(tc.mockGetResponse, tc.getError)

			ch := make(chan prometheus.Metric, 100)

			err := manager.CollectMetricsForBatch(context.Background(), instance, tc.metricsBatch, ch)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			close(ch)

			metricCount := 0
			for range ch {
				metricCount++
			}
			assert.Equal(t, tc.expectedMetricCount, metricCount)

			mockPI.AssertExpectations(t)
		})
	}
}

func TestCollectMetricsForBatchWithEmptyResponse(t *testing.T) {
	testCases := []struct {
		name                string
		instanceFactory     func() models.Instance
		metricsBatch        []string
		mockGetResponse     *awspi.GetResourceMetricsOutput
		expectedMetricCount int
	}{
		{
			name:                "Collect metrics for batch with empty response",
			instanceFactory:     testutils.NewTestInstancePostgreSQL,
			metricsBatch:        testutils.TestMetricNamesWithStats,
			mockGetResponse:     mocks.NewMockPIGetResourceMetricsResponseEmpty(),
			expectedMetricCount: 0,
		},
		{
			name:                "Collect metrics for batch with nil keys",
			instanceFactory:     testutils.NewTestInstancePostgreSQL,
			metricsBatch:        testutils.TestMetricNamesWithStats,
			mockGetResponse:     mocks.NewMockPIGetResourceMetricsResponseWithNilKeys(),
			expectedMetricCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance := tc.instanceFactory()

			mockPI := &mocks.MockPIService{}
			manager, _ := NewMetricManager(mockPI, testutils.CreateDefaultParsedTestConfig())

			mockPI.On("GetResourceMetrics", mock.Anything, instance.ResourceID, tc.metricsBatch).
				Return(tc.mockGetResponse, nil)

			ch := make(chan prometheus.Metric, 100)

			err := manager.CollectMetricsForBatch(context.Background(), instance, tc.metricsBatch, ch)

			assert.NoError(t, err)

			close(ch)

			metricCount := 0
			for range ch {
				metricCount++
			}
			assert.Equal(t, tc.expectedMetricCount, metricCount)

			mockPI.AssertExpectations(t)
		})
	}
}

func TestGetMetricBatchesWithNilMetrics(t *testing.T) {
	instance := models.Instance{
		ResourceID: "db-TEST",
		Engine:     models.PostgreSQL,
		Metrics:    nil,
	}

	mockPI := &mocks.MockPIService{}
	manager, _ := NewMetricManager(mockPI, testutils.CreateDefaultParsedTestConfig())

	batches, err := manager.GetMetricBatches(context.Background(), instance)

	assert.Error(t, err)
	assert.Nil(t, batches)
	assert.Contains(t, err.Error(), "Metrics not found")
}

func TestGetMetrics(t *testing.T) {
	testCases := []struct {
		name          string
		resourceID    string
		metrics       *models.Metrics
		mockResponse  *awspi.ListAvailableResourceMetricsOutput
		expectedError error
		shouldCallAPI bool
	}{
		{
			name:          "Get metrics within TTL",
			resourceID:    testutils.TestInstancePostgreSQL.ResourceID,
			metrics:       testutils.TestInstancePostgreSQL.Metrics,
			mockResponse:  nil,
			expectedError: nil,
			shouldCallAPI: false,
		},
		{
			name:          "Get metrics with expired cache success",
			resourceID:    testutils.TestInstancePostgreSQLExpired.ResourceID,
			metrics:       testutils.TestInstancePostgreSQLExpired.Metrics,
			mockResponse:  mocks.NewMockPIListMetricsResponse(),
			expectedError: nil,
			shouldCallAPI: true,
		},
		{
			name:          "Get metrics with nil metrics pointer",
			resourceID:    "",
			metrics:       nil,
			mockResponse:  nil,
			expectedError: errors.New("Metrics not found"),
			shouldCallAPI: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockPI := &mocks.MockPIService{}
			manager, _ := NewMetricManager(mockPI, testutils.CreateDefaultParsedTestConfig())

			if tc.shouldCallAPI {
				mockPI.On("ListAvailableResourceMetrics", mock.Anything, tc.resourceID).
					Return(tc.mockResponse, tc.expectedError)
			}

			metricsList, err := manager.getMetrics(context.Background(), tc.resourceID, models.PostgreSQL, tc.metrics)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, metricsList)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metricsList)
			}

			mockPI.AssertExpectations(t)
		})
	}
}

func TestGetAvailableMetrics(t *testing.T) {
	testCases := []struct {
		name          string
		resourceID    string
		mockResponse  *awspi.ListAvailableResourceMetricsOutput
		expectedError error
		expectedCount int
	}{
		{
			name:          "Get available metrics",
			resourceID:    "db-TESTPOSTGRES",
			mockResponse:  mocks.NewMockPIListMetricsResponse(),
			expectedError: nil,
			expectedCount: 5,
		},
		{
			name:          "LisAvailableResourceMetrics error",
			resourceID:    "db-TESTPOSTGRES",
			mockResponse:  nil,
			expectedError: errors.New("LisAvailableResourceMetrics error"),
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockPI := &mocks.MockPIService{}
			manager, _ := NewMetricManager(mockPI, testutils.CreateDefaultParsedTestConfig())

			mockPI.On("ListAvailableResourceMetrics", mock.Anything, tc.resourceID).
				Return(tc.mockResponse, tc.expectedError)

			metricsDetails, err := manager.getAvailableMetrics(context.Background(), tc.resourceID, models.PostgreSQL)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, metricsDetails)
			} else {
				assert.NoError(t, err)
				assert.Len(t, metricsDetails, tc.expectedCount)

				for _, metric := range metricsDetails {
					assert.NotEmpty(t, metric.Name)
					assert.NotEmpty(t, metric.Description)
					assert.NotEmpty(t, metric.Unit)
					assert.NotEmpty(t, metric.Statistics)
				}
			}

			mockPI.AssertExpectations(t)
		})
	}
}

func TestGetMetricData(t *testing.T) {
	testCases := []struct {
		name          string
		resourceID    string
		metricNames   []string
		mockResponse  *awspi.GetResourceMetricsOutput
		expectedError error
		expectedCount int
	}{
		{
			name:          "Get metric data success",
			resourceID:    "db-TESTPOSTGRES",
			metricNames:   testutils.TestMetricNamesWithStats,
			mockResponse:  mocks.NewMockPIGetResourceMetricsResponse(),
			expectedError: nil,
			expectedCount: 5,
		},
		{
			name:          "Get metric data empty response",
			resourceID:    "db-TESTPOSTGRES",
			metricNames:   testutils.TestMetricNamesWithStats,
			mockResponse:  mocks.NewMockPIGetResourceMetricsResponseEmpty(),
			expectedError: nil,
			expectedCount: 0,
		},
		{
			name:          "GetResourceMetrics with error",
			resourceID:    "db-TESTPOSTGRES",
			metricNames:   testutils.TestMetricNamesWithStats,
			mockResponse:  nil,
			expectedError: errors.New("GetResourceMetrics error"),
			expectedCount: 0,
		},
		{
			name:          "Get metric data with nil keys",
			resourceID:    "db-TESTPOSTGRES",
			metricNames:   testutils.TestMetricNamesWithStats,
			mockResponse:  mocks.NewMockPIGetResourceMetricsResponseWithNilKeys(),
			expectedError: nil,
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockPI := &mocks.MockPIService{}
			manager, _ := NewMetricManager(mockPI, testutils.CreateDefaultParsedTestConfig())

			mockPI.On("GetResourceMetrics", mock.Anything, tc.resourceID, tc.metricNames).
				Return(tc.mockResponse, tc.expectedError)

			metricData, err := manager.getMetricData(context.Background(), tc.resourceID, tc.metricNames)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, metricData)
			} else {
				assert.NoError(t, err)
				assert.Len(t, metricData, tc.expectedCount)

				for _, data := range metricData {
					assert.NotEmpty(t, data.Metric)
					assert.NotZero(t, data.Timestamp)
					assert.NotZero(t, data.Value)
				}
			}

			mockPI.AssertExpectations(t)
		})
	}
}

func TestFilterLatestValidMetricData(t *testing.T) {
	testCases := []struct {
		name          string
		mockResponse  *awspi.GetResourceMetricsOutput
		expectedCount int
	}{
		{
			name:          "Filter latest valid data",
			mockResponse:  mocks.NewMockPIGetResourceMetricsResponse(),
			expectedCount: 5,
		},
		{
			name:          "Filter with empty response",
			mockResponse:  mocks.NewMockPIGetResourceMetricsResponseEmpty(),
			expectedCount: 0,
		},
		{
			name:          "Filter with nil keys",
			mockResponse:  mocks.NewMockPIGetResourceMetricsResponseWithNilKeys(),
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockPI := &mocks.MockPIService{}
			manager, _ := NewMetricManager(mockPI, testutils.CreateDefaultParsedTestConfig())

			filtered := manager.filterLatestValidMetricData(tc.mockResponse)

			assert.Len(t, filtered, tc.expectedCount)

			for _, data := range filtered {
				assert.NotEmpty(t, data.Metric)
				assert.NotZero(t, data.Timestamp)
				assert.NotZero(t, data.Value)
			}
		})
	}
}

func TestGetLatestValidDataPoint(t *testing.T) {
	testCases := []struct {
		name          string
		dataPoints    []pitypes.DataPoint
		expectedValue *float64
		expectNil     bool
	}{
		{
			name:          "empty DataPoints slice returns nil",
			dataPoints:    []pitypes.DataPoint{},
			expectedValue: nil,
			expectNil:     true,
		},
		{
			name: "single valid DataPoint",
			dataPoints: []pitypes.DataPoint{
				{
					Timestamp: aws.Time(testutils.TestTimestamp),
					Value:     aws.Float64(42.0),
				},
			},
			expectedValue: aws.Float64(42.0),
			expectNil:     false,
		},
		{
			name: "multiple DataPoints where last one is valid",
			dataPoints: []pitypes.DataPoint{
				{
					Timestamp: aws.Time(testutils.TestTimestamp),
					Value:     aws.Float64(10.0),
				},
				{
					Timestamp: aws.Time(testutils.TestTimestamp.Add(1 * time.Minute)),
					Value:     aws.Float64(20.0),
				},
				{
					Timestamp: aws.Time(testutils.TestTimestamp.Add(2 * time.Minute)),
					Value:     aws.Float64(30.0),
				},
			},
			expectedValue: aws.Float64(30.0),
			expectNil:     false,
		},
		{
			name: "multiple DataPoints where only middle one is valid",
			dataPoints: []pitypes.DataPoint{
				{
					Timestamp: nil,
					Value:     aws.Float64(10.0),
				},
				{
					Timestamp: aws.Time(testutils.TestTimestamp),
					Value:     aws.Float64(20.0),
				},
				{
					Timestamp: nil,
					Value:     aws.Float64(30.0),
				},
			},
			expectedValue: aws.Float64(20.0),
			expectNil:     false,
		},
		{
			name: "multiple DataPoints where only first one is valid (reverse iteration)",
			dataPoints: []pitypes.DataPoint{
				{
					Timestamp: aws.Time(testutils.TestTimestamp),
					Value:     aws.Float64(10.0),
				},
				{
					Timestamp: nil,
					Value:     aws.Float64(20.0),
				},
				{
					Timestamp: nil,
					Value:     aws.Float64(30.0),
				},
			},
			expectedValue: aws.Float64(10.0),
			expectNil:     false,
		},
		{
			name: "DataPoint with nil Value",
			dataPoints: []pitypes.DataPoint{
				{
					Timestamp: aws.Time(testutils.TestTimestamp),
					Value:     nil,
				},
			},
			expectedValue: nil,
			expectNil:     true,
		},
		{
			name: "DataPoint with nil Timestamp",
			dataPoints: []pitypes.DataPoint{
				{
					Timestamp: nil,
					Value:     aws.Float64(42.0),
				},
			},
			expectedValue: nil,
			expectNil:     true,
		},
		{
			name: "DataPoint with both nil Value and Timestamp",
			dataPoints: []pitypes.DataPoint{
				{
					Timestamp: nil,
					Value:     nil,
				},
			},
			expectedValue: nil,
			expectNil:     true,
		},
		{
			name: "all DataPoints have nil values returns nil",
			dataPoints: []pitypes.DataPoint{
				{
					Timestamp: aws.Time(testutils.TestTimestamp),
					Value:     nil,
				},
				{
					Timestamp: nil,
					Value:     aws.Float64(20.0),
				},
				{
					Timestamp: nil,
					Value:     nil,
				},
			},
			expectedValue: nil,
			expectNil:     true,
		},
		{
			name: "mix of valid and invalid DataPoints in chronological order",
			dataPoints: []pitypes.DataPoint{
				{
					Timestamp: nil,
					Value:     aws.Float64(10.0),
				},
				{
					Timestamp: aws.Time(testutils.TestTimestamp),
					Value:     aws.Float64(20.0),
				},
				{
					Timestamp: aws.Time(testutils.TestTimestamp.Add(1 * time.Minute)),
					Value:     nil,
				},
				{
					Timestamp: aws.Time(testutils.TestTimestamp.Add(2 * time.Minute)),
					Value:     aws.Float64(40.0),
				},
			},
			expectedValue: aws.Float64(40.0),
			expectNil:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockPI := &mocks.MockPIService{}
			manager, _ := NewMetricManager(mockPI, testutils.CreateDefaultParsedTestConfig())

			result := manager.getLatestValidDataPoint(tc.dataPoints)

			if tc.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.NotNil(t, result.Value)
				assert.NotNil(t, result.Timestamp)
				assert.Equal(t, *tc.expectedValue, *result.Value)
			}
		})
	}
}
