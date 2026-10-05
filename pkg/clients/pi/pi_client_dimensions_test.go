package pi

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
)

func TestBuildMetricQueries(t *testing.T) {
	waitEvent := &models.ParsedDimensionGroup{
		Pattern: regexp.MustCompile(`^db\.load\..*`),
		Group:   "db.wait_event",
		Keys:    []string{"db.wait_event.type", "db.wait_event.name"},
		Limit:   10,
	}

	t.Run("leaves queries ungrouped when nothing is configured", func(t *testing.T) {
		queries := buildMetricQueries([]string{"db.load.avg", "os.cpuUtilization.idle.avg"}, nil)
		require.Len(t, queries, 2)

		for _, query := range queries {
			assert.Nil(t, query.GroupBy)
		}
	})

	t.Run("attaches GroupBy only to the configured metric", func(t *testing.T) {
		queries := buildMetricQueries(
			[]string{"db.load.avg", "os.cpuUtilization.idle.avg"},
			map[string]*models.ParsedDimensionGroup{"db.load.avg": waitEvent},
		)
		require.Len(t, queries, 2)

		require.NotNil(t, queries[0].GroupBy)
		assert.Equal(t, "db.load.avg", *queries[0].Metric)
		assert.Equal(t, "db.wait_event", *queries[0].GroupBy.Group)
		assert.Equal(t, []string{"db.wait_event.type", "db.wait_event.name"}, queries[0].GroupBy.Dimensions)
		assert.Equal(t, int32(10), *queries[0].GroupBy.Limit)

		assert.Nil(t, queries[1].GroupBy)
	})

	t.Run("a nil entry leaves the metric ungrouped", func(t *testing.T) {
		queries := buildMetricQueries(
			[]string{"db.load.avg"},
			map[string]*models.ParsedDimensionGroup{"db.load.avg": nil},
		)
		require.Len(t, queries, 1)
		assert.Nil(t, queries[0].GroupBy)
	})

	t.Run("grouping does not change the number of queries sent", func(t *testing.T) {
		metricNames := []string{"db.load.avg", "os.cpuUtilization.idle.avg", "db.Cache.blks_hit.avg"}

		ungrouped := buildMetricQueries(metricNames, nil)
		grouped := buildMetricQueries(metricNames, map[string]*models.ParsedDimensionGroup{"db.load.avg": waitEvent})

		assert.Equal(t, len(ungrouped), len(grouped))
	})
}
