package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pi/types"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils/mocks"
)

func TestGetMetricNamesWithStatistic(t *testing.T) {
	testCases := []struct {
		name     string
		metrics  map[string]models.MetricDetails
		expected []string
	}{
		{
			name:     "metric details map",
			metrics:  testutils.TestMetricsDetails,
			expected: testutils.TestMetricNamesWithStats,
		},
		{
			name:     "empty metric map",
			metrics:  map[string]models.MetricDetails{},
			expected: []string{},
		},
		{
			name:     "nil object",
			metrics:  nil,
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetMetricNamesWithStatistic(tc.metrics)
			assert.Equal(t, len(tc.expected), len(result))

			if len(tc.expected) > 0 {
				for _, expectedName := range tc.expected {
					assert.Contains(t, result, expectedName)
				}
			}
		})
	}
}

func TestGetCanonicalDescription(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(*MetricDescriptionRegistry)
		metricName  string
		description string
		expected    string
	}{
		{
			name:        "first description becomes canonical",
			setup:       func(r *MetricDescriptionRegistry) {},
			metricName:  "db.Transactions.active_transactions",
			description: "Number of active transactions",
			expected:    "Number of active transactions",
		},
		{
			name: "subsequent different description returns original canonical",
			setup: func(r *MetricDescriptionRegistry) {
				r.GetCanonicalDescription("db.Transactions.active_transactions", "Number of active transactions")
			},
			metricName:  "db.Transactions.active_transactions",
			description: "Number of Active transactions",
			expected:    "Number of active transactions",
		},
		{
			name: "completely different description still returns original",
			setup: func(r *MetricDescriptionRegistry) {
				r.GetCanonicalDescription("db.Transactions.active_transactions", "Number of active transactions")
			},
			metricName:  "db.Transactions.active_transactions",
			description: "Completely different description",
			expected:    "Number of active transactions",
		},
		{
			name:        "different metrics have independent descriptions",
			setup:       func(r *MetricDescriptionRegistry) {},
			metricName:  "metric1",
			description: "Description for metric 1",
			expected:    "Description for metric 1",
		},
		{
			name:        "case normalization - uppercase metric name stored first",
			setup:       func(r *MetricDescriptionRegistry) {},
			metricName:  "DB.SQL.TRANSACTIONS",
			description: "Number of database transactions",
			expected:    "Number of database transactions",
		},
		{
			name: "case normalization - lowercase variant returns same canonical",
			setup: func(r *MetricDescriptionRegistry) {
				r.GetCanonicalDescription("DB.SQL.TRANSACTIONS", "Number of database transactions")
			},
			metricName:  "db.sql.transactions",
			description: "Number of Database Transactions",
			expected:    "Number of database transactions",
		},
		{
			name: "case normalization - mixed case variant returns same canonical",
			setup: func(r *MetricDescriptionRegistry) {
				r.GetCanonicalDescription("DB.SQL.TRANSACTIONS", "Number of database transactions")
			},
			metricName:  "db.SQL.transactions",
			description: "Different description",
			expected:    "Number of database transactions",
		},
		{
			name: "case normalization - lowercase stored first then uppercase lookup",
			setup: func(r *MetricDescriptionRegistry) {
				r.GetCanonicalDescription("db.sql.transactions", "Number of database transactions")
			},
			metricName:  "DB.SQL.TRANSACTIONS",
			description: "Different description",
			expected:    "Number of database transactions",
		},
		{
			name: "case normalization - mixed case stored first then any case lookup",
			setup: func(r *MetricDescriptionRegistry) {
				r.GetCanonicalDescription("Db.Sql.Transactions", "Number of database transactions")
			},
			metricName:  "DB.SQL.TRANSACTIONS",
			description: "Different description",
			expected:    "Number of database transactions",
		},
		{
			name: "case normalization - multiple case variations all return first description",
			setup: func(r *MetricDescriptionRegistry) {
				r.GetCanonicalDescription("db.Transactions.active", "Active transaction count")
				r.GetCanonicalDescription("DB.TRANSACTIONS.ACTIVE", "Different description 1")
				r.GetCanonicalDescription("db.transactions.active", "Different description 2")
			},
			metricName:  "Db.TrAnSaCtIoNs.AcTiVe",
			description: "Yet another description",
			expected:    "Active transaction count",
		},
		{
			name: "case normalization - different metrics remain independent despite case variations",
			setup: func(r *MetricDescriptionRegistry) {
				r.GetCanonicalDescription("db.SQL.queries", "SQL query count")
				r.GetCanonicalDescription("db.SQL.transactions", "Transaction count")
			},
			metricName:  "DB.SQL.QUERIES",
			description: "Different description",
			expected:    "SQL query count",
		},
		{
			name: "case normalization - real world AWS metric name variations",
			setup: func(r *MetricDescriptionRegistry) {
				r.GetCanonicalDescription("db.User.max_connections", "The maximum number of client connections")
			},
			metricName:  "DB.USER.MAX_CONNECTIONS",
			description: "The Maximum Number Of Client Connections",
			expected:    "The maximum number of client connections",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := &MetricDescriptionRegistry{
				descriptions: make(map[string]string),
			}
			tc.setup(registry)

			result := registry.GetCanonicalDescription(tc.metricName, tc.description)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestResetRegistry(t *testing.T) {
	testCases := []struct {
		name               string
		setupDescriptions  map[string]string
		afterResetMetric   string
		afterResetDesc     string
		expectedAfterReset string
	}{
		{
			name: "reset clears all descriptions",
			setupDescriptions: map[string]string{
				"metric1": "Description 1",
				"metric2": "Description 2",
			},
			afterResetMetric:   "metric1",
			afterResetDesc:     "New Description 1",
			expectedAfterReset: "New Description 1",
		},
		{
			name: "reset allows new descriptions for previously stored metrics",
			setupDescriptions: map[string]string{
				"metric1": "Old Description",
			},
			afterResetMetric:   "metric1",
			afterResetDesc:     "Completely New Description",
			expectedAfterReset: "Completely New Description",
		},
		{
			name:               "reset on empty registry",
			setupDescriptions:  map[string]string{},
			afterResetMetric:   "metric1",
			afterResetDesc:     "First Description",
			expectedAfterReset: "First Description",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := &MetricDescriptionRegistry{
				descriptions: make(map[string]string),
			}

			for metric, desc := range tc.setupDescriptions {
				registry.GetCanonicalDescription(metric, desc)
			}

			registry.ResetRegistry()

			result := registry.GetCanonicalDescription(tc.afterResetMetric, tc.afterResetDesc)
			assert.Equal(t, tc.expectedAfterReset, result)
		})
	}
}

func TestGetEngineRegistry(t *testing.T) {
	testCases := []struct {
		name            string
		engine          models.Engine
		setupRegistries map[models.Engine]*MetricDescriptionRegistry
		validateResult  func(*testing.T, *MetricDescriptionRegistry, map[models.Engine]*MetricDescriptionRegistry)
	}{
		{
			name:            "get registry for new engine creates new registry",
			engine:          models.AuroraPostgreSQL,
			setupRegistries: map[models.Engine]*MetricDescriptionRegistry{},
			validateResult: func(t *testing.T, result *MetricDescriptionRegistry, registries map[models.Engine]*MetricDescriptionRegistry) {
				assert.NotNil(t, result)
				assert.NotNil(t, result.descriptions)
				assert.Contains(t, registries, models.AuroraPostgreSQL)
			},
		},
		{
			name:   "get registry for existing engine returns same registry",
			engine: models.AuroraMySQL,
			setupRegistries: map[models.Engine]*MetricDescriptionRegistry{
				models.AuroraMySQL: {
					descriptions: map[string]string{"metric1": "desc1"},
				},
			},
			validateResult: func(t *testing.T, result *MetricDescriptionRegistry, registries map[models.Engine]*MetricDescriptionRegistry) {
				assert.NotNil(t, result)
				assert.Equal(t, "desc1", result.descriptions["metric1"])
			},
		},
		{
			name:   "different engines get different registries",
			engine: models.PostgreSQL,
			setupRegistries: map[models.Engine]*MetricDescriptionRegistry{
				models.AuroraPostgreSQL: {
					descriptions: map[string]string{"metric1": "postgres desc"},
				},
			},
			validateResult: func(t *testing.T, result *MetricDescriptionRegistry, registries map[models.Engine]*MetricDescriptionRegistry) {
				assert.NotNil(t, result)
				assert.Empty(t, result.descriptions)
				assert.Len(t, registries, 2)
			},
		},
		{
			name:   "supports all engine types",
			engine: models.Oracle,
			setupRegistries: map[models.Engine]*MetricDescriptionRegistry{
				models.AuroraPostgreSQL: {descriptions: make(map[string]string)},
				models.AuroraMySQL:      {descriptions: make(map[string]string)},
				models.PostgreSQL:       {descriptions: make(map[string]string)},
				models.MySQL:            {descriptions: make(map[string]string)},
				models.MariaDB:          {descriptions: make(map[string]string)},
				models.SQLServer:        {descriptions: make(map[string]string)},
			},
			validateResult: func(t *testing.T, result *MetricDescriptionRegistry, registries map[models.Engine]*MetricDescriptionRegistry) {
				assert.NotNil(t, result)
				assert.Len(t, registries, 7)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			per := &PerEngineMetricRegistry{
				registries: tc.setupRegistries,
			}

			result := per.GetEngineRegistry(tc.engine)
			tc.validateResult(t, result, per.registries)
		})
	}
}

func TestResetAllRegistries(t *testing.T) {
	testCases := []struct {
		name            string
		setupRegistries map[models.Engine]*MetricDescriptionRegistry
		validateResult  func(*testing.T, *PerEngineMetricRegistry)
	}{
		{
			name: "reset clears all engine registries",
			setupRegistries: map[models.Engine]*MetricDescriptionRegistry{
				models.AuroraPostgreSQL: {
					descriptions: map[string]string{"metric1": "desc1"},
				},
				models.AuroraMySQL: {
					descriptions: map[string]string{"metric2": "desc2"},
				},
			},
			validateResult: func(t *testing.T, per *PerEngineMetricRegistry) {
				assert.Empty(t, per.registries)
			},
		},
		{
			name:            "reset on empty registry",
			setupRegistries: map[models.Engine]*MetricDescriptionRegistry{},
			validateResult: func(t *testing.T, per *PerEngineMetricRegistry) {
				assert.Empty(t, per.registries)
			},
		},
		{
			name: "after reset can create new registries",
			setupRegistries: map[models.Engine]*MetricDescriptionRegistry{
				models.PostgreSQL: {
					descriptions: map[string]string{"old": "old desc"},
				},
			},
			validateResult: func(t *testing.T, per *PerEngineMetricRegistry) {
				assert.Empty(t, per.registries)

				// Create new registry after reset
				newRegistry := per.GetEngineRegistry(models.PostgreSQL)
				assert.NotNil(t, newRegistry)
				assert.Empty(t, newRegistry.descriptions)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			per := &PerEngineMetricRegistry{
				registries: tc.setupRegistries,
			}

			per.ResetAllRegistries()
			tc.validateResult(t, per)
		})
	}
}

func TestResetEngineRegistry(t *testing.T) {
	testCases := []struct {
		name            string
		setupRegistries map[models.Engine]*MetricDescriptionRegistry
		resetEngine     models.Engine
		validateResult  func(*testing.T, *PerEngineMetricRegistry)
	}{
		{
			name: "reset specific engine registry clears only that engine",
			setupRegistries: map[models.Engine]*MetricDescriptionRegistry{
				models.AuroraPostgreSQL: {
					descriptions: map[string]string{"metric1": "postgres desc"},
				},
				models.AuroraMySQL: {
					descriptions: map[string]string{"metric2": "mysql desc"},
				},
			},
			resetEngine: models.AuroraPostgreSQL,
			validateResult: func(t *testing.T, per *PerEngineMetricRegistry) {
				assert.Len(t, per.registries, 2)
				assert.Empty(t, per.registries[models.AuroraPostgreSQL].descriptions)
				assert.NotEmpty(t, per.registries[models.AuroraMySQL].descriptions)
			},
		},
		{
			name: "reset non-existent engine does nothing",
			setupRegistries: map[models.Engine]*MetricDescriptionRegistry{
				models.PostgreSQL: {
					descriptions: map[string]string{"metric1": "desc1"},
				},
			},
			resetEngine: models.MySQL,
			validateResult: func(t *testing.T, per *PerEngineMetricRegistry) {
				assert.Len(t, per.registries, 1)
				assert.NotEmpty(t, per.registries[models.PostgreSQL].descriptions)
			},
		},
		{
			name:            "reset on empty registry does nothing",
			setupRegistries: map[models.Engine]*MetricDescriptionRegistry{},
			resetEngine:     models.Oracle,
			validateResult: func(t *testing.T, per *PerEngineMetricRegistry) {
				assert.Empty(t, per.registries)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			per := &PerEngineMetricRegistry{
				registries: tc.setupRegistries,
			}

			per.ResetEngineRegistry(tc.resetEngine)
			tc.validateResult(t, per)
		})
	}
}

func TestBuildMetricDefinitionMap(t *testing.T) {
	testCases := []struct {
		name                string
		resetGlobalRegistry bool
		engine              models.Engine
		availableMetrics    []types.ResponseResourceMetric
		metricConfig        *models.ParsedMetricsConfig
		expectedError       bool
		expectedCount       int
		validateResults     func(*testing.T, map[string]models.MetricDetails)
	}{
		{
			name:                "basic - valid metrics with nil config",
			resetGlobalRegistry: true,
			engine:              models.PostgreSQL,
			availableMetrics:    mocks.NewMockPIListMetricsResponse().Metrics,
			metricConfig:        nil,
			expectedError:       false,
			expectedCount:       5,
			validateResults: func(t *testing.T, result map[string]models.MetricDetails) {
				for metricName, metricDetails := range result {
					assert.Equal(t, metricName, metricDetails.Name)
					assert.NotEmpty(t, metricDetails.Description)
					assert.NotEmpty(t, metricDetails.Unit)
					assert.NotEmpty(t, metricDetails.Statistics)
				}
			},
		},
		{
			name:                "basic - empty metrics returns error",
			resetGlobalRegistry: true,
			engine:              models.MySQL,
			availableMetrics:    []types.ResponseResourceMetric{},
			metricConfig:        nil,
			expectedError:       true,
			expectedCount:       0,
			validateResults:     nil,
		},
		{
			name:                "basic - valid metrics with max statistic config",
			resetGlobalRegistry: true,
			engine:              models.AuroraPostgreSQL,
			availableMetrics:    mocks.NewMockPIListMetricsResponse().Metrics,
			metricConfig: &models.ParsedMetricsConfig{
				Statistic: models.StatisticMax,
			},
			expectedError: false,
			expectedCount: 5,
			validateResults: func(t *testing.T, result map[string]models.MetricDetails) {
				for _, metricDetails := range result {
					assert.Contains(t, metricDetails.Statistics, models.StatisticMax)
				}
			},
		},
		{
			name:                "basic - default avg statistic when no specific patterns match",
			resetGlobalRegistry: true,
			engine:              models.AuroraMySQL,
			availableMetrics:    mocks.NewMockPIListMetricsResponse().Metrics,
			metricConfig: &models.ParsedMetricsConfig{
				Statistic: models.StatisticAvg,
			},
			expectedError: false,
			expectedCount: 5,
			validateResults: func(t *testing.T, result map[string]models.MetricDetails) {
				for _, metricDetails := range result {
					assert.Contains(t, metricDetails.Statistics, models.StatisticAvg)
				}
			},
		},
		{
			name:                "validation - filters out metrics with nil Metric field",
			resetGlobalRegistry: true,
			engine:              models.MariaDB,
			availableMetrics: []types.ResponseResourceMetric{
				{
					Metric:      aws.String("os.general.numVCPUs"),
					Description: aws.String("The number of virtual CPUs"),
					Unit:        aws.String("vCPUs"),
				},
				{
					Metric:      nil,
					Description: aws.String("Invalid metric"),
					Unit:        aws.String("units"),
				},
			},
			metricConfig:  nil,
			expectedError: false,
			expectedCount: 1,
			validateResults: func(t *testing.T, result map[string]models.MetricDetails) {
				assert.Contains(t, result, "os.general.numVCPUs")
			},
		},
		{
			name:                "validation - filters out metrics with nil Description field",
			resetGlobalRegistry: true,
			engine:              models.Oracle,
			availableMetrics: []types.ResponseResourceMetric{
				{
					Metric:      aws.String("os.general.numVCPUs"),
					Description: aws.String("The number of virtual CPUs"),
					Unit:        aws.String("vCPUs"),
				},
				{
					Metric:      aws.String("os.cpuUtilization.guest"),
					Description: nil,
					Unit:        aws.String("Percent"),
				},
			},
			metricConfig:  nil,
			expectedError: false,
			expectedCount: 1,
			validateResults: func(t *testing.T, result map[string]models.MetricDetails) {
				assert.Contains(t, result, "os.general.numVCPUs")
				assert.NotContains(t, result, "os.cpuUtilization.guest")
			},
		},
		{
			name:                "validation - filters out metrics with nil Unit field",
			resetGlobalRegistry: true,
			engine:              models.SQLServer,
			availableMetrics: []types.ResponseResourceMetric{
				{
					Metric:      aws.String("os.general.numVCPUs"),
					Description: aws.String("The number of virtual CPUs"),
					Unit:        aws.String("vCPUs"),
				},
				{
					Metric:      aws.String("os.cpuUtilization.idle"),
					Description: aws.String("The percentage of CPU that is idle"),
					Unit:        nil,
				},
			},
			metricConfig:  nil,
			expectedError: false,
			expectedCount: 1,
			validateResults: func(t *testing.T, result map[string]models.MetricDetails) {
				assert.Contains(t, result, "os.general.numVCPUs")
				assert.NotContains(t, result, "os.cpuUtilization.idle")
			},
		},
		{
			name:                "per-engine - postgres uses first description",
			resetGlobalRegistry: true,
			engine:              models.AuroraPostgreSQL,
			availableMetrics: []types.ResponseResourceMetric{
				{
					Metric:      aws.String("db.Transactions.active_transactions"),
					Description: aws.String("Number of active transactions"),
					Unit:        aws.String("Transactions"),
				},
			},
			metricConfig:  nil,
			expectedError: false,
			expectedCount: 1,
			validateResults: func(t *testing.T, result map[string]models.MetricDetails) {
				assert.Equal(t, "Number of active transactions", result["db.Transactions.active_transactions"].Description)
			},
		},
		{
			name:                "per-engine - same metric postgres second call uses canonical",
			resetGlobalRegistry: false,
			engine:              models.AuroraPostgreSQL,
			availableMetrics: []types.ResponseResourceMetric{
				{
					Metric:      aws.String("db.Transactions.active_transactions"),
					Description: aws.String("Number of Active transactions"),
					Unit:        aws.String("Transactions"),
				},
			},
			metricConfig:  nil,
			expectedError: false,
			expectedCount: 1,
			validateResults: func(t *testing.T, result map[string]models.MetricDetails) {
				registry := NewPerEngineMetricRegistry()
				firstMetrics := []types.ResponseResourceMetric{
					{
						Metric:      aws.String("db.Transactions.active_transactions"),
						Description: aws.String("Number of active transactions"),
						Unit:        aws.String("Transactions"),
					},
				}
				firstResult, err := BuildMetricDefinitionMap(firstMetrics, nil, models.AuroraPostgreSQL, registry)
				assert.NoError(t, err)
				assert.Equal(t, "Number of active transactions", firstResult["db.Transactions.active_transactions"].Description)

				secondMetrics := []types.ResponseResourceMetric{
					{
						Metric:      aws.String("db.Transactions.active_transactions"),
						Description: aws.String("Number of Active transactions"),
						Unit:        aws.String("Transactions"),
					},
				}
				secondResult, err := BuildMetricDefinitionMap(secondMetrics, nil, models.AuroraPostgreSQL, registry)
				assert.NoError(t, err)
				assert.Equal(t, "Number of active transactions", secondResult["db.Transactions.active_transactions"].Description)
			},
		},
		{
			name:                "per-engine - mysql can have different description for same metric",
			resetGlobalRegistry: false,
			engine:              models.AuroraMySQL,
			availableMetrics: []types.ResponseResourceMetric{
				{
					Metric:      aws.String("db.Transactions.active_transactions"),
					Description: aws.String("Active transaction count for MySQL"),
					Unit:        aws.String("Transactions"),
				},
			},
			metricConfig:  nil,
			expectedError: false,
			expectedCount: 1,
			validateResults: func(t *testing.T, result map[string]models.MetricDetails) {
				assert.Equal(t, "Active transaction count for MySQL", result["db.Transactions.active_transactions"].Description)
			},
		},
		{
			name:                "per-engine - each engine maintains independent descriptions",
			resetGlobalRegistry: true,
			engine:              models.PostgreSQL,
			availableMetrics: []types.ResponseResourceMetric{
				{
					Metric:      aws.String("os.cpuUtilization.total"),
					Description: aws.String("PostgreSQL CPU description"),
					Unit:        aws.String("Percent"),
				},
			},
			metricConfig:  nil,
			expectedError: false,
			expectedCount: 1,
			validateResults: func(t *testing.T, result map[string]models.MetricDetails) {
				assert.Equal(t, "PostgreSQL CPU description", result["os.cpuUtilization.total"].Description)

				mysqlMetrics := []types.ResponseResourceMetric{
					{
						Metric:      aws.String("os.cpuUtilization.total"),
						Description: aws.String("MySQL CPU description"),
						Unit:        aws.String("Percent"),
					},
				}
				mysqlRegistry := NewPerEngineMetricRegistry()
				mysqlResult, err := BuildMetricDefinitionMap(mysqlMetrics, nil, models.MySQL, mysqlRegistry)
				assert.NoError(t, err)
				assert.Equal(t, "MySQL CPU description", mysqlResult["os.cpuUtilization.total"].Description)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewPerEngineMetricRegistry()
			result, err := BuildMetricDefinitionMap(tc.availableMetrics, tc.metricConfig, tc.engine, registry)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tc.expectedCount)

				if tc.validateResults != nil {
					tc.validateResults(t, result)
				}
			}
		})
	}
}

