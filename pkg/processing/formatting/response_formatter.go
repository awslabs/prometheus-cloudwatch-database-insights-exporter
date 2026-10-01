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
	labelValues := []string{instance.Identifier, string(instance.Engine), metric.Unit}

	// Label names come from the configured keys, not from the response, so every
	// series of a grouped metric presents the same label set.
	for _, dimensionKey := range metricData.DimensionKeys {
		metricLabels = append(metricLabels, utils.SnakeCase(dimensionKey))
		labelValues = append(labelValues, metricData.Dimensions[dimensionKey])
	}

	engineShortStr := utils.EngineToShortName(instance.Engine)
	prometheusDesc := buildPrometheusDescription(
		buildPrometheusMetricName(metricPrefix, engineShortStr, metricData.Metric, metricData.DimensionGroup),
		metric.Description,
		metricLabels,
	)

	prometheusMetric, err := prometheus.NewConstMetric(
		prometheusDesc,
		prometheus.GaugeValue,
		metricData.Value,
		labelValues...,
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

// buildPrometheusMetricName names the exported series. Series broken down by a
// dimension group get their own metric name, suffixed with the group.
//
// They deliberately do not share a name with the ungrouped metric: GroupBy.Limit
// returns the top N dimensions and adds no "other" bucket, so the breakdown can
// sum to less than the total. Keeping them apart means neither double counts nor
// silently under-reports.
func buildPrometheusMetricName(metricPrefix string, engineShortStr string, metricWithStatistic string, dimensionGroup string) string {
	if strings.HasPrefix(metricWithStatistic, "db.") {
		metricPrefix = metricPrefix + "_" + engineShortStr
	}

	name := metricPrefix + "_" + utils.SnakeCase(metricWithStatistic)
	if dimensionGroup != "" {
		name = name + "_by_" + utils.SnakeCase(dimensionGroup)
	}
	return name
}
