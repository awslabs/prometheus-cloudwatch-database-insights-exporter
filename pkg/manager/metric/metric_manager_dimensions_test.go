package metric

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awspi "github.com/aws/aws-sdk-go-v2/service/pi"
	"github.com/aws/aws-sdk-go-v2/service/pi/types"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils/mocks"
)

const (
	statementKey = "db.sql_tokenized.statement"
	dbIDKey      = "db.sql_tokenized.db_id"
)

// sqlBreakdownConfig groups db.load by tokenized statement and asks only for the
// statement text, so entries that differ in database id share their labels.
func sqlBreakdownConfig() *models.ParsedConfig {
	config := testutils.CreateDefaultParsedTestConfig()
	config.Discovery.Metrics.DimensionGroups = []models.ParsedDimensionGroup{{
		Pattern: regexp.MustCompile(`^db\.load\.`),
		Group:   "db.sql_tokenized",
		Keys:    []string{statementKey},
		Limit:   10,
	}}
	return config
}

func dbLoadInstance() models.Instance {
	instance := testutils.NewTestInstancePostgreSQL()
	instance.Metrics = &models.Metrics{
		MetricsDetails: map[string]models.MetricDetails{
			"db.load": testutils.NewTestMetricDetails("db.load", "Database load", "Average Active Sessions"),
		},
		MetricsList:        []string{"db.load.avg"},
		MetricsLastUpdated: time.Now(),
		MetadataTTL:        testutils.TestTTL,
		DimensionSeries:    models.NewDimensionSeriesStore(),
	}
	return instance
}

func sqlTokenizedAvailability() models.DimensionGroupAvailability {
	return models.DimensionGroupAvailability{
		"db.sql_tokenized": {statementKey: {}, dbIDKey: {}, "db.sql_tokenized.id": {}},
		"db.wait_event":    {"db.wait_event.type": {}, "db.wait_event.name": {}},
	}
}

// supportedInstance is a dbLoadInstance whose dimension lookup already ran and
// found the tokenized SQL group.
func supportedInstance() models.Instance {
	instance := dbLoadInstance()
	instance.Metrics.AvailableDimensions = map[string]models.DimensionGroupAvailability{"db.load": sqlTokenizedAvailability()}
	instance.Metrics.DimensionsLastUpdated = time.Now()
	return instance
}

// loadEntry is one entry of a GetResourceMetrics response with two datapoints a
// minute apart, enough for the manager to derive a cache TTL from.
func loadEntry(metric string, dimensions map[string]string, value float64, now time.Time) types.MetricKeyDataPoints {
	return types.MetricKeyDataPoints{
		Key: &types.ResponseResourceMetricKey{Metric: aws.String(metric), Dimensions: dimensions},
		DataPoints: []types.DataPoint{
			{Timestamp: aws.Time(now.Add(-time.Minute)), Value: aws.Float64(value)},
			{Timestamp: aws.Time(now), Value: aws.Float64(value)},
		},
	}
}

// collidingResponse holds the aggregate and two statements that only differ in
// the database they ran in.
func collidingResponse(metric string, now time.Time) *awspi.GetResourceMetricsOutput {
	return &awspi.GetResourceMetricsOutput{MetricList: []types.MetricKeyDataPoints{
		loadEntry(metric, nil, 1.0, now),
		loadEntry(metric, map[string]string{statementKey: "SELECT * FROM orders WHERE id = ?", dbIDKey: "pi-1"}, 0.3, now),
		loadEntry(metric, map[string]string{statementKey: "SELECT * FROM orders WHERE id = ?", dbIDKey: "pi-2"}, 0.4, now),
	}}
}

// batchCollector exposes one batch collection as a Prometheus collector, so the
// test sees the scrape the way the registry would check it.
type batchCollector struct {
	t        *testing.T
	manager  *MetricManager
	instance models.Instance
	batch    []string
}

func (collector batchCollector) Describe(chan<- *prometheus.Desc) {}

func (collector batchCollector) Collect(ch chan<- prometheus.Metric) {
	require.NoError(collector.t, collector.manager.CollectMetricsForBatch(context.Background(), collector.instance, collector.batch, ch))
}

func gatherGrouped(t *testing.T, collector batchCollector) []*dto.Metric {
	t.Helper()

	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(collector))

	families, err := registry.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if strings.HasSuffix(family.GetName(), "_by_db_sql_tokenized") {
			return family.GetMetric()
		}
	}
	t.Fatalf("grouped family not found in %d families", len(families))
	return nil
}

func TestCollidingDimensionsAreMergedIntoOneSeries(t *testing.T) {
	testCases := []struct {
		metric   string
		expected float64
	}{
		{metric: "db.load.avg", expected: 0.7},
		{metric: "db.load.sum", expected: 0.7},
		{metric: "db.load.max", expected: 0.4},
		{metric: "db.load.min", expected: 0.4},
	}

	for _, testCase := range testCases {
		t.Run(testCase.metric, func(t *testing.T) {
			instance := supportedInstance()
			batch := []string{testCase.metric}

			mockPI := &mocks.MockPIService{}
			manager, err := NewMetricManager(mockPI, sqlBreakdownConfig(), testutils.TestRegion)
			require.NoError(t, err)

			mockPI.On("GetResourceMetrics", mock.Anything, instance.ResourceID, batch).
				Return(collidingResponse(testCase.metric, time.Now()), nil).Once()

			series := gatherGrouped(t, batchCollector{t: t, manager: manager, instance: instance, batch: batch})
			require.Len(t, series, 1)
			assert.InDelta(t, testCase.expected, series[0].GetGauge().GetValue(), 1e-9)

			mockPI.AssertExpectations(t)
		})
	}
}

