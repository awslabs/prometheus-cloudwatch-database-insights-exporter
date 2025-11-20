package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
)

func TestLoadConfig(t *testing.T) {
	testCases := []struct {
		name          string
		configContent string
		expectedError bool
		validate      func(*testing.T, *models.ParsedConfig)
	}{
		{
			name: "load valid config with all fields",
			configContent: `discovery:
  regions:
  - us-west-2
  instances:
    max-instances: 10
  metrics:
    statistic: "avg"
export:
  port: 8081`,
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, []string{"us-west-2"}, cfg.Discovery.Regions)
				assert.Equal(t, 10, cfg.Discovery.Instances.MaxInstances)
				assert.Equal(t, models.StatisticAvg, cfg.Discovery.Metrics.Statistic.Default)
				assert.Equal(t, 8081, cfg.Export.Port)
			},
		},
		{
			name: "load config with defaults applied",
			configContent: `discovery:
  regions: []
  metrics:
    statistic: ""
export:
  port: 0`,
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, []string{"us-west-2"}, cfg.Discovery.Regions)
				assert.Equal(t, testutils.TestMaxInstances, cfg.Discovery.Instances.MaxInstances)
				assert.Equal(t, models.StatisticAvg, cfg.Discovery.Metrics.Statistic.Default)
				assert.Equal(t, 8081, cfg.Export.Port)
			},
		},
		{
			name: "load config with multiple regions (only first is used)",
			configContent: `discovery:
  regions:
  - us-west-2
  - us-east-1
  - eu-west-1
  metrics:
    statistic: "avg"
export:
  port: 8081`,
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, []string{"us-west-2"}, cfg.Discovery.Regions)
			},
		},
		{
			name: "load config with different statistic",
			configContent: `discovery:
  regions:
  - us-east-1
  metrics:
    statistic: "max"
export:
  port: 9090`,
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, []string{"us-east-1"}, cfg.Discovery.Regions)
				assert.Equal(t, models.StatisticMax, cfg.Discovery.Metrics.Statistic.Default)
				assert.Equal(t, 9090, cfg.Export.Port)
			},
		},
		{
			name: "load config with invalid statistic",
			configContent: `discovery:
  regions:
  - us-west-2
  metrics:
    statistic: "invalid"
export:
  port: 8081`,
			expectedError: true,
			validate:      nil,
		},
		{
			name:          "load config with non-existent file uses defaults",
			configContent: "",
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, []string{"us-west-2"}, cfg.Discovery.Regions)
				assert.Equal(t, testutils.TestMaxInstances, cfg.Discovery.Instances.MaxInstances)
				assert.Equal(t, models.StatisticAvg, cfg.Discovery.Metrics.Statistic.Default)
				assert.Equal(t, 8081, cfg.Export.Port)
			},
		},
		{
			name: "load config with invalid YAML",
			configContent: `discovery:
  regions:
  - us-west-2
  metrics:
    statistic: "avg"
  invalid yaml here: [[[`,
			expectedError: true,
			validate:      nil,
		},
		{
			name: "load config with custom max instances",
			configContent: `discovery:
  regions:
  - us-west-2
  instances:
    max-instances: 5
  metrics:
    statistic: "avg"
export:
  port: 8081`,
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, 5, cfg.Discovery.Instances.MaxInstances)
			},
		},
		{
			name: "load config with max instances exceeding limit gets capped",
			configContent: `discovery:
  regions:
  - us-west-2
  instances:
    max-instances: 100
  metrics:
    statistic: "avg"
export:
  port: 8081`,
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, testutils.TestMaxInstances, cfg.Discovery.Instances.MaxInstances)
			},
		},
		{
			name: "load config with zero max instances applies default",
			configContent: `discovery:
  regions:
  - us-west-2
  instances:
    max-instances: 0
  metrics:
    statistic: "avg"
export:
  port: 8081`,
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, testutils.TestMaxInstances, cfg.Discovery.Instances.MaxInstances)
			},
		},
		{
			name: "load config with negative max instances applies default",
			configContent: `discovery:
  regions:
  - us-west-2
  instances:
    max-instances: -5
  metrics:
    statistic: "avg"
export:
  port: 8081`,
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, testutils.TestMaxInstances, cfg.Discovery.Instances.MaxInstances)
			},
		},
		{
			name: "load config with max instances = 1",
			configContent: `discovery:
  regions:
  - us-west-2
  instances:
    max-instances: 1
  metrics:
    statistic: "avg"
export:
  port: 8081`,
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, 1, cfg.Discovery.Instances.MaxInstances)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var filePath string
			var err error

			if tc.configContent != "" && tc.name != "load config with non-existent file" {
				tmpFile, err := os.CreateTemp("", "config-*.yml")
				assert.NoError(t, err)
				defer os.Remove(tmpFile.Name())

				_, err = tmpFile.WriteString(tc.configContent)
				assert.NoError(t, err)
				tmpFile.Close()

				filePath = tmpFile.Name()
			} else {
				filePath = "non-existent-file.yml"
			}

			config, err := LoadConfig(filePath)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				if tc.validate != nil {
					tc.validate(t, config)
				}
			}
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	config := createDefaultConfig()

	assert.Empty(t, config.Discovery.Regions)
	assert.Equal(t, "", config.Discovery.Metrics.Statistic)
	assert.Equal(t, 0, config.Export.Port)

	applyDefaults(&config)
	assert.Equal(t, []string{"us-west-2"}, config.Discovery.Regions)
	assert.Equal(t, "avg", config.Discovery.Metrics.Statistic)
	assert.Equal(t, 8081, config.Export.Port)
}

