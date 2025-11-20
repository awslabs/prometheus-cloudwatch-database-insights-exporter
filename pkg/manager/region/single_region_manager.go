package region

import (
	"context"
	"sync"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/manager/instance"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/manager/metric"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
)

// DefaultMaxConcurrency defines the default maximum number of concurrent metric collection requests
const DefaultMaxConcurrency = 4

type SingleRegionManager struct {
	instanceManager instance.InstanceProvider
	metricManager   metric.MetricProvider
	region          string
	maxConcurrency  int
}

// SingleRegionManager handles the database metric collection within a single AWS region.
// It coordiantes between instance discovery (via RDS) and metric collection (via Performance Insights)
// to provide comprehensive database monitoring for all eligible instances in the region.
func NewSingleRegionManager(region string, instanceManager instance.InstanceProvider, metricManager metric.MetricProvider) *SingleRegionManager {
	return &SingleRegionManager{
		instanceManager: instanceManager,
		metricManager:   metricManager,
		region:          region,
		maxConcurrency:  DefaultMaxConcurrency,
	}
}

// CollectMetrics discovers and collects metrics from all eligible database instances in the region.
// This method discovers all Performance Insights enabled RDS database instances in the region,
// and collects available Performance Insights metrics on each instance.
func (singleRegionManager *SingleRegionManager) CollectMetrics(ctx context.Context, ch chan<- prometheus.Metric) error {
	instances, err := singleRegionManager.instanceManager.GetInstances(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(instances))
	semaphore := make(chan struct{}, singleRegionManager.maxConcurrency)

	for _, instance := range instances {
		wg.Add(1)
		go func(inst models.Instance) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release semaphore

			if err := singleRegionManager.metricManager.CollectMetrics(ctx, inst, ch); err != nil {
				errChan <- err
			}
		}(instance)
	}

	wg.Wait()
	close(errChan)

	// Return the first error if any occurred
	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}

// CollectMetricsForInstances discovers and collects metrics from all eligible and specified database instances in the region.
// This method discovers all Performance Insights enabled RDS database instances in the region that match the provided instance identifiers,
// and collects available Performance Insights metrics on each instance.
func (srm *SingleRegionManager) CollectMetricsForInstances(ctx context.Context, instanceIdentifiers []string, ch chan<- prometheus.Metric) error {
	allInstances, err := srm.instanceManager.GetInstances(ctx)
	if err != nil {
		return err
	}

	identifierMap := make(map[string]models.Instance, len(instanceIdentifiers))
	for _, identifier := range instanceIdentifiers {
		identifierMap[identifier] = models.Instance{}
	}

	filteredInstances := make([]models.Instance, 0, len(instanceIdentifiers))
	for _, instance := range allInstances {
		if _, exists := identifierMap[instance.Identifier]; exists {
			filteredInstances = append(filteredInstances, instance)
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(filteredInstances))
	semaphore := make(chan struct{}, srm.maxConcurrency)

	for _, instance := range filteredInstances {
		wg.Add(1)
		go func(inst models.Instance) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release semaphore

			if err := srm.metricManager.CollectMetrics(ctx, inst, ch); err != nil {
				errChan <- err
			}
		}(instance)
	}

	wg.Wait()
	close(errChan)

	// Return the first error if any occurred
	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}
