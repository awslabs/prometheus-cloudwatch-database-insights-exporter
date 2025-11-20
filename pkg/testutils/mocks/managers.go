package mocks

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
)

type MockRegionManager struct {
	mock.Mock
}

func (mockRegionManager *MockRegionManager) CollectMetrics(ctx context.Context, ch chan<- prometheus.Metric) error {
	args := mockRegionManager.Called(ctx, ch)
	return args.Error(0)
}

func (m *MockRegionManager) CollectMetricsForInstances(ctx context.Context, instances []string, ch chan<- prometheus.Metric) error {
	args := m.Called(ctx, instances, ch)
	return args.Error(0)
}

type MockInstanceProvider struct {
	mock.Mock
}

func (mockInstanceProvider *MockInstanceProvider) GetInstances(ctx context.Context) ([]models.Instance, error) {
	args := mockInstanceProvider.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Instance), args.Error(1)
}

type MockMetricProvider struct {
	mock.Mock
}

func (mockMetricProvider *MockMetricProvider) CollectMetrics(ctx context.Context, instance models.Instance, ch chan<- prometheus.Metric) error {
	args := mockMetricProvider.Called(ctx, instance, ch)
	return args.Error(0)
}
