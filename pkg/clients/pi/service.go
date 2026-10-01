package pi

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/pi"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
)

type PIService interface {
	ListAvailableResourceMetrics(ctx context.Context, resourceID string) (*pi.ListAvailableResourceMetricsOutput, error)
	GetResourceMetrics(ctx context.Context, resourceID string, metricNames []string, dimensionGroups map[string]*models.ParsedDimensionGroup) (*pi.GetResourceMetricsOutput, error)
}
