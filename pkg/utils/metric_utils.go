package utils

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/pi/types"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
)

// MetricDescriptionRegistry manages canonical descriptions for metrics to ensure consistency
// across different database engines that may return varying descriptions for the same metric
type MetricDescriptionRegistry struct {
	mu           sync.Mutex
	descriptions map[string]string
}

// PerEngineMetricRegistry manages separate metric description registries for each database engine
// This allows different engines to have their own canonical descriptions for the same metric name
type PerEngineMetricRegistry struct {
	mu         sync.Mutex
	registries map[models.Engine]*MetricDescriptionRegistry
}

func NewPerEngineMetricRegistry() *PerEngineMetricRegistry {
	return &PerEngineMetricRegistry{
		registries: make(map[models.Engine]*MetricDescriptionRegistry),
	}
}

func (r *MetricDescriptionRegistry) GetCanonicalDescription(metricName, awsDescription string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	normalizedMetricName := strings.ToLower(metricName)
	if canonical, exists := r.descriptions[normalizedMetricName]; exists {
		return canonical
	}

	r.descriptions[normalizedMetricName] = awsDescription
	return awsDescription
}

func (r *MetricDescriptionRegistry) ResetRegistry() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.descriptions = make(map[string]string)
}

func (per *PerEngineMetricRegistry) GetEngineRegistry(engine models.Engine) *MetricDescriptionRegistry {
	per.mu.Lock()
	defer per.mu.Unlock()

	if registry, exists := per.registries[engine]; exists {
		return registry
	}

	per.registries[engine] = &MetricDescriptionRegistry{
		descriptions: make(map[string]string),
	}
	return per.registries[engine]
}

func (per *PerEngineMetricRegistry) ResetAllRegistries() {
	per.mu.Lock()
	defer per.mu.Unlock()
	per.registries = make(map[models.Engine]*MetricDescriptionRegistry)
}

func (per *PerEngineMetricRegistry) ResetEngineRegistry(engine models.Engine) {
	per.mu.Lock()
	defer per.mu.Unlock()
	if registry, exists := per.registries[engine]; exists {
		registry.ResetRegistry()
	}
}

func GetMetricNamesWithStatistic(metricsDefinitionMap map[string]models.MetricDetails) []string {
	var metricNamesWithStat []string
	for _, metric := range metricsDefinitionMap {
		for _, statistic := range metric.Statistics {
			metricNamesWithStat = append(metricNamesWithStat, metric.Name+"."+statistic.String())
		}
	}
	return metricNamesWithStat
}

func BuildMetricDefinitionMap(availableMetrics []types.ResponseResourceMetric, metricConfig *models.ParsedMetricsConfig, engine models.Engine, registry *PerEngineMetricRegistry) (map[string]models.MetricDetails, error) {
	if len(availableMetrics) == 0 {
		return nil, fmt.Errorf("[METRIC UTILS] NO metrics provided to build")
	}

	metricDefinitionMap := make(map[string]models.MetricDetails, len(availableMetrics))
	engineRegistry := registry.GetEngineRegistry(engine)

	for _, metric := range availableMetrics {
		if validResponseResourceMetric(metric) {
			metricName := *metric.Metric
			statistics := getMetricStatistics(metricName, metricConfig)

			if len(statistics) > 0 {
				canonicalDescription := engineRegistry.GetCanonicalDescription(metricName, *metric.Description)

				metricDefinitionMap[metricName] = models.MetricDetails{
					Name:        metricName,
					Description: canonicalDescription,
					Unit:        *metric.Unit,
					Statistics:  statistics,
				}
			}
		}
	}

	return metricDefinitionMap, nil
}

func validResponseResourceMetric(metric types.ResponseResourceMetric) bool {
	return metric.Metric != nil && metric.Description != nil && metric.Unit != nil
}

func getMetricStatistics(metricName string, metricConfig *models.ParsedMetricsConfig) []models.Statistic {
	if metricConfig == nil {
		return []models.Statistic{models.StatisticAvg}
	}

	if shouldExcludeMetric(metricName, metricConfig) {
		return []models.Statistic{}
	}

	return determineIncludedStatistics(metricName, metricConfig)
}

