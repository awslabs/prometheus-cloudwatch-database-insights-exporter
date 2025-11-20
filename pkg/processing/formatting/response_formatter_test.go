package formatting

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
)

func TestConvertToPrometheusMetric(t *testing.T) {
	t.Run("converts metrics successfully", func(t *testing.T) {
		for _, metricData := range testutils.TestMetricData {
			t.Run(metricData.Metric, func(t *testing.T) {
				ch := make(chan prometheus.Metric, 1)

				err := ConvertToPrometheusMetric(ch, testutils.TestInstancePostgreSQL, metricData)
				assert.NoError(t, err)

				select {
				case metric := <-ch:
					assert.NotNil(t, metric)
					assert.NotNil(t, metric.Desc())
				default:
					t.Errorf("Expected a metric to be sent to the channel for test case %s", metricData.Metric)
				}
			})
		}
	})

	t.Run("db metrics include engine short name in metric name", func(t *testing.T) {
		dbMetric := testutils.NewTestMetricData("db.User.max_connections.avg", 100.0)
		ch := make(chan prometheus.Metric, 1)

		err := ConvertToPrometheusMetric(ch, testutils.TestInstancePostgreSQL, dbMetric)
		assert.NoError(t, err)

		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			desc := metric.Desc()
			assert.NotNil(t, desc)

			// Verify the metric name includes the engine short string (apg for Aurora PostgreSQL)
			metricName := desc.String()
			assert.Contains(t, metricName, "dbi_apg_db_user_max_connections_avg",
				"db. metrics should include engine short name in the metric name")
		default:
			t.Error("Expected a metric to be sent to the channel")
		}
	})

	t.Run("os metrics do not include engine short name in metric name", func(t *testing.T) {
		osMetric := testutils.NewTestMetricData("os.general.numVCPUs.avg", 4.0)
		ch := make(chan prometheus.Metric, 1)

		err := ConvertToPrometheusMetric(ch, testutils.TestInstancePostgreSQL, osMetric)
		assert.NoError(t, err)

		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			desc := metric.Desc()
			assert.NotNil(t, desc)

			// Verify the metric name does NOT include the engine short string for os metrics
			metricName := desc.String()
			assert.Contains(t, metricName, "dbi_os_general_numvcpus_avg",
				"os. metrics should not include engine short name in the metric name")
			assert.False(t, strings.Contains(metricName, "dbi_pg_os_"),
				"os. metrics should not have engine prefix")
		default:
			t.Error("Expected a metric to be sent to the channel")
		}
	})

	t.Run("db metrics with different engines have different prefixes", func(t *testing.T) {
		dbMetric := testutils.NewTestMetricData("db.User.max_connections.avg", 100.0)

		// Test with Aurora PostgreSQL instance (has apg prefix)
		chPg := make(chan prometheus.Metric, 1)
		err := ConvertToPrometheusMetric(chPg, testutils.TestInstancePostgreSQL, dbMetric)
		assert.NoError(t, err)

		metricPg := <-chPg
		descPg := metricPg.Desc().String()
		assert.Contains(t, descPg, "dbi_apg_db_user_max_connections_avg",
			"Aurora PostgreSQL db metrics should have apg prefix")

		// Test with Aurora MySQL instance (has ams prefix)
		// Create a MySQL instance with the full metrics details
		mysqlInstance := testutils.NewTestInstance("db-TESTMYSQL", "test-mysql-db", testutils.TestEngineMySQL)
		chMysql := make(chan prometheus.Metric, 1)
		err = ConvertToPrometheusMetric(chMysql, mysqlInstance, dbMetric)
		assert.NoError(t, err)

		metricMysql := <-chMysql
		descMysql := metricMysql.Desc().String()
		assert.Contains(t, descMysql, "dbi_mysql_db_user_max_connections_avg",
			"MySQL db metrics should have mysql prefix")
	})
}

func TestBuildPrometheusDescription(t *testing.T) {
	testCases := []struct {
		name           string
		metricName     string
		description    string
		labels         []string
		expectedName   string
		expectedDesc   string
		expectedLabels []string
	}{
		{
			name:           "builds description with all fields",
			metricName:     "dbi_os_general_numvcpus_avg",
			description:    "The number of virtual CPUs for the DB instance",
			labels:         []string{"identifier", "engine", "unit"},
			expectedName:   "dbi_os_general_numvcpus_avg",
			expectedDesc:   "The number of virtual CPUs for the DB instance",
			expectedLabels: []string{"identifier", "engine", "unit"},
		},
		{
			name:           "builds description with empty labels",
			metricName:     "dbi_test_metric",
			description:    "Test description",
			labels:         []string{},
			expectedName:   "dbi_test_metric",
			expectedDesc:   "Test description",
			expectedLabels: []string{},
		},
		{
			name:           "builds description with nil labels",
			metricName:     "dbi_test_metric",
			description:    "Test description",
			labels:         nil,
			expectedName:   "dbi_test_metric",
			expectedDesc:   "Test description",
			expectedLabels: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildPrometheusDescription(tc.metricName, tc.description, tc.labels)
			expected := prometheus.NewDesc(tc.expectedName, tc.expectedDesc, tc.expectedLabels, nil)

			assert.Equal(t, expected, result)
		})
	}
}

func TestBuildPrometheusMetricName(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		engineShortStr string
		expected       string
	}{
		{
			name:           "os metric with statistics",
			input:          "os.general.numVCPUs.avg",
			engineShortStr: "apg",
			expected:       "dbi_os_general_numvcpus_avg",
		},
		{
			name:           "os metric with different statistic",
			input:          "os.cpuUtilization.guest.max",
			engineShortStr: "apg",
			expected:       "dbi_os_cpuutilization_guest_max",
		},
		{
			name:           "db metric with apg engine",
			input:          "db.User.max_connections.avg",
			engineShortStr: "apg",
			expected:       "dbi_apg_db_user_max_connections_avg",
		},
		{
			name:           "db metric with mysql engine",
			input:          "db.User.max_connections.avg",
			engineShortStr: "mysql",
			expected:       "dbi_mysql_db_user_max_connections_avg",
		},
		{
			name:           "db metric with pg engine",
			input:          "db.SQL.total_query_time.sum",
			engineShortStr: "pg",
			expected:       "dbi_pg_db_sql_total_query_time_sum",
		},
		{
			name:           "os metric with mixed case",
			input:          "os.cpuUtilization.idle.avg",
			engineShortStr: "apg",
			expected:       "dbi_os_cpuutilization_idle_avg",
		},
		{
			name:           "empty metric name",
			input:          "",
			engineShortStr: "apg",
			expected:       "dbi_",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildPrometheusMetricName(tc.input, tc.engineShortStr)
			assert.Equal(t, tc.expected, result)
		})
	}
}
