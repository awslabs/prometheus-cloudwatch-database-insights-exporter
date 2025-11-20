package collector

import (
	"context"
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/manager/region"
)

type Collector struct {
	regionManager region.RegionManager
}

// Collector implements prometheus.Collector interface for collecting database insights metrics.
// It orchestrates metric collection across configured regions and database isntances,
// converting AWS Performance Insights data into Prometheus-compatible metrics.
func NewCollector(regionManager region.RegionManager) *Collector {
	return &Collector{
		regionManager: regionManager,
	}
}

func (collector *Collector) Describe(ch chan<- *prometheus.Desc) {
	// Dynamic metrics are described during Collect()
}

// Collect gathers metrics from all configured regions and sends them to the provided channel.
// This method is invoked by Prometheus during metric scraping operations.
func (collector *Collector) Collect(ch chan<- prometheus.Metric) {
	log.Println("[COLLECT] Collect() called - Prometheus is scraping")
	ctx := context.Background()

	err := collector.regionManager.CollectMetrics(ctx, ch)
	if err != nil {
		log.Println("[COLLECT] Error collecting metrics:", err)
	}
}
