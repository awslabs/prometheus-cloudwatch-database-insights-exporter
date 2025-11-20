package pi

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pi"
	"github.com/aws/aws-sdk-go-v2/service/pi/types"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
)

const PIMetricLookbackSeconds = 60

type PIClient struct {
	client *pi.Client
}

// AWS Performance Insights (PI) is a database monitoring tool that provides visibility into database performance by collecting real-time performance metrics.

// PIClient wraps the AWS Performance Insights SDK client with application-specific functionality.
// It provides high-level methos for metric discovery and data collection operations.
func NewPIClient(region string) (*PIClient, error) {
	log.Println("[PI] Creating new PI client...")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Printf("[PI] FATAL: Failed to load AWS config: %v", err)
		return nil, err
	}

	log.Printf("[PI] AWS config loaded, region: %s", region)
	return &PIClient{
		client: pi.NewFromConfig(cfg),
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

func (piClient *PIClient) GetResourceMetrics(ctx context.Context, resourceID string, metricNames []string) (*pi.GetResourceMetricsOutput, error) {
	var metricQueries []types.MetricQuery
	for _, metricName := range metricNames {
		metricQueries = append(metricQueries, types.MetricQuery{
			Metric: aws.String(metricName),
		})
	}

	input := &pi.GetResourceMetricsInput{
		Identifier:      aws.String(resourceID),
		MetricQueries:   metricQueries,
		ServiceType:     types.ServiceTypeRds,
		StartTime:       aws.Time(time.Now().Add(-PIMetricLookbackSeconds * time.Second)),
		EndTime:         aws.Time(time.Now()),
		PeriodInSeconds: aws.Int32(1),
	}

	result, err := piClient.client.GetResourceMetrics(ctx, input)
	if err != nil {
		return nil, err
	}

	return result, nil
}
