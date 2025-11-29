package utils

import (
	"cmp"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/filter"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"

	"gopkg.in/yaml.v2"
)

const (
	MaxInstances        = 25
	BatchSize           = 15
	MaximumConcurrency  = 60
	DefaultConcurrency  = 4
	MinTTL              = time.Minute
	MaxTTL              = time.Hour * 24
	DefaultInstanceTTL  = time.Minute * 5
	DefaultMetadataTTL  = time.Minute * 60
	ValidPrometheusName = `^[a-zA-Z_:][a-zA-Z0-9_:]*$`
)

func LoadConfig(filePath string) (*models.ParsedConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			config := createDefaultConfig()
			applyDefaults(&config)
			return parsedValidateConfig(&config)
		}
		return nil, err
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	applyDefaults(&config)

	return parsedValidateConfig(&config)
}

func createDefaultConfig() models.Config {
	return models.Config{
		Discovery: models.DiscoveryConfig{
			Regions: []string{},
			Instances: models.InstancesConfig{
				MaxInstances: 0,
				InstanceTTL:  "",
			},
			Metrics: models.MetricsConfig{
				Statistic:   "",
				MetadataTTL: "",
			},
			Processing: models.ProcessingConfig{
				Concurrency: 0,
			},
		},
		Export: models.ExportConfig{
			Port: 0,
			Prometheus: models.PrometheusConfig{
				MetricPrefix: "",
			},
		},
	}
}

func applyDefaults(config *models.Config) {
	if len(config.Discovery.Regions) == 0 {
		config.Discovery.Regions = []string{"us-west-2"}
	}

	if config.Discovery.Instances.MaxInstances <= 0 {
		config.Discovery.Instances.MaxInstances = MaxInstances
	}

	if config.Discovery.Instances.InstanceTTL == "" {
		config.Discovery.Instances.InstanceTTL = "5m"
	}

	if config.Discovery.Metrics.Statistic == "" {
		config.Discovery.Metrics.Statistic = "avg"
	}

	if config.Discovery.Metrics.MetadataTTL == "" {
		config.Discovery.Metrics.MetadataTTL = "60m"
	}

	if config.Discovery.Processing.Concurrency == 0 {
		config.Discovery.Processing.Concurrency = DefaultConcurrency
	}

	if config.Export.Port == 0 {
		config.Export.Port = 8081
	}

	if config.Export.Prometheus.MetricPrefix == "" {
		config.Export.Prometheus.MetricPrefix = "dbi"
	}
}

func parsedValidateConfig(config *models.Config) (*models.ParsedConfig, error) {
	var parsedConfig models.ParsedConfig

	if len(config.Discovery.Regions) > 1 {
		// Current version only supports single region exporter
		parsedConfig.Discovery.Regions = []string{config.Discovery.Regions[0]}
	} else {
		parsedConfig.Discovery.Regions = config.Discovery.Regions
	}

	instancesConfig, err := parseInstancesConfig(config.Discovery.Instances)
	if err != nil {
		return nil, err
	}
	parsedConfig.Discovery.Instances = instancesConfig

	metricsConfig, err := parsedMetricsConfig(config.Discovery.Metrics)
	if err != nil {
		return nil, err
	}
	parsedConfig.Discovery.Metrics = metricsConfig

	parsedConfig.Discovery.Processing = parseProcessingConfig(config.Discovery.Processing)

	exportConfig, err := parseExportConfig(config.Export)
	if err != nil {
		return nil, err
	}
	parsedConfig.Export = exportConfig

	return &parsedConfig, nil
}

func getAllValidFilterFields() map[string]bool {
	validFields := make(map[string]bool)

	instance := models.Instance{}
	for fieldName := range instance.GetFilterableFields() {
		validFields[fieldName] = true
	}

	metric := models.MetricDetails{}
	for fieldName := range metric.GetFilterableFields() {
		validFields[fieldName] = true
	}

	return validFields
}

func isValidFilterField(fieldName string) bool {
	if strings.HasPrefix(fieldName, models.FilterTypeTagPrefix.String()) {
		return len(fieldName) > len(models.FilterTypeTagPrefix)
	}

	validFields := getAllValidFilterFields()
	return validFields[fieldName]
}

func compileFilterConfig(config models.FilterConfig) (filter.Patterns, error) {
	if config == nil {
		return nil, nil
	}

	filter := filter.Patterns{}
	for fieldName, patterns := range config {
		if !isValidFilterField(fieldName) {
			return nil, fmt.Errorf("invalid filter field '%s' in config.yml", fieldName)
		}

		compiledPatterns, err := compileRegexPatterns(patterns)
		if err != nil {
			return nil, fmt.Errorf("invalid filter patterns in config.yml: %v", err)
		}

		filter[fieldName] = compiledPatterns
	}

	return filter, nil
}