func TestValidResponseResourceMetric(t *testing.T) {
	testCases := []struct {
		name     string
		metric   types.ResponseResourceMetric
		expected bool
	}{
		{
			name: "valid - all fields present",
			metric: types.ResponseResourceMetric{
				Metric:      aws.String("os.general.numVCPUs"),
				Description: aws.String("The number of virtual CPUs"),
				Unit:        aws.String("vCPUs"),
			},
			expected: true,
		},
		{
			name: "invalid - nil Metric field",
			metric: types.ResponseResourceMetric{
				Metric:      nil,
				Description: aws.String("Description"),
				Unit:        aws.String("units"),
			},
			expected: false,
		},
		{
			name: "invalid - nil Description field",
			metric: types.ResponseResourceMetric{
				Metric:      aws.String("os.general.numVCPUs"),
				Description: nil,
				Unit:        aws.String("vCPUs"),
			},
			expected: false,
		},
		{
			name: "invalid - nil Unit field",
			metric: types.ResponseResourceMetric{
				Metric:      aws.String("os.general.numVCPUs"),
				Description: aws.String("Description"),
				Unit:        nil,
			},
			expected: false,
		},
		{
			name: "invalid - all fields nil",
			metric: types.ResponseResourceMetric{
				Metric:      nil,
				Description: nil,
				Unit:        nil,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validResponseResourceMetric(tc.metric)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTrimStatisticFromMetricName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trim avg statistic",
			input:    "os.general.numVCPUs.avg",
			expected: "os.general.numVCPUs",
		},
		{
			name:     "trim min statistic",
			input:    "os.cpuUtilization.guest.min",
			expected: "os.cpuUtilization.guest",
		},
		{
			name:     "trim max statistic",
			input:    "os.memory.total.max",
			expected: "os.memory.total",
		},
		{
			name:     "trim sum statistic",
			input:    "db.User.max_connections.sum",
			expected: "db.User.max_connections",
		},
		{
			name:     "no statistic suffix returns empty",
			input:    "os.general.numVCPUs",
			expected: "",
		},
		{
			name:     "invalid statistic suffix returns empty",
			input:    "os.general.numVCPUs.invalid",
			expected: "",
		},
		{
			name:     "empty string returns empty",
			input:    "",
			expected: "",
		},
		{
			name:     "metric with multiple dots and avg",
			input:    "db.SQL.total_query_time.avg",
			expected: "db.SQL.total_query_time",
		},
		{
			name:     "metric with underscores and max",
			input:    "db.User.max_connections.max",
			expected: "db.User.max_connections",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := TrimStatisticFromMetricName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEngineToShortName(t *testing.T) {
	testCases := []struct {
		name     string
		engine   models.Engine
		expected string
	}{
		{
			name:     "aurora-postgresql to apg",
			engine:   models.AuroraPostgreSQL,
			expected: "apg",
		},
		{
			name:     "aurora-mysql to ams",
			engine:   models.AuroraMySQL,
			expected: "ams",
		},
		{
			name:     "postgres to pg",
			engine:   models.PostgreSQL,
			expected: "pg",
		},
		{
			name:     "mysql to mysql",
			engine:   models.MySQL,
			expected: "mysql",
		},
		{
			name:     "mariadb to mariadb",
			engine:   models.MariaDB,
			expected: "mariadb",
		},
		{
			name:     "oracle to oracle",
			engine:   models.Oracle,
			expected: "oracle",
		},
		{
			name:     "sqlserver to sqlserver",
			engine:   models.SQLServer,
			expected: "sqlserver",
		},
		{
			name:     "empty engine returns empty",
			engine:   models.Engine(""),
			expected: "",
		},
		{
			name:     "invalid engine returns empty",
			engine:   models.Engine("invalid-engine"),
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := EngineToShortName(tc.engine)
			assert.Equal(t, tc.expected, result)
		})
	}
}
