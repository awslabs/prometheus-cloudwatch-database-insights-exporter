package models

import (
	"testing"
	"time"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/filter"
	"github.com/stretchr/testify/assert"
)

func TestEngineIsValid(t *testing.T) {
	tests := []struct {
		name     string
		engine   Engine
		expected bool
	}{
		{
			name:     "AuroraPostgreSQL is valid",
			engine:   AuroraPostgreSQL,
			expected: true,
		},
		{
			name:     "AuroraMySQL is valid",
			engine:   AuroraMySQL,
			expected: true,
		},
		{
			name:     "PostgreSQL is valid",
			engine:   PostgreSQL,
			expected: true,
		},
		{
			name:     "MySQL is valid",
			engine:   MySQL,
			expected: true,
		},
		{
			name:     "MariaDB is valid",
			engine:   MariaDB,
			expected: true,
		},
		{
			name:     "Oracle is valid",
			engine:   Oracle,
			expected: true,
		},
		{
			name:     "SQLServer is valid",
			engine:   SQLServer,
			expected: true,
		},
		{
			name:     "Empty engine is invalid",
			engine:   "",
			expected: false,
		},
		{
			name:     "Invalid engine string",
			engine:   Engine("invalid-engine"),
			expected: false,
		},
		{
			name:     "Random string is invalid",
			engine:   Engine("random-database"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.engine.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewEngine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Engine
	}{
		{
			name:     "Exact match: aurora-postgresql",
			input:    "aurora-postgresql",
			expected: AuroraPostgreSQL,
		},
		{
			name:     "Exact match: aurora-mysql",
			input:    "aurora-mysql",
			expected: AuroraMySQL,
		},
		{
			name:     "Exact match: postgres",
			input:    "postgres",
			expected: PostgreSQL,
		},
		{
			name:     "Exact match: mysql",
			input:    "mysql",
			expected: MySQL,
		},
		{
			name:     "Exact match: mariadb",
			input:    "mariadb",
			expected: MariaDB,
		},
		{
			name:     "Partial match: oracle (lowercase)",
			input:    "oracle",
			expected: Oracle,
		},
		{
			name:     "Partial match: Oracle (mixed case)",
			input:    "Oracle",
			expected: Oracle,
		},
		{
			name:     "Partial match: ORACLE (uppercase)",
			input:    "ORACLE",
			expected: Oracle,
		},
		{
			name:     "Partial match: oracle-ee",
			input:    "oracle-ee",
			expected: Oracle,
		},
		{
			name:     "Partial match: oracle-se2",
			input:    "oracle-se2",
			expected: Oracle,
		},
		{
			name:     "Partial match: custom-oracle-db",
			input:    "custom-oracle-db",
			expected: Oracle,
		},
		{
			name:     "Partial match: sqlserver (lowercase)",
			input:    "sqlserver",
			expected: SQLServer,
		},
		{
			name:     "Partial match: SQLServer (mixed case)",
			input:    "SQLServer",
			expected: SQLServer,
		},
		{
			name:     "Partial match: SQLSERVER (uppercase)",
			input:    "SQLSERVER",
			expected: SQLServer,
		},
		{
			name:     "Partial match: sqlserver-ee",
			input:    "sqlserver-ee",
			expected: SQLServer,
		},
		{
			name:     "Partial match: sqlserver-se",
			input:    "sqlserver-se",
			expected: SQLServer,
		},
		{
			name:     "Partial match: custom-sqlserver-db",
			input:    "custom-sqlserver-db",
			expected: SQLServer,
		},
		{
			name:     "Invalid engine returns empty",
			input:    "invalid-engine",
			expected: "",
		},
		{
			name:     "Empty string returns empty",
			input:    "",
			expected: "",
		},
		{
			name:     "Partial match should not work for postgres",
			input:    "my-postgres-db",
			expected: "",
		},
		{
			name:     "Partial match should not work for mysql",
			input:    "my-mysql-db",
			expected: "",
		},
		{
			name:     "Partial match should not work for mariadb",
			input:    "my-mariadb-db",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewEngine(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatisticIsValid(t *testing.T) {
	tests := []struct {
		name      string
		statistic Statistic
		expected  bool
	}{
		{
			name:      "StatisticAvg is valid",
			statistic: StatisticAvg,
			expected:  true,
		},
		{
			name:      "StatisticMin is valid",
			statistic: StatisticMin,
			expected:  true,
		},
		{
			name:      "StatisticMax is valid",
			statistic: StatisticMax,
			expected:  true,
		},
		{
			name:      "StatisticSum is valid",
			statistic: StatisticSum,
			expected:  true,
		},
		{
			name:      "Invalid statistic returns false",
			statistic: Statistic("invalid"),
			expected:  false,
		},
		{
			name:      "Empty statistic returns false",
			statistic: Statistic(""),
			expected:  false,
		},
		{
			name:      "Random statistic returns false",
			statistic: Statistic("random"),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.statistic.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewStatistic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Statistic
	}{
		{
			name:     "Valid avg statistic",
			input:    "avg",
			expected: StatisticAvg,
		},
		{
			name:     "Valid min statistic",
			input:    "min",
			expected: StatisticMin,
		},
		{
			name:     "Valid max statistic",
			input:    "max",
			expected: StatisticMax,
		},
		{
			name:     "Valid sum statistic",
			input:    "sum",
			expected: StatisticSum,
		},
		{
			name:     "Invalid statistic returns empty",
			input:    "invalid",
			expected: "",
		},
		{
			name:     "Empty string returns empty",
			input:    "",
			expected: "",
		},
		{
			name:     "Uppercase AVG returns empty",
			input:    "AVG",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewStatistic(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatisticString(t *testing.T) {
	tests := []struct {
		name      string
		statistic Statistic
		expected  string
	}{
		{
			name:      "StatisticAvg to string",
			statistic: StatisticAvg,
			expected:  "avg",
		},
		{
			name:      "StatisticMin to string",
			statistic: StatisticMin,
			expected:  "min",
		},
		{
			name:      "StatisticMax to string",
			statistic: StatisticMax,
			expected:  "max",
		},
		{
			name:      "StatisticSum to string",
			statistic: StatisticSum,
			expected:  "sum",
		},
		{
			name:      "Empty statistic to string",
			statistic: Statistic(""),
			expected:  "",
		},
		{
			name:      "Invalid statistic to string",
			statistic: Statistic("invalid"),
			expected:  "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.statistic.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAllStatistics(t *testing.T) {
	tests := []struct {
		name     string
		expected []Statistic
	}{
		{
			name:     "Returns all four statistics",
			expected: []Statistic{StatisticAvg, StatisticMin, StatisticMax, StatisticSum},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAllStatistics()
			assert.Equal(t, tt.expected, result)
			assert.Len(t, result, 4)
		})
	}
}

func TestFilterTypeString(t *testing.T) {
	tests := []struct {
		name       string
		filterType FilterType
		expected   string
	}{
		{
			name:       "FilterTypeIdentifier to string",
			filterType: FilterTypeIdentifier,
			expected:   "identifier",
		},
		{
			name:       "FilterTypeEngine to string",
			filterType: FilterTypeEngine,
			expected:   "engine",
		},
		{
			name:       "FilterTypeName to string",
			filterType: FilterTypeName,
			expected:   "name",
		},
		{
			name:       "FilterTypeCategory to string",
			filterType: FilterTypeCategory,
			expected:   "category",
		},
		{
			name:       "FilterTypeUnit to string",
			filterType: FilterTypeUnit,
			expected:   "unit",
		},
		{
			name:       "FilterTypeTagPrefix to string",
			filterType: FilterTypeTagPrefix,
			expected:   "tag.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filterType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterTypeIsValid(t *testing.T) {
	tests := []struct {
		name       string
		filterType FilterType
		expected   bool
	}{
		{
			name:       "FilterTypeIdentifier is valid",
			filterType: FilterTypeIdentifier,
			expected:   true,
		},
		{
			name:       "FilterTypeEngine is valid",
			filterType: FilterTypeEngine,
			expected:   true,
		},
		{
			name:       "FilterTypeName is valid",
			filterType: FilterTypeName,
			expected:   true,
		},
		{
			name:       "FilterTypeCategory is valid",
			filterType: FilterTypeCategory,
			expected:   true,
		},
		{
			name:       "FilterTypeUnit is valid",
			filterType: FilterTypeUnit,
			expected:   true,
		},
		{
			name:       "FilterTypeTagPrefix is valid",
			filterType: FilterTypeTagPrefix,
			expected:   true,
		},
		{
			name:       "tag.environment is valid (tag prefix)",
			filterType: FilterType("tag.environment"),
			expected:   true,
		},
		{
			name:       "tag.team is valid (tag prefix)",
			filterType: FilterType("tag.team"),
			expected:   true,
		},
		{
			name:       "Invalid filter type returns false",
			filterType: FilterType("invalid"),
			expected:   false,
		},
		{
			name:       "Empty filter type returns false",
			filterType: FilterType(""),
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filterType.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInstanceGetFilterableFields(t *testing.T) {
	tests := []struct {
		name     string
		instance Instance
		expected map[string]string
	}{
		{
			name: "PostgreSQL instance returns correct fields",
			instance: Instance{
				ResourceID: "db-TESTPOSTGRES",
				Identifier: "test-postgres-db",
				Engine:     PostgreSQL,
			},
			expected: map[string]string{
				"identifier": "test-postgres-db",
				"engine":     "postgres",
			},
		},
		{
			name: "MySQL instance returns correct fields",
			instance: Instance{
				ResourceID: "db-TESTMYSQL",
				Identifier: "test-mysql-db",
				Engine:     MySQL,
			},
			expected: map[string]string{
				"identifier": "test-mysql-db",
				"engine":     "mysql",
			},
		},
		{
			name: "Aurora PostgreSQL instance returns correct fields",
			instance: Instance{
				ResourceID: "db-AURORAPG",
				Identifier: "aurora-postgres-cluster",
				Engine:     AuroraPostgreSQL,
			},
			expected: map[string]string{
				"identifier": "aurora-postgres-cluster",
				"engine":     "aurora-postgresql",
			},
		},
		{
			name: "Empty identifier returns empty string",
			instance: Instance{
				ResourceID: "db-EMPTY",
				Identifier: "",
				Engine:     PostgreSQL,
			},
			expected: map[string]string{
				"identifier": "",
				"engine":     "postgres",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.instance.GetFilterableFields()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInstanceGetFilterableTags(t *testing.T) {
	tests := []struct {
		name     string
		instance Instance
		expected map[string]string
	}{
		{
			name: "Instance returns empty tags map",
			instance: Instance{
				ResourceID: "db-TEST",
				Identifier: "test-db",
				Engine:     PostgreSQL,
			},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.instance.GetFilterableTags()
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result)
			assert.Empty(t, result)
		})
	}
}

func TestMetricDetailsGetFilterableFields(t *testing.T) {
	tests := []struct {
		name          string
		metricDetails MetricDetails
		expected      map[string]string
	}{
		{
			name: "OS metric returns correct fields with os category",
			metricDetails: MetricDetails{
				Name:        "os.cpuUtilization.idle",
				Description: "CPU idle percentage",
				Unit:        "Percent",
			},
			expected: map[string]string{
				"name":     "os.cpuUtilization.idle",
				"category": "os",
				"unit":     "Percent",
			},
		},
		{
			name: "DB metric returns correct fields with db category",
			metricDetails: MetricDetails{
				Name:        "db.User.max_connections",
				Description: "Maximum connections",
				Unit:        "Connections",
			},
			expected: map[string]string{
				"name":     "db.User.max_connections",
				"category": "db",
				"unit":     "Connections",
			},
		},
		{
			name: "Other metric returns correct fields with other category",
			metricDetails: MetricDetails{
				Name:        "custom.metric.name",
				Description: "Custom metric",
				Unit:        "Count",
			},
			expected: map[string]string{
				"name":     "custom.metric.name",
				"category": "other",
				"unit":     "Count",
			},
		},
		{
			name: "Metric with empty name returns other category",
			metricDetails: MetricDetails{
				Name:        "",
				Description: "Empty name metric",
				Unit:        "None",
			},
			expected: map[string]string{
				"name":     "",
				"category": "other",
				"unit":     "None",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metricDetails.GetFilterableFields()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetricDetailsGetFilterableTags(t *testing.T) {
	tests := []struct {
		name          string
		metricDetails MetricDetails
		expected      map[string]string
	}{
		{
			name: "Metric returns empty tags map",
			metricDetails: MetricDetails{
				Name:        "os.cpuUtilization.idle",
				Description: "CPU idle percentage",
				Unit:        "Percent",
			},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metricDetails.GetFilterableTags()
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result)
			assert.Empty(t, result)
		})
	}
}

func TestDeriveMetricCategory(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		expected   string
	}{
		{
			name:       "os. prefix returns os category",
			metricName: "os.cpuUtilization.idle",
			expected:   "os",
		},
		{
			name:       "os.memory metric returns os category",
			metricName: "os.memory.total",
			expected:   "os",
		},
		{
			name:       "db. prefix returns db category",
			metricName: "db.User.max_connections",
			expected:   "db",
		},
		{
			name:       "db.SQL metric returns db category",
			metricName: "db.SQL.total_calls",
			expected:   "db",
		},
		{
			name:       "custom metric returns other category",
			metricName: "custom.metric.name",
			expected:   "other",
		},
		{
			name:       "empty string returns other category",
			metricName: "",
			expected:   "other",
		},
		{
			name:       "metric without prefix returns other category",
			metricName: "random_metric",
			expected:   "other",
		},
		{
			name:       "metric containing os but not prefixed returns other",
			metricName: "chaos.os.metric",
			expected:   "other",
		},
		{
			name:       "metric containing db but not prefixed returns other",
			metricName: "feedback.metric",
			expected:   "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveMetricCategory(tt.metricName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsedInstancesConfigShouldIncludeInstance(t *testing.T) {
	tests := []struct {
		name     string
		config   ParsedInstancesConfig
		instance Instance
		expected bool
	}{
		{
			name: "no filter includes all instances",
			config: ParsedInstancesConfig{
				Filter: nil,
			},
			instance: Instance{
				Identifier: "any-instance",
				Engine:     PostgreSQL,
			},
			expected: true,
		},
		{
			name: "no filter includes instance with empty identifier",
			config: ParsedInstancesConfig{
				Filter: nil,
			},
			instance: Instance{
				Identifier: "",
				Engine:     MySQL,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ShouldIncludeInstance(tt.instance)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsedMetricsConfigShouldIncludeMetric(t *testing.T) {
	tests := []struct {
		name          string
		config        ParsedMetricsConfig
		metricDetails MetricDetails
		expected      bool
	}{
		{
			name: "no filter includes all metrics",
			config: ParsedMetricsConfig{
				Filter: nil,
			},
			metricDetails: MetricDetails{
				Name:        "os.cpuUtilization.idle",
				Description: "CPU idle percentage",
				Unit:        "Percent",
			},
			expected: true,
		},
		{
			name: "no filter includes db metrics",
			config: ParsedMetricsConfig{
				Filter: nil,
			},
			metricDetails: MetricDetails{
				Name:        "db.User.max_connections",
				Description: "Maximum connections",
				Unit:        "Connections",
			},
			expected: true,
		},
		{
			name: "no filter includes metrics with empty name",
			config: ParsedMetricsConfig{
				Filter: nil,
			},
			metricDetails: MetricDetails{
				Name:        "",
				Description: "Empty metric",
				Unit:        "None",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ShouldIncludeMetric(tt.metricDetails)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInstanceWithMetrics(t *testing.T) {
	tests := []struct {
		name     string
		instance Instance
		validate func(*testing.T, Instance)
	}{
		{
			name: "complete instance with all fields",
			instance: Instance{
				ResourceID:   "db-TESTPOSTGRES",
				Identifier:   "test-postgres-db",
				Engine:       AuroraPostgreSQL,
				CreationTime: time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC),
				Metrics: &Metrics{
					MetricsDetails: map[string]MetricDetails{
						"os.cpuUtilization.idle": {
							Name:        "os.cpuUtilization.idle",
							Description: "CPU idle percentage",
							Unit:        "Percent",
							Statistics:  []Statistic{StatisticAvg},
						},
					},
					MetricsList:        []string{"os.cpuUtilization.idle.avg"},
					MetricsLastUpdated: time.Now(),
					MetadataTTL:        5 * time.Minute,
				},
			},
			validate: func(t *testing.T, inst Instance) {
				assert.Equal(t, "db-TESTPOSTGRES", inst.ResourceID)
				assert.Equal(t, "test-postgres-db", inst.Identifier)
				assert.Equal(t, AuroraPostgreSQL, inst.Engine)
				assert.NotNil(t, inst.Metrics)
				assert.Len(t, inst.Metrics.MetricsDetails, 1)
				assert.Len(t, inst.Metrics.MetricsList, 1)
			},
		},
		{
			name: "instance with nil metrics",
			instance: Instance{
				ResourceID:   "db-EMPTY",
				Identifier:   "empty-db",
				Engine:       MySQL,
				CreationTime: time.Now(),
				Metrics:      nil,
			},
			validate: func(t *testing.T, inst Instance) {
				assert.Equal(t, "db-EMPTY", inst.ResourceID)
				assert.Nil(t, inst.Metrics)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.instance)
		})
	}
}

func TestFilterConfigTypes(t *testing.T) {
	tests := []struct {
		name   string
		config FilterConfig
		expect func(*testing.T, FilterConfig)
	}{
		{
			name: "filter config with identifier patterns",
			config: FilterConfig{
				"identifier": []string{"^prod-", "^test-"},
				"engine":     []string{"postgres", "mysql"},
			},
			expect: func(t *testing.T, cfg FilterConfig) {
				assert.Len(t, cfg, 2)
				assert.Contains(t, cfg, "identifier")
				assert.Contains(t, cfg, "engine")
				assert.Equal(t, []string{"^prod-", "^test-"}, cfg["identifier"])
			},
		},
		{
			name:   "empty filter config",
			config: FilterConfig{},
			expect: func(t *testing.T, cfg FilterConfig) {
				assert.Empty(t, cfg)
			},
		},
		{
			name:   "nil filter config",
			config: nil,
			expect: func(t *testing.T, cfg FilterConfig) {
				assert.Nil(t, cfg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect(t, tt.config)
		})
	}
}

func TestParsedInstancesConfigWithFilter(t *testing.T) {
	tests := []struct {
		name     string
		config   ParsedInstancesConfig
		instance Instance
		expected bool
	}{
		{
			name: "with nil filter includes everything",
			config: ParsedInstancesConfig{
				MaxInstances: 25,
				InstanceTTL:  5 * time.Minute,
				Filter:       nil,
			},
			instance: Instance{
				Identifier: "test-db",
				Engine:     PostgreSQL,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ShouldIncludeInstance(tt.instance)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsedMetricsConfigWithFilter(t *testing.T) {
	tests := []struct {
		name          string
		config        ParsedMetricsConfig
		metricDetails MetricDetails
		expected      bool
	}{
		{
			name: "with nil filter includes everything",
			config: ParsedMetricsConfig{
				Statistic:   StatisticAvg,
				MetadataTTL: 60 * time.Minute,
				Filter:      nil,
			},
			metricDetails: MetricDetails{
				Name:        "os.cpuUtilization.idle",
				Description: "CPU idle",
				Unit:        "Percent",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ShouldIncludeMetric(tt.metricDetails)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetricDataStructure(t *testing.T) {
	tests := []struct {
		name       string
		metricData MetricData
		validate   func(*testing.T, MetricData)
	}{
		{
			name: "valid metric data with all fields",
			metricData: MetricData{
				Metric:    "os.cpuUtilization.idle.avg",
				Timestamp: time.Date(2025, 10, 28, 10, 0, 0, 0, time.UTC),
				Value:     74.5,
			},
			validate: func(t *testing.T, md MetricData) {
				assert.Equal(t, "os.cpuUtilization.idle.avg", md.Metric)
				assert.Equal(t, 74.5, md.Value)
				assert.False(t, md.Timestamp.IsZero())
			},
		},
		{
			name: "metric data with zero value",
			metricData: MetricData{
				Metric:    "os.memory.free.avg",
				Timestamp: time.Now(),
				Value:     0.0,
			},
			validate: func(t *testing.T, md MetricData) {
				assert.Equal(t, 0.0, md.Value)
			},
		},
		{
			name: "metric data with negative value",
			metricData: MetricData{
				Metric:    "db.change_rate.avg",
				Timestamp: time.Now(),
				Value:     -10.5,
			},
			validate: func(t *testing.T, md MetricData) {
				assert.Equal(t, -10.5, md.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.metricData)
		})
	}
}

func TestMetricsStructure(t *testing.T) {
	tests := []struct {
		name     string
		metrics  Metrics
		validate func(*testing.T, Metrics)
	}{
		{
			name: "metrics with empty details map",
			metrics: Metrics{
				MetricsDetails:     map[string]MetricDetails{},
				MetricsList:        []string{},
				MetricsLastUpdated: time.Time{},
				MetadataTTL:        5 * time.Minute,
			},
			validate: func(t *testing.T, m Metrics) {
				assert.Empty(t, m.MetricsDetails)
				assert.Empty(t, m.MetricsList)
				assert.True(t, m.MetricsLastUpdated.IsZero())
			},
		},
		{
			name: "metrics with nil details map",
			metrics: Metrics{
				MetricsDetails:     nil,
				MetricsList:        nil,
				MetricsLastUpdated: time.Time{},
				MetadataTTL:        5 * time.Minute,
			},
			validate: func(t *testing.T, m Metrics) {
				assert.Nil(t, m.MetricsDetails)
				assert.Nil(t, m.MetricsList)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.metrics)
		})
	}
}

func TestFilterInterfaceImplementation(t *testing.T) {
	tests := []struct {
		name       string
		filterable filter.Filterable
		validate   func(*testing.T, filter.Filterable)
	}{
		{
			name: "Instance implements Filterable interface",
			filterable: Instance{
				ResourceID: "db-TEST",
				Identifier: "test-db",
				Engine:     PostgreSQL,
			},
			validate: func(t *testing.T, f filter.Filterable) {
				fields := f.GetFilterableFields()
				tags := f.GetFilterableTags()
				assert.NotNil(t, fields)
				assert.NotNil(t, tags)
				assert.Contains(t, fields, "identifier")
				assert.Contains(t, fields, "engine")
			},
		},
		{
			name: "MetricDetails implements Filterable interface",
			filterable: MetricDetails{
				Name:        "os.cpuUtilization.idle",
				Description: "CPU idle",
				Unit:        "Percent",
			},
			validate: func(t *testing.T, f filter.Filterable) {
				fields := f.GetFilterableFields()
				tags := f.GetFilterableTags()
				assert.NotNil(t, fields)
				assert.NotNil(t, tags)
				assert.Contains(t, fields, "name")
				assert.Contains(t, fields, "category")
				assert.Contains(t, fields, "unit")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.filterable)
		})
	}
}
