package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils/mocks"
)

func TestMetricsHandler(t *testing.T) {
	testCases := []struct {
		name               string
		queryParams        string
		expectedStatusCode int
		expectedInstances  []string
		regionManagerError error
	}{
		{
			name:               "GET /metrics without query params - all instances",
			queryParams:        "",
			expectedStatusCode: 200,
			expectedInstances:  nil,
			regionManagerError: nil,
		},
		{
			name:               "GET /metrics with single instance identifiers",
			queryParams:        "?identifiers=test-db-1",
			expectedStatusCode: 200,
			expectedInstances:  []string{"test-db-1"},
			regionManagerError: nil,
		},
		{
			name:               "GET /metrics with multiple instance identifiers (comma-separated)",
			queryParams:        "?identifiers=test-db-1,test-db-2",
			expectedStatusCode: 200,
			expectedInstances:  []string{"test-db-1", "test-db-2"},
			regionManagerError: nil,
		},
		{
			name:               "GET /metrics with empty identifiers parameter",
			queryParams:        "?identifiers=",
			expectedStatusCode: 200,
			expectedInstances:  nil,
			regionManagerError: nil,
		},
		{
			name:               "GET /metrics with region manager error - all instances",
			queryParams:        "",
			expectedStatusCode: 200,
			expectedInstances:  nil,
			regionManagerError: assert.AnError,
		},
		{
			name:               "GET /metrics with region manager error - filtered",
			queryParams:        "?identifiers=test-db-1",
			expectedStatusCode: 200,
			expectedInstances:  []string{"test-db-1"},
			regionManagerError: assert.AnError,
		},
		{
			name:               "GET /metrics with too many identifiers (exceeds limit)",
			queryParams:        "?identifiers=test-db-1,test-db-2,test-db-3,test-db-4,test-db-5,test-db-6",
			expectedStatusCode: 400,
			expectedInstances:  nil,
			regionManagerError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRM := &mocks.MockRegionManager{}

			if tc.expectedStatusCode == 200 {
				if tc.expectedInstances != nil {
					mockRM.On("CollectMetricsForInstances", mock.Anything, tc.expectedInstances, mock.Anything).
						Return(tc.regionManagerError)
				} else {
					mockRM.On("CollectMetrics", mock.Anything, mock.Anything).
						Return(tc.regionManagerError)
				}
			}

			req := httptest.NewRequest(http.MethodGet, "/metrics"+tc.queryParams, nil)
			recorder := httptest.NewRecorder()

			metricsHandler(recorder, req, mockRM)

			assert.Equal(t, tc.expectedStatusCode, recorder.Code)
			mockRM.AssertExpectations(t)
		})
	}
}
