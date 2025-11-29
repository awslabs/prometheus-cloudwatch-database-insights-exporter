package utils

import (
	"os"
	"testing"
	"time"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/filter"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
	"github.com/stretchr/testify/assert"
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
				assert.Equal(t, models.StatisticAvg, cfg.Discovery.Metrics.Statistic)
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
				assert.Equal(t, models.StatisticAvg, cfg.Discovery.Metrics.Statistic)
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
  port: 8082`,
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, []string{"us-east-1"}, cfg.Discovery.Regions)
				assert.Equal(t, models.StatisticMax, cfg.Discovery.Metrics.Statistic)
				assert.Equal(t, 8082, cfg.Export.Port)
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
				assert.Equal(t, models.StatisticAvg, cfg.Discovery.Metrics.Statistic)
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
					Port: 8082,
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
					Port: 8082,
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
			name:          "valid config with single region",
			config:        testutils.CreateTestConfig(),
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, []string{"us-west-2"}, cfg.Discovery.Regions)
				assert.Equal(t, models.StatisticAvg, cfg.Discovery.Metrics.Statistic)
				assert.Equal(t, 8081, cfg.Export.Port)
			},
		},
		{
			name: "valid config with multiple regions (only first used)",
			config: testutils.CreateTestConfig(map[string]interface{}{
				"statistic": "max",
				"port":      8082,
				"regions":   []string{"us-west-2", "us-east-1", "eu-west-1"},
			}),
			expectedError: false,
			validate: func(t *testing.T, cfg *models.ParsedConfig) {
				assert.Equal(t, []string{"us-west-2"}, cfg.Discovery.Regions)
				assert.Equal(t, models.StatisticMax, cfg.Discovery.Metrics.Statistic)
				assert.Equal(t, 8082, cfg.Export.Port)
			},
		},
		{
			name: "invalid statistic returns error",
			config: testutils.CreateTestConfig(map[string]interface{}{
				"statistic": "invalid",
			}),
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

func TestParsedMetricsConfig(t *testing.T) {
	testCases := []struct {
		name     string
		input    models.MetricsConfig
		expected models.ParsedMetricsConfig
	}{
		{
			name: "build with avg statistic",
			input: models.MetricsConfig{
				Statistic:   "avg",
				Include:     nil,
				Exclude:     nil,
				MetadataTTL: "60m",
			},
			expected: models.ParsedMetricsConfig{
				Statistic: models.StatisticAvg,
			},
		},
		{
			name: "build with max statistic",
			input: models.MetricsConfig{
				Statistic:   "max",
				Include:     nil,
				Exclude:     nil,
				MetadataTTL: "60m",
			},
			expected: models.ParsedMetricsConfig{
				Statistic: models.StatisticMax,
			},
		},
		{
			name: "build with min statistic",
			input: models.MetricsConfig{
				Statistic:   "min",
				Include:     nil,
				Exclude:     nil,
				MetadataTTL: "60m",
			},
			expected: models.ParsedMetricsConfig{
				Statistic: models.StatisticMin,
			},
		},
		{
			name: "build with sum statistic",
			input: models.MetricsConfig{
				Statistic:   "sum",
				Include:     nil,
				Exclude:     nil,
				MetadataTTL: "60m",
			},
			expected: models.ParsedMetricsConfig{
				Statistic: models.StatisticSum,
			},
		},
		{
			name: "build with invalid statistic returns empty",
			input: models.MetricsConfig{
				Statistic:   "invalid",
				Include:     nil,
				Exclude:     nil,
				MetadataTTL: "60m",
			},
			expected: models.ParsedMetricsConfig{},
		},
		{
			name: "build with empty statistic returns empty",
			input: models.MetricsConfig{
				Statistic:   "",
				Include:     nil,
				Exclude:     nil,
				MetadataTTL: "60m",
			},
			expected: models.ParsedMetricsConfig{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parsedMetricsConfig(tc.input)

			if tc.expected.Statistic == "" {
				assert.Error(t, err)
				assert.Equal(t, models.Statistic(""), result.Statistic)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected.Statistic, result.Statistic)
			}
		})
	}
}

func TestCompileRegexPatterns(t *testing.T) {
	tests := []struct {
		name          string
		patterns      []string
		expectedError bool
		expectedCount int
	}{
		{
			name:          "empty patterns",
			patterns:      []string{},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:          "nil patterns",
			patterns:      nil,
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:          "single valid pattern",
			patterns:      []string{"^prod-"},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:          "multiple valid patterns",
			patterns:      []string{"^prod-", "^test-", ".*-db-.*"},
			expectedError: false,
			expectedCount: 3,
		},
		{
			name:          "invalid regex pattern",
			patterns:      []string{"^prod-", "[invalid"},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:          "complex valid patterns",
			patterns:      []string{"^(prod|staging)-.*", ".*\\\\.(com|net)$"},
			expectedError: false,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compileRegexPatterns(tt.patterns)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedCount)
			}
		})
	}
}

func TestExtractMetricAndStatistic(t *testing.T) {
	tests := []struct {
		name              string
		pattern           string
		expectedMetric    string
		expectedStatistic string
	}{
		{
			name:              "metric with avg statistic",
			pattern:           "os.cpuUtilization.idle.avg",
			expectedMetric:    "os.cpuUtilization.idle",
			expectedStatistic: "avg",
		},
		{
			name:              "metric with max statistic",
			pattern:           "db.SQL.queries.max",
			expectedMetric:    "db.SQL.queries",
			expectedStatistic: "max",
		},
		{
			name:              "metric with min statistic",
			pattern:           "os.memory.total.min",
			expectedMetric:    "os.memory.total",
			expectedStatistic: "min",
		},
		{
			name:              "metric with sum statistic",
			pattern:           "db.User.connections.sum",
			expectedMetric:    "db.User.connections",
			expectedStatistic: "sum",
		},
		{
			name:              "metric without statistic",
			pattern:           "os.cpuUtilization.idle",
			expectedMetric:    "",
			expectedStatistic: "",
		},
		{
			name:              "metric with invalid statistic",
			pattern:           "os.cpuUtilization.idle.invalid",
			expectedMetric:    "",
			expectedStatistic: "",
		},
		{
			name:              "empty pattern",
			pattern:           "",
			expectedMetric:    "",
			expectedStatistic: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, statistic := extractMetricAndStatistic(tt.pattern)
			assert.Equal(t, tt.expectedMetric, metric)
			assert.Equal(t, tt.expectedStatistic, statistic)
		})
	}
}

func TestParseInstancesConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        models.InstancesConfig
		expectedError bool
		validate      func(*testing.T, models.ParsedInstancesConfig)
	}{
		{
			name: "valid config with no include/exclude",
			config: models.InstancesConfig{
				MaxInstances: 10,
				InstanceTTL:  "5m",
				Include:      nil,
				Exclude:      nil,
			},
			expectedError: false,
			validate: func(t *testing.T, cfg models.ParsedInstancesConfig) {
				assert.Equal(t, 10, cfg.MaxInstances)
				assert.Equal(t, 5*time.Minute, cfg.InstanceTTL)
				assert.Nil(t, cfg.Filter)
			},
		},
		{
			name: "valid config with include patterns",
			config: models.InstancesConfig{
				MaxInstances: 10,
				InstanceTTL:  "5m",
				Include: models.FilterConfig{
					"identifier": []string{"^prod-", "^test-"},
				},
				Exclude: nil,
			},
			expectedError: false,
			validate: func(t *testing.T, cfg models.ParsedInstancesConfig) {
				assert.Equal(t, 10, cfg.MaxInstances)
				assert.NotNil(t, cfg.Filter)
				assert.True(t, cfg.Filter.HasFilters())
			},
		},
		{
			name: "valid config with exclude patterns",
			config: models.InstancesConfig{
				MaxInstances: 10,
				InstanceTTL:  "5m",
				Include:      nil,
				Exclude: models.FilterConfig{
					"identifier": []string{"^temp-", "^old-"},
				},
			},
			expectedError: false,
			validate: func(t *testing.T, cfg models.ParsedInstancesConfig) {
				assert.Equal(t, 10, cfg.MaxInstances)
				assert.NotNil(t, cfg.Filter)
				assert.True(t, cfg.Filter.HasFilters())
			},
		},
		{
			name: "valid config with both include and exclude",
			config: models.InstancesConfig{
				MaxInstances: 10,
				InstanceTTL:  "5m",
				Include: models.FilterConfig{
					"identifier": []string{"^prod-"},
				},
				Exclude: models.FilterConfig{
					"identifier": []string{"^temp-"},
				},
			},
			expectedError: false,
			validate: func(t *testing.T, cfg models.ParsedInstancesConfig) {
				assert.NotNil(t, cfg.Filter)
				assert.True(t, cfg.Filter.HasFilters())
			},
		},
		{
			name: "invalid include regex pattern",
			config: models.InstancesConfig{
				MaxInstances: 10,
				InstanceTTL:  "5m",
				Include: models.FilterConfig{
					"identifier": []string{"[invalid"},
				},
				Exclude: nil,
			},
			expectedError: true,
			validate:      nil,
		},
		{
			name: "invalid exclude regex pattern",
			config: models.InstancesConfig{
				MaxInstances: 10,
				InstanceTTL:  "5m",
				Include:      nil,
				Exclude: models.FilterConfig{
					"identifier": []string{"(unclosed"},
				},
			},
			expectedError: true,
			validate:      nil,
		},
		{
			name: "invalid instance TTL format",
			config: models.InstancesConfig{
				MaxInstances: 10,
				InstanceTTL:  "invalid",
				Include: models.FilterConfig{
					"identifier": []string{"^prod-"},
				},
				Exclude: nil,
			},
			expectedError: true,
			validate:      nil,
		},
		{
			name: "maxInstances exceeds limit gets capped",
			config: models.InstancesConfig{
				MaxInstances: 100,
				InstanceTTL:  "5m",
				Include:      nil,
				Exclude:      nil,
			},
			expectedError: false,
			validate: func(t *testing.T, cfg models.ParsedInstancesConfig) {
				assert.Equal(t, MaxInstances, cfg.MaxInstances)
			},
		},
		{
			name: "invalid include field name",
			config: models.InstancesConfig{
				MaxInstances: 10,
				InstanceTTL:  "5m",
				Include: models.FilterConfig{
					"invalid_field": []string{"^prod-"},
				},
				Exclude: nil,
			},
			expectedError: true,
			validate:      nil,
		},
		{
			name: "valid tag-based filtering",
			config: models.InstancesConfig{
				MaxInstances: 10,
				InstanceTTL:  "5m",
				Include: models.FilterConfig{
					"tag.Environment": []string{"^prod"},
				},
				Exclude: nil,
			},
			expectedError: false,
			validate: func(t *testing.T, cfg models.ParsedInstancesConfig) {
				assert.NotNil(t, cfg.Filter)
				assert.True(t, cfg.Filter.HasFilters())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseInstancesConfig(tt.config)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestIsValidFilterField(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{
			name:      "valid identifier field",
			fieldName: "identifier",
			expected:  true,
		},
		{
			name:      "valid engine field",
			fieldName: "engine",
			expected:  true,
		},
		{
			name:      "valid tag field",
			fieldName: "tag.Environment",
			expected:  true,
		},
		{
			name:      "valid tag field with complex name",
			fieldName: "tag.Team-Name",
			expected:  true,
		},
		{
			name:      "invalid field name",
			fieldName: "invalid_field",
			expected:  false,
		},
		{
			name:      "empty field name",
			fieldName: "",
			expected:  false,
		},
		{
			name:      "tag prefix without tag name",
			fieldName: "tag.",
			expected:  false,
		},
		{
			name:      "tag prefix with short name",
			fieldName: "tag.a",
			expected:  true,
		},
		{
			name:      "field starting with tag but not tag field",
			fieldName: "tagname",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidFilterField(tt.fieldName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompileFilterConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        models.FilterConfig
		expectedError bool
		validate      func(*testing.T, filter.Patterns)
	}{
		{
			name:          "nil config returns nil",
			config:        nil,
			expectedError: false,
			validate: func(t *testing.T, patterns filter.Patterns) {
				assert.Nil(t, patterns)
			},
		},
		{
			name:          "empty config returns empty patterns",
			config:        models.FilterConfig{},
			expectedError: false,
			validate: func(t *testing.T, patterns filter.Patterns) {
				assert.Empty(t, patterns)
			},
		},
		{
			name: "valid single field config",
			config: models.FilterConfig{
				"identifier": []string{"^prod-", "^test-"},
			},
			expectedError: false,
			validate: func(t *testing.T, patterns filter.Patterns) {
				assert.Len(t, patterns, 1)
				assert.Len(t, patterns["identifier"], 2)
			},
		},
		{
			name: "valid multiple field config",
			config: models.FilterConfig{
				"identifier": []string{"^prod-"},
				"engine":     []string{"postgres"},
			},
			expectedError: false,
			validate: func(t *testing.T, patterns filter.Patterns) {
				assert.Len(t, patterns, 2)
				assert.Len(t, patterns["identifier"], 1)
				assert.Len(t, patterns["engine"], 1)
			},
		},
		{
			name: "valid tag field config",
			config: models.FilterConfig{
				"tag.Environment": []string{"^prod"},
				"tag.Team":        []string{"backend", "frontend"},
			},
			expectedError: false,
			validate: func(t *testing.T, patterns filter.Patterns) {
				assert.Len(t, patterns, 2)
				assert.Len(t, patterns["tag.Environment"], 1)
				assert.Len(t, patterns["tag.Team"], 2)
			},
		},
		{
			name: "invalid field name",
			config: models.FilterConfig{
				"invalid_field": []string{"^prod-"},
			},
			expectedError: true,
			validate:      nil,
		},
		{
			name: "invalid regex pattern",
			config: models.FilterConfig{
				"identifier": []string{"[invalid"},
			},
			expectedError: true,
			validate:      nil,
		},
		{
			name: "mixed valid and invalid fields",
			config: models.FilterConfig{
				"identifier":    []string{"^prod-"},
				"invalid_field": []string{"^test-"},
			},
			expectedError: true,
			validate:      nil,
		},
		{
			name: "valid field with empty patterns",
			config: models.FilterConfig{
				"identifier": []string{},
			},
			expectedError: false,
			validate: func(t *testing.T, patterns filter.Patterns) {
				assert.Len(t, patterns, 1)
				assert.Empty(t, patterns["identifier"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compileFilterConfig(tt.config)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestParsedMetricsConfigWithIncludeExclude(t *testing.T) {
	tests := []struct {
		name          string
		config        models.MetricsConfig
		expectedError bool
		validate      func(*testing.T, models.ParsedMetricsConfig)
	}{
		{
			name: "valid config with no include/exclude",
			config: models.MetricsConfig{
				Statistic:   "avg",
				MetadataTTL: "60m",
				Include:     nil,
				Exclude:     nil,
			},
			expectedError: false,
			validate: func(t *testing.T, cfg models.ParsedMetricsConfig) {
				assert.Equal(t, models.StatisticAvg, cfg.Statistic)
				assert.Equal(t, 60*time.Minute, cfg.MetadataTTL)
				assert.Nil(t, cfg.Filter)
			},
		},
		{
			name: "valid config with exclude patterns",
			config: models.MetricsConfig{
				Statistic:   "avg",
				MetadataTTL: "60m",
				Include:     nil,
				Exclude: models.FilterConfig{
					"name": []string{"^db\\."},
				},
			},
			expectedError: false,
			validate: func(t *testing.T, cfg models.ParsedMetricsConfig) {
				assert.NotNil(t, cfg.Filter)
				assert.True(t, cfg.Filter.HasFilters())
			},
		},
		{
			name: "valid config with custom statistics in include",
			config: models.MetricsConfig{
				Statistic:   "avg",
				MetadataTTL: "60m",
				Include: models.FilterConfig{
					"name": []string{"os.cpuUtilization.idle.max", "os.memory.total.min"},
				},
				Exclude: nil,
			},
			expectedError: false,
			validate: func(t *testing.T, cfg models.ParsedMetricsConfig) {
				assert.Equal(t, models.StatisticAvg, cfg.Statistic)
				assert.NotNil(t, cfg.Include)
				assert.Equal(t, 2, len(cfg.Include["name"]))
				// Filter is created for include patterns containing statistics
				assert.NotNil(t, cfg.Filter)
			},
		},
		{
			name: "exclude with statistic transforms to metric name",
			config: models.MetricsConfig{
				Statistic:   "avg",
				MetadataTTL: "60m",
				Include:     nil,
				Exclude: models.FilterConfig{
					"name": []string{"os.cpuUtilization.idle.max"},
				},
			},
			expectedError: false,
			validate: func(t *testing.T, cfg models.ParsedMetricsConfig) {
				assert.Equal(t, models.StatisticAvg, cfg.Statistic)
				assert.NotNil(t, cfg.Filter)
				assert.True(t, cfg.Filter.HasFilters())
			},
		},
		{
			name: "invalid exclude regex pattern",
			config: models.MetricsConfig{
				Statistic:   "avg",
				MetadataTTL: "60m",
				Include:     nil,
				Exclude: models.FilterConfig{
					"name": []string{"(unclosed"},
				},
			},
			expectedError: true,
			validate:      nil,
		},
		{
			name: "invalid metadata TTL format",
			config: models.MetricsConfig{
				Statistic:   "avg",
				MetadataTTL: "invalid",
				Include:     nil,
				Exclude:     nil,
			},
			expectedError: true,
			validate:      nil,
		},
		{
			name: "invalid statistic",
			config: models.MetricsConfig{
				Statistic:   "invalid",
				MetadataTTL: "60m",
				Include:     nil,
				Exclude:     nil,
			},
			expectedError: true,
			validate:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsedMetricsConfig(tt.config)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestGetOrDefaultInt(t *testing.T) {
	tests := []struct {
		name         string
		value        int
		min          int
		max          int
		defaultValue int
		expected     int
	}{
		{
			name:         "value within range",
			value:        5,
			min:          1,
			max:          10,
			defaultValue: 3,
			expected:     5,
		},
		{
			name:         "value below minimum",
			value:        0,
			min:          1,
			max:          10,
			defaultValue: 3,
			expected:     3,
		},
		{
			name:         "value above maximum",
			value:        15,
			min:          1,
			max:          10,
			defaultValue: 3,
			expected:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOrDefault(tt.value, tt.min, tt.max, tt.defaultValue, "test-field")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetOrDefaultDuration(t *testing.T) {
	tests := []struct {
		name         string
		value        time.Duration
		min          time.Duration
		max          time.Duration
		defaultValue time.Duration
		expected     time.Duration
	}{
		{
			name:         "duration within range",
			value:        5 * time.Minute,
			min:          time.Minute,
			max:          10 * time.Minute,
			defaultValue: 3 * time.Minute,
			expected:     5 * time.Minute,
		},
		{
			name:         "duration below minimum",
			value:        30 * time.Second,
			min:          time.Minute,
			max:          10 * time.Minute,
			defaultValue: 3 * time.Minute,
			expected:     3 * time.Minute,
		},
		{
			name:         "duration above maximum",
			value:        15 * time.Minute,
			min:          time.Minute,
			max:          10 * time.Minute,
			defaultValue: 3 * time.Minute,
			expected:     3 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOrDefault(tt.value, tt.min, tt.max, tt.defaultValue, "test-field")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetOrDefaultString(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		min          string
		max          string
		defaultValue string
		expected     string
	}{
		{
			name:         "string within range",
			value:        "hello",
			min:          "a",
			max:          "z",
			defaultValue: "default",
			expected:     "hello",
		},
		{
			name:         "string below minimum",
			value:        "a",
			min:          "b",
			max:          "z",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOrDefault(tt.value, tt.min, tt.max, tt.defaultValue, "test-field")
			assert.Equal(t, tt.expected, result)
		})
	}
}
