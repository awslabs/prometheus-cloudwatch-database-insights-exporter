package formatting

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/utils"
)

func ConvertToPrometheusMetric(ch chan<- prometheus.Metric, instance models.Instance, metricData models.MetricData, metricPrefix string) error {

	metricName := utils.TrimStatisticFromMetricName(metricData.Metric)
	if metricName == "" {
		return fmt.Errorf("metric name is empty")
	}
	metric, err := safeGetMetricDetails(instance, metricName)
	if err != nil {
		return err
	}

	metricLabels := []string{"identifier", "engine", "unit"}

	engineShortStr := utils.EngineToShortName(instance.Engine)
	prometheusDesc := buildPrometheusDescription(
		buildPrometheusMetricName(metricPrefix, engineShortStr, metricData.Metric),
		metric.Description,
		metricLabels,
	)

	prometheusMetric, err := prometheus.NewConstMetric(
		prometheusDesc,
		prometheus.GaugeValue,
		metricData.Value,
		instance.Identifier,
		string(instance.Engine),
		metric.Unit,
	)
	if err != nil {
		return err
	}

	ch <- prometheus.NewMetricWithTimestamp(metricData.Timestamp, prometheusMetric)
	return nil
}

func safeGetMetricDetails(instance models.Instance, metricName string) (*models.MetricDetails, error) {
	if instance.Metrics == nil {
		return nil, fmt.Errorf("instance.Metrics is nil for instance %s", instance.Identifier)
	}

	if instance.Metrics.MetricsDetails == nil {
		return nil, fmt.Errorf("instance.Metrics.MetricsDetails is nil for instance %s", instance.Identifier)
	}

	metric, exists := instance.Metrics.MetricsDetails[metricName]
	if !exists {
		return nil, fmt.Errorf("metric %s not found for instance %s", metricName, instance.Identifier)
	}

	return &metric, nil
}

func buildPrometheusDescription(metricNameWithStat string, metricDescription string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(
		metricNameWithStat,
		metricDescription,
		labels,
		nil,
	)
}

func buildPrometheusMetricName(metricPrefix string, engineShortStr string, metricWithStatistic string) string {
	if strings.HasPrefix(metricWithStatistic, "db.") {
		metricPrefix = metricPrefix + "_" + engineShortStr
	}
	return metricPrefix + "_" + utils.SnakeCase(metricWithStatistic)
}