func TestApplyDefaults(t *testing.T) {
	testCases := []struct {
		name     string
		config   *models.Config
		expected *models.Config
	}{
		{
			name: "apply all defaults",
			config: &models.Config{
				Discovery: models.DiscoveryConfig{
					Regions: nil,
					Metrics: models.MetricsConfig{
						Statistic: "",
					},
				},
				Export: models.ExportConfig{
					Port: 0,
				},
			},
			expected: &models.Config{
				Discovery: models.DiscoveryConfig{
					Regions: []string{"us-west-2"},
					Metrics: models.MetricsConfig{
						Statistic: "avg",
					},
				},
				Export: models.ExportConfig{
					Port: 8081,
				},
			},
		},
		{
			name: "apply no defaults when all values set",
			config: &models.Config{
				Discovery: models.DiscoveryConfig{
					Regions: []string{"us-east-1"},
					Metrics: models.MetricsConfig{
						Statistic: "max",
					},
				},
				Export: models.ExportConfig{
					Port: 9090,
				},
			},
			expected: &models.Config{
				Discovery: models.DiscoveryConfig{
					Regions: []string{"us-east-1"},
					Metrics: models.MetricsConfig{
						Statistic: "max",
					},
				},
				Export: models.ExportConfig{
					Port: 9090,
				},
			},
		},
		{
			name: "apply partial defaults",
			config: &models.Config{
				Discovery: models.DiscoveryConfig{
					Regions: []string{"eu-west-1"},
					Metrics: models.MetricsConfig{
						Statistic: "",
					},
				},
				Export: models.ExportConfig{
					Port: 0,
				},
			},
			expected: &models.Config{
				Discovery: models.DiscoveryConfig{
					Regions: []string{"eu-west-1"},
					Metrics: models.MetricsConfig{
						Statistic: "avg",
					},
				},
				Export: models.ExportConfig{
					Port: 8081,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			applyDefaults(tc.config)

			assert.Equal(t, tc.expected.Discovery.Regions, tc.config.Discovery.Regions)
			assert.Equal(t, tc.expected.Discovery.Metrics.Statistic, tc.config.Discovery.Metrics.Statistic)
			assert.Equal(t, tc.expected.Export.Port, tc.config.Export.Port)
		})
	}
}

func TestParsedValidateConfig(t *testing.T) {
	testCases := []struct {
		name          string
		config        *models.Config
		expectedError bool
		validate      func(*testing.T, *models.ParsedConfig)
	}{
		{
			name: "valid config with single region",
			config: &models.Config{
				Discovery: models.DiscoveryConfig{
					Regions: []string{"us-west-2"},
					Metrics: models.MetricsConfig{
						Statistic: "avg",
					},
				},
				Export: models.ExportConfig{
					Port: 8081,
				},
			},
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, []string{"us-west-2"}, cfg.Discovery.Regions)
				assert.Equal(t, models.StatisticAvg, cfg.Discovery.Metrics.Statistic.Default)
				assert.Equal(t, 8081, cfg.Export.Port)
			},
		},
		{
			name: "valid config with multiple regions (only first used)",
			config: &models.Config{
				Discovery: models.DiscoveryConfig{
					Regions: []string{"us-west-2", "us-east-1", "eu-west-1"},
					Metrics: models.MetricsConfig{
						Statistic: "max",
					},
				},
				Export: models.ExportConfig{
					Port: 9090,
				},
			},
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, []string{"us-west-2"}, cfg.Discovery.Regions)
				assert.Equal(t, models.StatisticMax, cfg.Discovery.Metrics.Statistic.Default)
				assert.Equal(t, 9090, cfg.Export.Port)
			},
		},
		{
			name: "invalid statistic returns error",
			config: &models.Config{
				Discovery: models.DiscoveryConfig{
					Regions: []string{"us-west-2"},
					Metrics: models.MetricsConfig{
						Statistic: "invalid",
					},
				},
				Export: models.ExportConfig{
					Port: 8081,
				},
			},
			expectedError: true,
			validate:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsedConfig, err := parsedValidateConfig(tc.config)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Nil(t, parsedConfig)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parsedConfig)
				if tc.validate != nil {
					tc.validate(t, parsedConfig)
				}
			}
		})
	}
}

