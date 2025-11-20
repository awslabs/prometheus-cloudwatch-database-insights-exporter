package region

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type MultiRegionManager struct {
	RegionManagers map[string]RegionManager
}

// MultiRegionManager orchestrates database metric collection across multiple AWS regions.
// It implements the RegionManager interface to provide a unified view of database instances and their metrics.
func NewMultiRegionManager() *MultiRegionManager {
	return &MultiRegionManager{
		RegionManagers: make(map[string]RegionManager),
	}
}

func (multiRegionManager *MultiRegionManager) AddRegionManager(region string, regionManager RegionManager) {
	multiRegionManager.RegionManagers[region] = regionManager
}

// CollectMetrics gathers metrics from all database instances across all configured regions.
// This method invokes CollectMetrics on each region manager.
func (multiRegionManager *MultiRegionManager) CollectMetrics(ctx context.Context, ch chan<- prometheus.Metric) error {
	for _, regionManager := range multiRegionManager.RegionManagers {
		err := regionManager.CollectMetrics(ctx, ch)
		if err != nil {
			return err
		}
	}

	return nil
}

// CollectMetricsForInstancesics gathers metrics from the specified database instances across all configured regions
// This method invokes CollectMetricsForInstancesics on each region manager.
func (multiRegionManager *MultiRegionManager) CollectMetricsForInstances(ctx context.Context, instanceIdentifiers []string, ch chan<- prometheus.Metric) error {
	for _, regionManager := range multiRegionManager.RegionManagers {
		err := regionManager.CollectMetricsForInstances(ctx, instanceIdentifiers, ch)
		if err != nil {
			return err
		}
	}

	return nil
}
