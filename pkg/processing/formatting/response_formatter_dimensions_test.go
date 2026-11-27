package formatting

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
)

// dbLoadInstance mirrors the shared PostgreSQL fixture but knows about db.load,
// which is the metric a wait-event breakdown applies to.
func dbLoadInstance() models.Instance {
	instance := testutils.TestInstancePostgreSQL
	instance.Metrics = &models.Metrics{
		MetricsDetails: map[string]models.MetricDetails{
			"db.load": testutils.NewTestMetricDetails("db.load", "Database load", "Average Active Sessions"),
		},
		MetadataTTL:     testutils.TestTTL,
		DimensionSeries: models.NewDimensionSeriesStore(),
	}
	return instance
}

func groupedMetricData(value float64, dimensions map[string]string, keys []string) models.MetricData {
	metricData := testutils.NewTestMetricData("db.load.avg", value)
	metricData.Dimensions = dimensions
	metricData.DimensionKeys = keys
	metricData.DimensionGroup = "db.wait_event"
	return metricData
}

func collect(t *testing.T, metricData models.MetricData) prometheus.Metric {
	t.Helper()

	ch := make(chan prometheus.Metric, 1)
	require.NoError(t, ConvertToPrometheusMetric(ch, dbLoadInstance(), metricData, "dbi"))

	select {
	case metric := <-ch:
		return metric
	default:
		t.Fatal("expected a metric on the channel")
		return nil
	}
}

func TestGroupedSeriesUseTheirOwnMetricName(t *testing.T) {
	grouped := collect(t, groupedMetricData(1.5,
		map[string]string{"db.wait_event.type": "IO"},
		[]string{"db.wait_event.type"}))

	assert.Contains(t, grouped.Desc().String(), "db_load_avg_by_db_wait_event")

	ungrouped := collect(t, testutils.NewTestMetricData("db.load.avg", 4.0))
	assert.NotContains(t, ungrouped.Desc().String(), "_by_db_wait_event")
}

func TestGroupedSeriesCarryDimensionLabels(t *testing.T) {
	metric := collect(t, groupedMetricData(1.5,
		map[string]string{"db.wait_event.type": "IO", "db.wait_event.name": "DataFileRead"},
		[]string{"db.wait_event.type", "db.wait_event.name"}))

	var written dto.Metric
	require.NoError(t, metric.Write(&written))

	labels := map[string]string{}
	for _, label := range written.GetLabel() {
		labels[label.GetName()] = label.GetValue()
	}

	assert.Equal(t, "IO", labels["db_wait_event_type"])
	assert.Equal(t, "DataFileRead", labels["db_wait_event_name"])
	assert.Equal(t, testutils.TestInstancePostgreSQL.Identifier, labels["identifier"])
	assert.Equal(t, 1.5, written.GetGauge().GetValue())
}

// A dimension the API left out must still appear as a label, otherwise two
// series of the same metric would carry different label sets and the registry
// would reject one of them.
func TestLabelSetStaysStableWhenADimensionIsMissing(t *testing.T) {
	complete := collect(t, groupedMetricData(1.0,
		map[string]string{"db.wait_event.type": "IO", "db.wait_event.name": "DataFileRead"},
		[]string{"db.wait_event.type", "db.wait_event.name"}))

	partial := collect(t, groupedMetricData(2.0,
		map[string]string{"db.wait_event.type": "CPU"},
		[]string{"db.wait_event.type", "db.wait_event.name"}))

	assert.Equal(t, complete.Desc().String(), partial.Desc().String())

	var written dto.Metric
	require.NoError(t, partial.Write(&written))

	labels := map[string]string{}
	for _, label := range written.GetLabel() {
		labels[label.GetName()] = label.GetValue()
	}

	assert.Equal(t, "CPU", labels["db_wait_event_type"])
	assert.Contains(t, labels, "db_wait_event_name")
	assert.Equal(t, "", labels["db_wait_event_name"])
}

func TestBuildPrometheusMetricNameWithDimensionGroup(t *testing.T) {
	testCases := []struct {
		name     string
		metric   string
		group    string
		expected string
	}{
		{
			name:     "no group leaves the name untouched",
			metric:   "db.load.avg",
			group:    "",
			expected: "dbi_pg_db_load_avg",
		},
		{
			name:     "group is appended",
			metric:   "db.load.avg",
			group:    "db.wait_event",
			expected: "dbi_pg_db_load_avg_by_db_wait_event",
		},
		{
			name:     "os metrics do not gain the engine prefix",
			metric:   "os.cpuUtilization.user.avg",
			group:    "db.user",
			expected: "dbi_os_cpuutilization_user_avg_by_db_user",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := buildPrometheusMetricName("dbi", "pg", testCase.metric, testCase.group)
			assert.Equal(t, testCase.expected, strings.ToLower(result))
		})
	}
}
