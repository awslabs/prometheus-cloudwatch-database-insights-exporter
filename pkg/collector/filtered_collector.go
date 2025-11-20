package collector

import (
	"context"
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/manager/region"
)

type FilteredCollector struct {
	regionManager  region.RegionManager
	instanceFilter []string
}

// FilteredCollector implements prometheus.Collector interface for targeted metric collection
// It provies the same functionality as Collector with instance-level filtering,
// allowing Prometheus to collect metrics from specific database instances rather than all discovered instances across all regions.
func NewFilteredCollector(regionManager region.RegionManager, instanceFilter []string) *FilteredCollector {
	return &FilteredCollector{
		regionManager:  regionManager,
		instanceFilter: instanceFilter,
	}
}

func (fc *FilteredCollector) Describe(ch chan<- *prometheus.Desc) {
	// Dynamic metrics are described during Collect()
}

// Collect gathers metrics from the specific instances and sends them to the provided channel.
// This method is invoked by Prometheus during metric scraping operations.
func (fc *FilteredCollector) Collect(ch chan<- prometheus.Metric) {
	log.Println("[FILTERED COLLECT] Collect() called - Prometheus is scraping")
	ctx := context.Background()

	err := fc.regionManager.CollectMetricsForInstances(ctx, fc.instanceFilter, ch)
	if err != nil {
		log.Println("[FILTERED COLLECT] Error collecting metrics:", err)
	}
}
