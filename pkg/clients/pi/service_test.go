package pi

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/pi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils/mocks"
)

func TestListAvailableResourceMetrics(t *testing.T) {
	testCases := []struct {
		name          string
		resourceID    string
		mockResponse  *pi.ListAvailableResourceMetricsOutput
		expectedError error
		expectedCount int
	}{
		{
			name:          "list metrics",
			resourceID:    "db-TESTPOSTGRES",
			mockResponse:  mocks.NewMockPIListMetricsResponse(),
			expectedError: nil,
			expectedCount: 5,
		},
		{
			name:          "list empty",
			resourceID:    "db-TESTPOSTGRES",
			mockResponse:  mocks.NewMockPIListMetricsResponseEmpty(),
			expectedError: nil,
			expectedCount: 0,
		},
		{
			name:          "list error",
			resourceID:    "db-TESTPOSTGRES",
			mockResponse:  nil,
			expectedError: errors.New("Error listing available metrics"),
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := &mocks.MockPIService{}
			mockService.On("ListAvailableResourceMetrics", mock.Anything, tc.resourceID).Return(tc.mockResponse, tc.expectedError)

			result, err := mockService.ListAvailableResourceMetrics(context.Background(), tc.resourceID)
			if tc.expectedError != nil {
				assert.Nil(t, result)
				assert.Error(t, err)
			} else {
				assert.NotNil(t, result)
				assert.NoError(t, err)
				assert.Len(t, result.Metrics, tc.expectedCount)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestGetResourceMetrics(t *testing.T) {
	testCases := []struct {
		name          string
		resourceID    string
		metricNames   []string
		mockResponse  *pi.GetResourceMetricsOutput
		expectedError error
		expectedCount int
	}{
		{
			name:          "get metrics",
			resourceID:    "db-TESTPOSTGRES",
			metricNames:   testutils.TestMetricNamesWithStats,
			mockResponse:  mocks.NewMockPIGetResourceMetricsResponse(),
			expectedError: nil,
			expectedCount: 5,
		},
		{
			name:          "get empty",
			resourceID:    "db-TESTPOSTGRES",
			metricNames:   testutils.TestMetricNamesEmpty,
			mockResponse:  mocks.NewMockPIGetResourceMetricsResponseEmpty(),
			expectedError: nil,
			expectedCount: 0,
		},
		{
			name:          "get error",
			metricNames:   testutils.TestMetricNamesWithStats,
			resourceID:    "db-TESTPOSTGRES",
			mockResponse:  nil,
			expectedError: errors.New("Error getting metrics"),
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := &mocks.MockPIService{}
			mockService.On("GetResourceMetrics", mock.Anything, tc.resourceID, tc.metricNames).Return(tc.mockResponse, tc.expectedError)

			result, err := mockService.GetResourceMetrics(context.Background(), tc.resourceID, tc.metricNames)
			if tc.expectedError != nil {
				assert.Nil(t, result)
				assert.Error(t, err)
			} else {
				assert.NotNil(t, result)
				assert.NoError(t, err)
				assert.Len(t, result.MetricList, tc.expectedCount)
			}

			mockService.AssertExpectations(t)
		})
	}
}
