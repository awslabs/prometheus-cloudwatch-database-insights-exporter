package models

type Config struct {
	Discovery DiscoveryConfig
	Export    ExportConfig
}

type DiscoveryConfig struct {
	Regions   []string
	Instances InstancesConfig
	Metrics   MetricsConfig
}

type ExportConfig struct {
	Port int
}

type InstancesConfig struct {
	MaxInstances int `yaml:"max-instances"`
}

type MetricsConfig struct {
	Statistic string
}

type ParsedConfig struct {
	Discovery ParsedDiscoveryConfig
	Export    ParsedExportConfig
}

type ParsedDiscoveryConfig struct {
	Regions   []string
	Instances ParsedInstancesConfig
	Metrics   ParsedMetricsConfig
}

type ParsedExportConfig struct {
	Port int
}

type ParsedInstancesConfig struct {
	MaxInstances int `yaml:"max-instances"`
}

type ParsedMetricsConfig struct {
	Statistic MetricStatisticConfig
}

type MetricStatisticConfig struct {
	Default    Statistic
	Configured map[string][]Statistic // key is "metricName"
}
