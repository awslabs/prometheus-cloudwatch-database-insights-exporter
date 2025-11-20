package metric

import (
	"context"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricProvider interface {
	CollectMetrics(ctx context.Context, instance models.Instance, ch chan<- prometheus.Metric) error
}
