package models

import (
	"time"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/filter"
)

type Config struct {
	Discovery DiscoveryConfig
	Export    ExportConfig
}

type DiscoveryConfig struct {
	Regions    []string
	Instances  InstancesConfig
	Metrics    MetricsConfig
	Processing ProcessingConfig
}

type ExportConfig struct {
	Port       int
	Prometheus PrometheusConfig
}

type InstancesConfig struct {
	MaxInstances int          `yaml:"max-instances"`
	InstanceTTL  string       `yaml:"ttl"`
	Include      FilterConfig `yaml:"include,omitempty"`
	Exclude      FilterConfig `yaml:"exclude,omitempty"`
}

type MetricsConfig struct {
	Statistic   string
	MetadataTTL string       `yaml:"metadata-ttl"`
	Include     FilterConfig `yaml:"include,omitempty"`
	Exclude     FilterConfig `yaml:"exclude,omitempty"`
}

type ProcessingConfig struct {
	Concurrency int
}

type PrometheusConfig struct {
	MetricPrefix string `yaml:"metric-prefix"`
}

type FilterConfig map[string][]string

type ParsedConfig struct {
	Discovery ParsedDiscoveryConfig
	Export    ParsedExportConfig
}

type ParsedDiscoveryConfig struct {
	Regions    []string
	Instances  ParsedInstancesConfig
	Metrics    ParsedMetricsConfig
	Processing ParsedProcessingConfig
}

type ParsedExportConfig struct {
	Port       int
	Prometheus ParsedPrometheusConfig
}

type ParsedInstancesConfig struct {
	MaxInstances int `yaml:"max-instances"`
	InstanceTTL  time.Duration
	Filter       filter.Filter
}

type ParsedMetricsConfig struct {
	Statistic   Statistic
	MetadataTTL time.Duration `yaml:"metadata-ttl"`
	Filter      filter.Filter
	Include     FilterConfig
	Exclude     FilterConfig
}

type ParsedProcessingConfig struct {
	Concurrency int
}

type ParsedPrometheusConfig struct {
	MetricPrefix string `yaml:"metric-prefix"`
}

func (instanceConfig *ParsedInstancesConfig) ShouldIncludeInstance(instance filter.Filterable) bool {
	if instanceConfig.Filter == nil {
		return true
	}
	return instanceConfig.Filter.ShouldInclude(instance)
}

func (metricConfig *ParsedMetricsConfig) ShouldIncludeMetric(metricDetails filter.Filterable) bool {
	if metricConfig.Filter == nil {
		return true
	}
	return metricConfig.Filter.ShouldInclude(metricDetails)
}
