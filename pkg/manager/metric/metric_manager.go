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
	piService pi.PIService
}

// MetricManager handles Performance Insights metric collection and caching for database instances.
// It coordinates between metric discovery and data collection to provide comprehensive database performance monitoring with efficient AWS API usage.
func NewMetricManager(pi pi.PIService) *MetricManager {
	return &MetricManager{
		piService: pi,
	}
}

// CollectMetrics orchestrates complete metric collection for a database instance.
// This method retrieves or discovers available metrics for the instance and collects current metric data values via the Performance Insights API with batching optimization.
// It converts collected data to Prometheus format and sends the metrics to the provided channel.
func (metricManager *MetricManager) CollectMetrics(ctx context.Context, instance models.Instance, ch chan<- prometheus.Metric) error {
	metricsList, err := metricManager.getMetrics(ctx, instance.ResourceID, instance.Engine, instance.Metrics)
	if err != nil {
		return err
	}

	metricsBatches := utils.BatchMetricNames(metricsList)

	var data []models.MetricData
	for _, metricsBatch := range metricsBatches {
		metricData, err := metricManager.getMetricData(ctx, instance.ResourceID, metricsBatch)
		if err != nil {
			log.Printf("[METRIC MANAGER] Error getting metric data for these metrics: %v, error: %v", metricsBatch, err)
			return err
		}
		data = append(data, metricData...)
	}

	for _, metricDatum := range data {
		if err := formatting.ConvertToPrometheusMetric(ch, instance, metricDatum); err != nil {
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

	if metrics.MetricsDetails == nil || metrics.MetricsLastUpdated.IsZero() || time.Now().After(metrics.MetricsLastUpdated.Add(metrics.MetricsTTL)) {
		availableMetrics, err := metricManager.getAvailableMetrics(ctx, resourceID, engine)
		if err != nil {
			return nil, err
		}

		metrics.MetricsDetails = availableMetrics
		metrics.MetricsList = utils.GetMetricNamesWithStatistic(availableMetrics)
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

	return utils.BuildMetricDefinitionMap(availableMetrics.Metrics, nil, engine)
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