func shouldExcludeMetric(metricName string, metricConfig *models.ParsedMetricsConfig) bool {
	if len(metricConfig.Exclude) == 0 {
		return false
	}

	if namePatterns, exists := metricConfig.Exclude[models.FilterTypeName.String()]; exists {
		for _, pattern := range namePatterns {
			if patternMatchesMetric(pattern, metricName) {
				return true
			}
		}
	}

	if categoryPatterns, exists := metricConfig.Exclude[models.FilterTypeCategory.String()]; exists {
		metricCategory := models.DeriveMetricCategory(metricName)
		for _, pattern := range categoryPatterns {
			if patternMatchesMetric(pattern, metricCategory) {
				return true
			}
		}
	}

	return false
}

func determineIncludedStatistics(metricName string, metricConfig *models.ParsedMetricsConfig) []models.Statistic {
	var statistics []models.Statistic
	seenStatistics := make(map[models.Statistic]bool)

	statistics = append(statistics, metricConfig.Statistic)
	seenStatistics[metricConfig.Statistic] = true

	if len(metricConfig.Include) == 0 {
		return statistics
	}

	explicitStats := extractExplicitStatisticsFromInclude(metricName, metricConfig.Include)
	for _, stat := range explicitStats {
		if !seenStatistics[stat] {
			statistics = append(statistics, stat)
			seenStatistics[stat] = true
		}
	}

	if matchesIncludePatterns(metricName, metricConfig.Include) {
		if !seenStatistics[metricConfig.Statistic] {
			statistics = append(statistics, metricConfig.Statistic)
			seenStatistics[metricConfig.Statistic] = true
		}
	}

	return statistics
}

func extractExplicitStatisticsFromInclude(metricName string, patterns models.FilterConfig) []models.Statistic {
	var statistics []models.Statistic

	if namePatterns, exists := patterns[models.FilterTypeName.String()]; exists {
		for _, pattern := range namePatterns {
			if basePattern, statisticStr := extractMetricAndStatistic(pattern); basePattern != "" && statisticStr != "" {
				if patternMatchesMetric(basePattern, metricName) {
					if stat := models.NewStatistic(statisticStr); stat.IsValid() {
						statistics = append(statistics, stat)
					}
				}
			}
		}
	}

	return statistics
}

func matchesIncludePatterns(metricName string, patterns models.FilterConfig) bool {
	if namePatterns, exists := patterns[models.FilterTypeName.String()]; exists {
		for _, pattern := range namePatterns {
			if _, statisticStr := extractMetricAndStatistic(pattern); statisticStr != "" {
				continue
			}
			if patternMatchesMetric(pattern, metricName) {
				return true
			}
		}
	}

	if categoryPatterns, exists := patterns[models.FilterTypeCategory.String()]; exists {
		metricCategory := models.DeriveMetricCategory(metricName)
		for _, pattern := range categoryPatterns {
			if patternMatchesMetric(pattern, metricCategory) {
				return true
			}
		}
	}

	return false
}

func patternMatchesMetric(pattern, metricName string) bool {
	if pattern == metricName {
		return true
	}

	if isRegexPattern(pattern) {
		if regex, err := regexp.Compile(pattern); err == nil {
			return regex.MatchString(metricName)
		}
	}

	return false
}

func TrimStatisticFromMetricName(metricNameWithStat string) string {
	for _, statistic := range models.GetAllStatistics() {
		if strings.HasSuffix(metricNameWithStat, "."+statistic.String()) {
			return strings.TrimSuffix(metricNameWithStat, "."+statistic.String())
		}
	}
	return ""
}

// EngineToShortName converts full engine names to their short versions
// aurora-postgresql -> apg
// aurora-mysql -> ams
// postgres -> postgres
// mysql -> mysql
// mariadb -> mariadb
// oracle -> oracle
// sqlserver -> sqlserver
func EngineToShortName(engine models.Engine) string {
	switch engine {
	case models.AuroraPostgreSQL:
		return "apg"
	case models.AuroraMySQL:
		return "ams"
	case models.PostgreSQL:
		return "pg"
	case models.MySQL:
		return "mysql"
	case models.MariaDB:
		return "mariadb"
	case models.Oracle:
		return "oracle"
	case models.SQLServer:
		return "sqlserver"
	default:
		return ""
	}
}
