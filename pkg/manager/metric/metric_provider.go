package metric

import (
	"context"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricProvider interface {
	GetMetricBatches(ctx context.Context, instance models.Instance) ([][]string, error)
	CollectMetricsForBatch(ctx context.Context, instance models.Instance, metricsBatch []string, ch chan<- prometheus.Metric) error
}
