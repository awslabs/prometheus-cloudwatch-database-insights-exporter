package mocks

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/pi"
	"github.com/stretchr/testify/mock"
)

type MockAWSPIClient struct {
	mock.Mock
}

func (mockPIClient *MockAWSPIClient) ListAvailableResourceMetrics(ctx context.Context, params *pi.ListAvailableResourceMetricsInput, optFns ...func(*pi.Options)) (*pi.ListAvailableResourceMetricsOutput, error) {
	args := mockPIClient.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pi.ListAvailableResourceMetricsOutput), args.Error(1)
}

func (mockPIClient *MockAWSPIClient) GetResourceMetrics(ctx context.Context, params *pi.GetResourceMetricsInput, optFns ...func(*pi.Options)) (*pi.GetResourceMetricsOutput, error) {
	args := mockPIClient.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pi.GetResourceMetricsOutput), args.Error(1)
}