func parseInstancesConfig(config models.InstancesConfig) (models.ParsedInstancesConfig, error) {
	maxInstances := GetOrDefault(config.MaxInstances, 1, MaxInstances, MaxInstances, "max-instances")

	instanceTTL, err := time.ParseDuration(config.InstanceTTL)
	if err != nil {
		return models.ParsedInstancesConfig{}, fmt.Errorf("invalid instances.ttl format '%s' in config.yml: %v", config.InstanceTTL, err)
	}

	instanceTTL = GetOrDefault(instanceTTL, MinTTL, MaxTTL, DefaultInstanceTTL, "instances.ttl")

	includePatterns, err := compileFilterConfig(config.Include)
	if err != nil {
		return models.ParsedInstancesConfig{}, fmt.Errorf("invalid instance.include patterns in config.yml: %v", err)
	}

	excludePatterns, err := compileFilterConfig(config.Exclude)
	if err != nil {
		return models.ParsedInstancesConfig{}, fmt.Errorf("invalid instance.exclude patterns in config.yml: %v", err)
	}

	var instanceFilter filter.Filter
	if len(includePatterns) > 0 || len(excludePatterns) > 0 {
		instanceFilter = filter.NewPatternFilter(includePatterns, excludePatterns)
	}

	return models.ParsedInstancesConfig{
		MaxInstances: maxInstances,
		InstanceTTL:  instanceTTL,
		Filter:       instanceFilter,
	}, nil
}

func extractMetricAndStatistic(pattern string) (string, string) {
	for _, statistic := range models.GetAllStatistics() {
		suffix := "." + statistic.String()
		if strings.HasSuffix(strings.ToLower(pattern), suffix) {
			metricName := strings.TrimSuffix(pattern, suffix)
			return metricName, statistic.String()
		}
	}
	return "", ""
}

func parsedMetricsConfig(config models.MetricsConfig) (models.ParsedMetricsConfig, error) {
	defaultStatistic := models.NewStatistic(config.Statistic)
	if defaultStatistic == "" {
		return models.ParsedMetricsConfig{}, fmt.Errorf("invalid statistic %s provided in config.yml", config.Statistic)
	}

	metadataTTL, err := time.ParseDuration(config.MetadataTTL)
	if err != nil {
		return models.ParsedMetricsConfig{}, fmt.Errorf("invalid metrics.metadata-ttl format '%s' in config.yml: %v", config.MetadataTTL, err)
	}

	metadataTTL = GetOrDefault(metadataTTL, MinTTL, MaxTTL, DefaultMetadataTTL, "metrics.metadata-ttl")

	includePatterns, err := compileFilterConfig(config.Include)
	if err != nil {
		return models.ParsedMetricsConfig{}, fmt.Errorf("invalid metrics.include patterns in config.yml: %v", err)
	}

	excludePatterns, err := compileFilterConfig(config.Exclude)
	if err != nil {
		return models.ParsedMetricsConfig{}, fmt.Errorf("invalid metrics.exclude patterns in config.yml: %v", err)
	}

	var metricFilter filter.Filter
	if len(includePatterns) > 0 || len(excludePatterns) > 0 {
		metricFilter = filter.NewPatternFilter(includePatterns, excludePatterns)
	}

	return models.ParsedMetricsConfig{
		Statistic:   defaultStatistic,
		MetadataTTL: metadataTTL,
		Filter:      metricFilter,
		Include:     config.Include,
		Exclude:     config.Exclude,
	}, nil
}

func parseProcessingConfig(config models.ProcessingConfig) models.ParsedProcessingConfig {
	concurrency := GetOrDefault(config.Concurrency, 1, DefaultConcurrency, DefaultConcurrency, "concurrency")

	return models.ParsedProcessingConfig{
		Concurrency: concurrency,
	}
}

func parseExportConfig(config models.ExportConfig) (models.ParsedExportConfig, error) {
	port := config.Port
	if port <= 0 || port > 65535 {
		port = 8081
	}

	if !isPortAvailable(port) {
		return models.ParsedExportConfig{}, fmt.Errorf("invalid export.port in config.yml, port %d is not available", port)
	}

	metricPrefix := config.Prometheus.MetricPrefix
	if err := validatePrometheusMetricPrefix(metricPrefix); err != nil {
		return models.ParsedExportConfig{}, err
	}

	return models.ParsedExportConfig{
		Port: port,
		Prometheus: models.ParsedPrometheusConfig{
			MetricPrefix: metricPrefix,
		},
	}, nil
}

func isPortAvailable(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf(":%d", port), time.Second)
	if err != nil {
		return true
	}
	conn.Close()
	return false
}

func validatePrometheusMetricPrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("invalid prometheus.metric-prefix in config.yml, prefix cannot be empty")
	}

	validName := regexp.MustCompile(ValidPrometheusName)
	if !validName.MatchString(prefix) {
		return fmt.Errorf("invalid prometheus.metric-prefix in config.yml, prefix '%s' is not valid", prefix)
	}

	if strings.HasPrefix(prefix, "_") {
		return fmt.Errorf("invalid prometheus.metric-prefix in config.yml, prefix '%s' cannot start with '_'", prefix)
	}

	return nil
}

func GetOrDefault[T cmp.Ordered](value, min, max, defaultValue T, fieldName string) T {
	if value < min || value > max {
		log.Printf("[CONFIG] %s %v is outside the allowed range [%v, %v], setting to %v", fieldName, value, min, max, defaultValue)
		return defaultValue
	}
	return value
}
