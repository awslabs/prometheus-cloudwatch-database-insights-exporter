package utils

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"gopkg.in/yaml.v2"
)

const (
	MaxInstances = 25
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
			},
			Metrics: models.MetricsConfig{
				Statistic: "",
			},
		},
		Export: models.ExportConfig{
			Port: 0,
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

	if config.Discovery.Metrics.Statistic == "" {
		config.Discovery.Metrics.Statistic = "avg"
	}

	if config.Export.Port == 0 {
		config.Export.Port = 8081
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

	if config.Discovery.Instances.MaxInstances > MaxInstances {
		parsedConfig.Discovery.Instances = models.ParsedInstancesConfig{
			MaxInstances: MaxInstances,
		}
	} else {
		parsedConfig.Discovery.Instances = models.ParsedInstancesConfig{
			MaxInstances: config.Discovery.Instances.MaxInstances,
		}
	}

	parsedConfig.Discovery.Metrics = buildParsedMetricsConfig(config.Discovery.Metrics)
	if parsedConfig.Discovery.Metrics.Statistic.Default == "" {
		return nil, errors.New("invalid statistic inputted in config.yml")
	}

	parsedConfig.Export = models.ParsedExportConfig{
		Port: config.Export.Port,
	}

	return &parsedConfig, nil
}

func buildParsedMetricsConfig(metricsConfig models.MetricsConfig) models.ParsedMetricsConfig {
	defaultStatistic := models.NewStatistic(metricsConfig.Statistic)
	if defaultStatistic != "" {
		return models.ParsedMetricsConfig{
			Statistic: models.MetricStatisticConfig{
				Default: defaultStatistic,
			},
		}
	}
	return models.ParsedMetricsConfig{}
}
