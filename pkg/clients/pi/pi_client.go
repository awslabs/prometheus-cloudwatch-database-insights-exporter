package pi

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pi"
	"github.com/aws/aws-sdk-go-v2/service/pi/types"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/utils"
)

const PIMetricLookbackSeconds = 180 // 3 minutes for dynamic TTL calculation

type PIClient struct {
	client *pi.Client
}

// AWS Performance Insights (PI) is a database monitoring tool that provides visibility into database performance by collecting real-time performance metrics.

// PIClient wraps the AWS Performance Insights SDK client with application-specific functionality.
// It provides high-level methos for metric discovery and data collection operations.
func NewPIClient(region string) (*PIClient, error) {
	return NewPIClientWithEndpoint(region, "")
}

func NewPIClientWithEndpoint(region, endpoint string) (*PIClient, error) {
	log.Println("[PI] Creating new PI client...")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Printf("[PI] FATAL: Failed to load AWS config: %v", err)
		return nil, err
	}

	client := pi.NewFromConfig(cfg)
	if endpoint != "" {
		client = pi.NewFromConfig(cfg, func(o *pi.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
		log.Printf("[PI] Using custom endpoint: %s", endpoint)
	}

	log.Printf("[PI] AWS config loaded, region: %s", region)
	return &PIClient{
		client: client,
	}, nil
}

func (piClient *PIClient) ListAvailableResourceMetrics(ctx context.Context, resourceID string) (*pi.ListAvailableResourceMetricsOutput, error) {
	input := &pi.ListAvailableResourceMetricsInput{
		Identifier:  aws.String(resourceID),
		MetricTypes: []string{string(models.MetricTypeDB), string(models.MetricTypeOS)},
		ServiceType: types.ServiceTypeRds,
	}

	result, err := piClient.client.ListAvailableResourceMetrics(ctx, input)
	if err != nil {
		log.Printf("[LIST_AVAILABLE_RESOURCE_METRICS] Error listing available metrics for resourceID: %s, error: %v", resourceID, err)
		return nil, err
	}

	return result, nil
}

// buildMetricQueries turns metric names into API queries, attaching a GroupBy to
// the ones configured to be broken down.
func buildMetricQueries(metricNames []string, dimensionGroups map[string]*models.ParsedDimensionGroup) []types.MetricQuery {
	var metricQueries []types.MetricQuery

	for _, metricName := range metricNames {
		metricQuery := types.MetricQuery{
			Metric: aws.String(metricName),
		}

		if group := dimensionGroups[metricName]; group != nil {
			metricQuery.GroupBy = &types.DimensionGroup{
				Group:      aws.String(group.Group),
				Dimensions: group.Keys,
				Limit:      aws.Int32(group.Limit),
			}
		}

		metricQueries = append(metricQueries, metricQuery)
	}

	return metricQueries
}

// GetResourceMetrics fetches metric data for a resource. Metrics listed in
// dimensionGroups are requested with a GroupBy, so the response carries one
// series per dimension in addition to the aggregate. A nil or absent entry
// leaves the metric ungrouped.
//
// Grouping does not add API requests: GroupBy travels inside the MetricQuery it
// applies to.
func (piClient *PIClient) GetResourceMetrics(ctx context.Context, resourceID string, metricNames []string, dimensionGroups map[string]*models.ParsedDimensionGroup) (*pi.GetResourceMetricsOutput, error) {
	metricQueries := buildMetricQueries(metricNames, dimensionGroups)

	startTime := time.Now().Add(-PIMetricLookbackSeconds * time.Second)
	endTime := time.Now()

	input := &pi.GetResourceMetricsInput{
		Identifier:      aws.String(resourceID),
		MetricQueries:   metricQueries,
		ServiceType:     types.ServiceTypeRds,
		StartTime:       aws.Time(startTime),
		EndTime:         aws.Time(endTime),
		PeriodInSeconds: aws.Int32(1),
	}

	// Log request details if debug mode is enabled
	if utils.IsDebugEnabled() {
		log.Printf("[DEBUG] GetResourceMetrics Request: resource_id=%s, metrics=%v, start_time=%s, end_time=%s, period=%ds",
			resourceID, metricNames, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), 1)
	}

	result, err := piClient.client.GetResourceMetrics(ctx, input)
	if err != nil {
		return nil, err
	}

	// Log response details if debug mode is enabled
	if utils.IsDebugEnabled() {
		metricCount := 0
		dataPointCount := 0
		if result.MetricList != nil {
			metricCount = len(result.MetricList)
			log.Printf("[DEBUG] GetResourceMetrics Response: resource_id=%s, metrics_returned=%d",
				resourceID, metricCount)

			// Log each metric with its datapoints
			for _, metric := range result.MetricList {
				metricKey := ""
				if metric.Key != nil && metric.Key.Metric != nil {
					metricKey = *metric.Key.Metric
				}

				if metric.DataPoints != nil {
					dataPointCount += len(metric.DataPoints)
					log.Printf("[DEBUG]   Metric: %s, data_points=%d", metricKey, len(metric.DataPoints))

					// Log each datapoint with timestamp and value
					for i, dp := range metric.DataPoints {
						timestamp := "nil"
						value := "nil"

						if dp.Timestamp != nil {
							timestamp = dp.Timestamp.Format(time.RFC3339)
						}
						if dp.Value != nil {
							value = fmt.Sprintf("%.6f", *dp.Value)
						}

						log.Printf("[DEBUG]     DataPoint[%d]: timestamp=%s, value=%s", i, timestamp, value)
					}
				} else {
					log.Printf("[DEBUG]   Metric: %s, data_points=0", metricKey)
				}
			}

			log.Printf("[DEBUG] GetResourceMetrics Summary: resource_id=%s, total_data_points=%d",
				resourceID, dataPointCount)
		}
	}

	return result, nil
}
