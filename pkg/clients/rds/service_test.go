package rds

import (
	"context"
	"errors"
	"testing"

	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils/mocks"
)

func TestDescribeDBInstancesPaginator(t *testing.T) {
	testCases := []struct {
		name          string
		mockResponse  []rdstypes.DBInstance
		expectedError error
		expectedCount int
	}{
		{
			name:          "describe instances with pagination",
			mockResponse:  mocks.NewMockRDSDescribeInstances(),
			expectedError: nil,
			expectedCount: 2,
		},
		{
			name:          "describe empty instances",
			mockResponse:  mocks.NewMockRDSDescribeInstancesEmpty(),
			expectedError: nil,
			expectedCount: 0,
		},
		{
			name:          "describe instances with error",
			mockResponse:  nil,
			expectedError: errors.New("Error describing instances"),
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := &mocks.MockRDSService{}
			mockService.On("DescribeDBInstancesPaginator", mock.Anything).Return(tc.mockResponse, tc.expectedError)

			instances, err := mockService.DescribeDBInstancesPaginator(context.Background())
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, instances)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, instances)
				assert.Len(t, instances, tc.expectedCount)
			}

			mockService.AssertExpectations(t)
		})
	}
}
