package metric

import (
	"context"
	"fmt"
	"log"
	"time"

	awsPI "github.com/aws/aws-sdk-go-v2/service/pi"
	"github.com/aws/aws-sdk-go-v2/service/pi/types"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/clients/pi"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/processing/formatting"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/utils"
)

const (
	MaxRetries = 3
	BaseDelay  = time.Second
)

type MetricManager struct {
	piService     pi.PIService
	configuration *models.ParsedConfig
	registry      *utils.PerEngineMetricRegistry
}

// MetricManager handles Performance Insights metric collection and caching for database instances.
// It coordinates between metric discovery and data collection to provide comprehensive database performance monitoring with efficient AWS API usage.
func NewMetricManager(pi pi.PIService, config *models.ParsedConfig) (*MetricManager, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration parameter cannot be nil")
	}
	return &MetricManager{
		piService:     pi,
		configuration: config,
		registry:      utils.NewPerEngineMetricRegistry(),
	}, nil
}

// GetMetricBatches retrieves and batches the metrics for an instance without collecting data.
// This method is used by the queue-based worker pool to generate all metric batch requests upfront.
func (metricManager *MetricManager) GetMetricBatches(ctx context.Context, instance models.Instance) ([][]string, error) {
	metricsList, err := metricManager.getMetrics(ctx, instance.ResourceID, instance.Engine, instance.Metrics)
	if err != nil {
		return nil, err
	}

	return utils.BatchMetricNames(metricsList, utils.BatchSize), nil
}

// CollectMetricsForBatch collects metric data for a specific batch of metrics for an instance.
// This method is called by worker goroutines in the queue-based worker pool pattern.
func (metricManager *MetricManager) CollectMetricsForBatch(ctx context.Context, instance models.Instance, metricsBatch []string, ch chan<- prometheus.Metric) error {
	metricData, err := metricManager.getMetricData(ctx, instance.ResourceID, metricsBatch)
	if err != nil {
		log.Printf("[METRIC MANAGER] Error getting metric data for these metrics: %v, error: %v", metricsBatch, err)
		return err
	}

	for _, metricDatum := range metricData {
		if err := formatting.ConvertToPrometheusMetric(ch, instance, metricDatum, metricManager.configuration.Export.Prometheus.MetricPrefix); err != nil {
			log.Printf("[METRIC MANAGER] Error converting metric data to prometheus metric: %v, error: %v", metricDatum, err)
			continue
		}
	}

	return nil
}

func (metricManager *MetricManager) getMetrics(ctx context.Context, resourceID string, engine models.Engine, metrics *models.Metrics) ([]string, error) {
	if metrics == nil {
		return nil, fmt.Errorf("[METRIC MANAGER] Metrics not found for instance: %s", resourceID)
	}

	if metrics.MetricsDetails == nil || metrics.MetricsLastUpdated.IsZero() || time.Now().After(metrics.MetricsLastUpdated.Add(metrics.MetadataTTL)) {
		availableMetrics, err := metricManager.getAvailableMetrics(ctx, resourceID, engine)
		if err != nil {
			return nil, err
		}

		filteredMetrics := make(map[string]models.MetricDetails)
		metricConfig := metricManager.configuration.Discovery.Metrics
		for metricName, metric := range availableMetrics {
			if metricConfig.ShouldIncludeMetric(metric) {
				filteredMetrics[metricName] = metric
			}
		}

		filteredMetricList := utils.GetMetricNamesWithStatistic(filteredMetrics)

		metrics.MetricsDetails = filteredMetrics
		metrics.MetricsList = filteredMetricList
		metrics.MetricsLastUpdated = time.Now()
	}
	return metrics.MetricsList, nil
}

func (metricManager *MetricManager) getAvailableMetrics(ctx context.Context, resourceID string, engine models.Engine) (map[string]models.MetricDetails, error) {
	availableMetrics, err := utils.WithRetry(ctx, func() (*awsPI.ListAvailableResourceMetricsOutput, error) {
		return metricManager.piService.ListAvailableResourceMetrics(ctx, resourceID)
	}, MaxRetries, BaseDelay)
	if err != nil {
		return nil, err
	}

	return utils.BuildMetricDefinitionMap(availableMetrics.Metrics, &metricManager.configuration.Discovery.Metrics, engine, metricManager.registry)
}

func (metricManager *MetricManager) getMetricData(ctx context.Context, resourceID string, metricNamesWithStat []string) ([]models.MetricData, error) {
	metricDataResult, err := utils.WithRetry(ctx, func() (*awsPI.GetResourceMetricsOutput, error) {
		return metricManager.piService.GetResourceMetrics(ctx, resourceID, metricNamesWithStat)
	}, MaxRetries, BaseDelay)
	if err != nil {
		return nil, err
	}

	return metricManager.filterLatestValidMetricData(metricDataResult), nil
}

func (metricManager *MetricManager) filterLatestValidMetricData(result *awsPI.GetResourceMetricsOutput) []models.MetricData {
	var filteredData []models.MetricData

	for _, metricData := range result.MetricList {
		if metricData.Key == nil || metricData.Key.Metric == nil {
			continue
		}

		latestDataPoint := metricManager.getLatestValidDataPoint(metricData.DataPoints)
		if latestDataPoint != nil && latestDataPoint.Value != nil && latestDataPoint.Timestamp != nil {
			filteredData = append(filteredData, models.MetricData{
				Metric:    *metricData.Key.Metric,
				Timestamp: *latestDataPoint.Timestamp,
				Value:     *latestDataPoint.Value,
			})
		}
	}

	return filteredData
}

func (metricManager *MetricManager) getLatestValidDataPoint(dataPoints []types.DataPoint) *types.DataPoint {
	if len(dataPoints) == 0 {
		return nil
	}

	for i := len(dataPoints) - 1; i >= 0; i-- {
		dataPoint := &dataPoints[i]
		if dataPoint.Value != nil && dataPoint.Timestamp != nil {
			return dataPoint
		}
	}

	return nil
}
