package region

import (
	"context"
	"sync"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/manager/instance"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/manager/metric"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
)

// instanceBatches holds the metric batches for a single instance
type instanceBatches struct {
	instance models.Instance
	batches  [][]string
	err      error
}

// metricRequest represents a single metric batch request for an instance
type metricRequest struct {
	instance     models.Instance
	metricsBatch []string
}

type SingleRegionManager struct {
	instanceManager instance.InstanceProvider
	metricManager   metric.MetricProvider
	region          string
	maxConcurrency  int
}

// SingleRegionManager handles the database metric collection within a single AWS region.
// It coordiantes between instance discovery (via RDS) and metric collection (via Performance Insights)
// to provide comprehensive database monitoring for all eligible instances in the region.
func NewSingleRegionManager(region string, instanceManager instance.InstanceProvider, metricManager metric.MetricProvider, concurrency int) *SingleRegionManager {
	return &SingleRegionManager{
		instanceManager: instanceManager,
		metricManager:   metricManager,
		region:          region,
		maxConcurrency:  concurrency,
	}
}

// CollectMetrics discovers and collects metrics from all eligible database instances in the region.
// This method discovers all Performance Insights enabled RDS database instances in the region,
// and collects available Performance Insights metrics on each instance using a queue-based worker pool
// to parallelize API calls across all metric batches from all instances.
func (singleRegionManager *SingleRegionManager) CollectMetrics(ctx context.Context, ch chan<- prometheus.Metric) error {
	instances, err := singleRegionManager.instanceManager.GetInstances(ctx)
	if err != nil {
		return err
	}

	return singleRegionManager.collectMetricsWithQueue(ctx, instances, ch)
}

// CollectMetricsForInstances discovers and collects metrics from all eligible and specified database instances in the region.
// This method discovers all Performance Insights enabled RDS database instances in the region that match the provided instance identifiers,
// and collects available Performance Insights metrics on each instance using a queue-based worker pool
// to parallelize API calls across all metric batches from all instances.
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

	return srm.collectMetricsWithQueue(ctx, filteredInstances, ch)
}

// fetchMetricBatchesInParallel fetches metric batches for all instances concurrently.
// This avoids the sequential API call bottleneck on first run when metrics aren't cached.
// Concurrency is limited by maxConcurrency to avoid overwhelming the API.
// Returns a slice of results containing instance, batches, and any errors encountered.
func (srm *SingleRegionManager) fetchMetricBatchesInParallel(ctx context.Context, instances []models.Instance) []instanceBatches {
	results := make([]instanceBatches, len(instances))
	var wg sync.WaitGroup

	// Semaphore to limit concurrent API calls
	semaphore := make(chan struct{}, srm.maxConcurrency)

	for i, inst := range instances {
		wg.Add(1)
		go func(index int, instance models.Instance) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }() // Release semaphore
			case <-ctx.Done():
				results[index] = instanceBatches{
					instance: instance,
					err:      ctx.Err(),
				}
				return
			}

			batches, err := srm.metricManager.GetMetricBatches(ctx, instance)
			results[index] = instanceBatches{
				instance: instance,
				batches:  batches,
				err:      err,
			}
		}(i, inst)
	}

	wg.Wait()
	return results
}

// collectMetricsWithQueue implements a queue-based worker pool pattern to parallelize
// metric data collection across all instances and their metric batches.
// This allows for better parallelization even when there's only a single instance with many metrics.
// Uses a bounded queue with producer goroutine to balance memory usage and performance.
// Continues processing on errors and collects all errors to report at the end.
func (srm *SingleRegionManager) collectMetricsWithQueue(ctx context.Context, instances []models.Instance, ch chan<- prometheus.Metric) error {
	// Fetch metric batches for all instances in parallel
	batchResults := srm.fetchMetricBatchesInParallel(ctx, instances)

	// Use a bounded queue to limit memory usage
	// Size = workers * 10 provides good balance between memory and throughput
	queueSize := srm.maxConcurrency * 10
	requestQueue := make(chan metricRequest, queueSize)

	// Error slice to collect all errors (protected by mutex)
	var errorsMu sync.Mutex
	var errors []error

	// WaitGroup for workers
	var workerWg sync.WaitGroup

	// Start worker pool
	for i := 0; i < srm.maxConcurrency; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for {
				select {
				case req, ok := <-requestQueue:
					if !ok {
						return // Channel closed
					}
					if err := srm.metricManager.CollectMetricsForBatch(ctx, req.instance, req.metricsBatch, ch); err != nil {
						errorsMu.Lock()
						errors = append(errors, err)
						errorsMu.Unlock()
					}
				case <-ctx.Done():
					return // Context cancelled - exit immediately
				}
			}
		}()
	}

	// Producer goroutine: feeds the queue from fetched batches
	var producerWg sync.WaitGroup
	producerWg.Add(1)
	go func() {
		defer producerWg.Done()
		defer close(requestQueue)

		for _, result := range batchResults {
			if result.err != nil {
				errorsMu.Lock()
				errors = append(errors, result.err)
				errorsMu.Unlock()
				continue
			}

			// Queue all batches for this instance
			for _, batch := range result.batches {
				select {
				case requestQueue <- metricRequest{
					instance:     result.instance,
					metricsBatch: batch,
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Wait for producer to finish
	producerWg.Wait()

	// Wait for all workers to complete
	workerWg.Wait()

	// Return the first error if any occurred
	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}
