package region

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type RegionManager interface {
	CollectMetrics(ctx context.Context, ch chan<- prometheus.Metric) error
	CollectMetricsForInstances(ctx context.Context, instanceIdentifiers []string, ch chan<- prometheus.Metric) error
}