// The merged series is what gets cached, so a scrape served from the cache must
// replay one series as well, not the two entries the response carried.
func TestMergedSeriesReplayFromCacheWithoutDuplicates(t *testing.T) {
	instance := supportedInstance()
	batch := []string{"db.load.avg"}

	mockPI := &mocks.MockPIService{}
	manager, err := NewMetricManager(mockPI, sqlBreakdownConfig(), testutils.TestRegion)
	require.NoError(t, err)

	mockPI.On("GetResourceMetrics", mock.Anything, instance.ResourceID, batch).
		Return(collidingResponse("db.load.avg", time.Now()), nil).Once()

	collector := batchCollector{t: t, manager: manager, instance: instance, batch: batch}
	fetched := gatherGrouped(t, collector)
	replayed := gatherGrouped(t, collector)

	require.Len(t, replayed, 1)
	assert.Equal(t, fetched[0].GetGauge().GetValue(), replayed[0].GetGauge().GetValue())

	sets, found := instance.Metrics.DimensionSeries.Get("db.load.avg")
	require.True(t, found)
	assert.Contains(t, sets, map[string]string{statementKey: "SELECT * FROM orders WHERE id = ?"})
	assert.Len(t, sets, 2, "the aggregate and one merged series")

	mockPI.AssertExpectations(t)
}

func TestDimensionLookupRunsOncePerMetadataTTL(t *testing.T) {
	instance := dbLoadInstance()

	mockPI := &mocks.MockPIService{}
	manager, err := NewMetricManager(mockPI, sqlBreakdownConfig(), testutils.TestRegion)
	require.NoError(t, err)

	mockPI.On("ListAvailableResourceDimensions", mock.Anything, instance.ResourceID, "db.load").
		Return(sqlTokenizedAvailability(), nil).Once()

	for range 2 {
		_, err := manager.GetMetricBatches(context.Background(), instance)
		require.NoError(t, err)
	}

	assert.True(t, instance.Metrics.SupportsDimensionGroup("db.load", "db.sql_tokenized", []string{statementKey}))
	mockPI.AssertExpectations(t)
}

func TestNoDimensionLookupWithoutRules(t *testing.T) {
	instance := dbLoadInstance()

	mockPI := &mocks.MockPIService{}
	manager, err := NewMetricManager(mockPI, testutils.CreateDefaultParsedTestConfig(), testutils.TestRegion)
	require.NoError(t, err)

	_, err = manager.GetMetricBatches(context.Background(), instance)
	require.NoError(t, err)

	mockPI.AssertNotCalled(t, "ListAvailableResourceDimensions", mock.Anything, mock.Anything, mock.Anything)
}

func TestDimensionAvailabilityDecidesWhetherToGroup(t *testing.T) {
	testCases := []struct {
		name          string
		availability  models.DimensionGroupAvailability
		lookupError   error
		expectGrouped bool
	}{
		{
			name:          "group and keys available",
			availability:  sqlTokenizedAvailability(),
			expectGrouped: true,
		},
		{
			name:         "group not offered by the engine",
			availability: models.DimensionGroupAvailability{"db.wait_event": {"db.wait_event.type": {}}},
		},
		{
			name:         "group offered without the configured key",
			availability: models.DimensionGroupAvailability{"db.sql_tokenized": {dbIDKey: {}}},
		},
		{
			name:        "lookup fails, for example without the IAM permission",
			lookupError: errors.New("AccessDeniedException"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			instance := dbLoadInstance()
			batch := []string{"db.load.avg"}

			mockPI := &mocks.MockPIService{}
			manager, err := NewMetricManager(mockPI, sqlBreakdownConfig(), testutils.TestRegion)
			require.NoError(t, err)

			var availability any
			if testCase.availability != nil {
				availability = testCase.availability
			}
			mockPI.On("ListAvailableResourceDimensions", mock.Anything, instance.ResourceID, "db.load").
				Return(availability, testCase.lookupError)
			mockPI.On("GetResourceMetrics", mock.Anything, instance.ResourceID, batch).
				Return(&awspi.GetResourceMetricsOutput{MetricList: []types.MetricKeyDataPoints{
					loadEntry("db.load.avg", nil, 1.0, time.Now()),
				}}, nil)

			// A cancelled context stops the retries of a failed lookup after the
			// first attempt, which keeps the test fast. The lookup is the only
			// call that uses it, since the metric metadata is still fresh.
			lookupContext, cancel := context.WithCancel(context.Background())
			if testCase.lookupError != nil {
				cancel()
			}
			defer cancel()

			_, err = manager.GetMetricBatches(lookupContext, instance)
			require.NoError(t, err, "a failed dimension lookup must not fail metric collection")

			ch := make(chan prometheus.Metric, 10)
			require.NoError(t, manager.CollectMetricsForBatch(context.Background(), instance, batch, ch))

			if testCase.expectGrouped {
				assert.Contains(t, mockPI.LastDimensionGroups, "db.load.avg")
			} else {
				assert.Empty(t, mockPI.LastDimensionGroups)
			}
		})
	}
}
