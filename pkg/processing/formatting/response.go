package formatting

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
)

type Response interface {
	ConvertToPrometheusMetric(ch chan<- prometheus.Metric, instance models.Instance, metricData models.MetricData) error
}
