package utils

import (
	"testing"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

var (
	testMetricNames = []string{
		"db.SQL.Innodb_rows_read",
		"db.SQL.Innodb_rows_inserted",
		"db.SQL.Innodb_rows_updated",
		"db.SQL.Innodb_rows_deleted",
		"db.SQL.Select_scan",
		"db.SQL.Select_full_join",
		"db.SQL.Select_range",
		"db.SQL.Select_range_check",
		"db.SQL.Sort_merge_passes",
		"db.SQL.Sort_range",
		"db.SQL.Sort_rows",
		"db.SQL.Sort_scan",
		"db.SQL.Created_tmp_disk_tables",
		"db.SQL.Created_tmp_files",
		"db.SQL.Created_tmp_tables",
	}

	testLargeMetricNames = []string{
		"metric1", "metric2", "metric3", "metric4", "metric5",
		"metric6", "metric7", "metric8", "metric9", "metric10",
		"metric11", "metric12", "metric13", "metric14", "metric15",
		"metric16", "metric17", "metric18", "metric19", "metric20",
		"metric21", "metric22", "metric23", "metric24", "metric25",
		"metric26", "metric27", "metric28", "metric29", "metric30",
		"metric31", "metric32", "metric33", "metric34", "metric35",
	}

	testSingleMetric = []string{"db.SQL.Innodb_rows_read"}
)

func TestSnakeCase(t *testing.T) {
	t.Run("metric names", func(t *testing.T) {
		for i, metricName := range testutils.TestMetricNamesWithStats {
			expected := testutils.TestSnakeCaseMetricNamesWithStats[i]
			t.Run(metricName, func(t *testing.T) {
				result := SnakeCase(metricName)
				assert.Equal(t, expected, result)
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "special characters",
				input:    "metric.with-special@chars!",
				expected: "metric_withspecialchars",
			},
			{
				name:     "numbers and underscores",
				input:    "metric_123.test_456",
				expected: "metric_123_test_456",
			},
			{
				name:     "empty string",
				input:    "",
				expected: "",
			},
			{
				name:     "only dots",
				input:    "...",
				expected: "___",
			},
			{
				name:     "only special chars",
				input:    "@#$%",
				expected: "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := SnakeCase(tc.input)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

func TestBatchMetricNames(t *testing.T) {
	t.Run("constant batch size scenarios", func(t *testing.T) {
		tests := []struct {
			name                  string
			metricNames           []string
			expectedBatches       int
			expectedLastBatchSize int
			expectedTotalMetrics  int
		}{
			{
				name:                  "small metric list - single batch",
				metricNames:           []string{"metric1", "metric2", "metric3"},
				expectedBatches:       1,
				expectedLastBatchSize: 3,
				expectedTotalMetrics:  3,
			},
			{
				name:                  "exactly batch size - single batch",
				metricNames:           testMetricNames,
				expectedBatches:       1,
				expectedLastBatchSize: 15,
				expectedTotalMetrics:  15,
			},
			{
				name:                  "large metric list - multiple batches",
				metricNames:           testLargeMetricNames,
				expectedBatches:       3,
				expectedLastBatchSize: 5,
				expectedTotalMetrics:  35,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				batches := BatchMetricNames(tt.metricNames, BatchSize)

				assert.Len(t, batches, tt.expectedBatches)

				if tt.expectedBatches > 0 {
					lastBatch := batches[len(batches)-1]
					assert.Len(t, lastBatch, tt.expectedLastBatchSize)

					for i, batch := range batches {
						if i < len(batches)-1 {
							assert.Len(t, batch, BatchSize, "all batches except the last should have size %d", BatchSize)
						}
					}

					totalMetrics := 0
					for _, batch := range batches {
						totalMetrics += len(batch)
					}
					assert.Equal(t, tt.expectedTotalMetrics, totalMetrics)

					allBatchedMetrics := make([]string, 0, totalMetrics)
					for _, batch := range batches {
						allBatchedMetrics = append(allBatchedMetrics, batch...)
					}
					assert.ElementsMatch(t, tt.metricNames, allBatchedMetrics)
				}
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		tests := []struct {
			name                  string
			metricNames           []string
			expectedBatches       int
			expectedLastBatchSize int
			expectedTotalMetrics  int
		}{
			{
				name:                  "empty metrics list",
				metricNames:           []string{},
				expectedBatches:       0,
				expectedLastBatchSize: 0,
				expectedTotalMetrics:  0,
			},
			{
				name:                  "nil metrics list",
				metricNames:           nil,
				expectedBatches:       0,
				expectedLastBatchSize: 0,
				expectedTotalMetrics:  0,
			},
			{
				name:                  "single metric",
				metricNames:           testSingleMetric,
				expectedBatches:       1,
				expectedLastBatchSize: 1,
				expectedTotalMetrics:  1,
			},
			{
				name:                  "small metric count",
				metricNames:           []string{"m1", "m2", "m3", "m4", "m5"},
				expectedBatches:       1,
				expectedLastBatchSize: 5,
				expectedTotalMetrics:  5,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				batches := BatchMetricNames(tt.metricNames, BatchSize)

				assert.Len(t, batches, tt.expectedBatches)

				if tt.expectedBatches > 0 {
					lastBatch := batches[len(batches)-1]
					assert.Len(t, lastBatch, tt.expectedLastBatchSize)

					totalMetrics := 0
					for _, batch := range batches {
						totalMetrics += len(batch)
					}
					assert.Equal(t, tt.expectedTotalMetrics, totalMetrics)

					allBatchedMetrics := make([]string, 0, totalMetrics)
					for _, batch := range batches {
						allBatchedMetrics = append(allBatchedMetrics, batch...)
					}
					assert.ElementsMatch(t, tt.metricNames, allBatchedMetrics)
				}
			})
		}
	})
}