func TestBuildParsedMetricsConfig(t *testing.T) {
	testCases := []struct {
		name     string
		input    models.MetricsConfig
		expected models.ParsedMetricsConfig
	}{
		{
			name: "build with avg statistic",
			input: models.MetricsConfig{
				Statistic: "avg",
			},
			expected: models.ParsedMetricsConfig{
				Statistic: models.MetricStatisticConfig{
					Default: models.StatisticAvg,
				},
			},
		},
		{
			name: "build with max statistic",
			input: models.MetricsConfig{
				Statistic: "max",
			},
			expected: models.ParsedMetricsConfig{
				Statistic: models.MetricStatisticConfig{
					Default: models.StatisticMax,
				},
			},
		},
		{
			name: "build with min statistic",
			input: models.MetricsConfig{
				Statistic: "min",
			},
			expected: models.ParsedMetricsConfig{
				Statistic: models.MetricStatisticConfig{
					Default: models.StatisticMin,
				},
			},
		},
		{
			name: "build with sum statistic",
			input: models.MetricsConfig{
				Statistic: "sum",
			},
			expected: models.ParsedMetricsConfig{
				Statistic: models.MetricStatisticConfig{
					Default: models.StatisticSum,
				},
			},
		},
		{
			name: "build with invalid statistic returns empty",
			input: models.MetricsConfig{
				Statistic: "invalid",
			},
			expected: models.ParsedMetricsConfig{},
		},
		{
			name: "build with empty statistic returns empty",
			input: models.MetricsConfig{
				Statistic: "",
			},
			expected: models.ParsedMetricsConfig{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildParsedMetricsConfig(tc.input)

			assert.Equal(t, tc.expected.Statistic.Default, result.Statistic.Default)
		})
	}
}
