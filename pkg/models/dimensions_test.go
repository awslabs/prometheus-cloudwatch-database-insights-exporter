package models

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFingerprintDimensions(t *testing.T) {
	testCases := []struct {
		name       string
		dimensions map[string]string
		expected   string
	}{
		{
			name:       "nil renders empty so ungrouped metrics keep their existing key",
			dimensions: nil,
			expected:   "",
		},
		{
			name:       "empty renders empty",
			dimensions: map[string]string{},
			expected:   "",
		},
		{
			name:       "single dimension",
			dimensions: map[string]string{"db.wait_event.type": "IO"},
			expected:   "db.wait_event.type=IO",
		},
		{
			name: "keys are sorted so the result does not depend on map order",
			dimensions: map[string]string{
				"db.wait_event.name": "DataFileRead",
				"db.wait_event.type": "IO",
			},
			expected: "db.wait_event.name=DataFileRead,db.wait_event.type=IO",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, FingerprintDimensions(testCase.dimensions))
		})
	}
}

func TestFingerprintDimensionsIsStableAcrossCalls(t *testing.T) {
	dimensions := map[string]string{
		"db.wait_event.name": "DataFileRead",
		"db.wait_event.type": "IO",
		"db.user.name":       "reader",
	}

	first := FingerprintDimensions(dimensions)
	for i := 0; i < 50; i++ {
		assert.Equal(t, first, FingerprintDimensions(dimensions))
	}
}

func TestDimensionGroupFor(t *testing.T) {
	config := ParsedMetricsConfig{
		DimensionGroups: []ParsedDimensionGroup{
			{Pattern: regexp.MustCompile(`^db\.load\..*`), Group: "db.wait_event"},
			{Pattern: regexp.MustCompile(`^db\..*`), Group: "db.user"},
		},
	}

	t.Run("returns the first matching rule", func(t *testing.T) {
		group := config.DimensionGroupFor("db.load.avg")
		assert.NotNil(t, group)
		assert.Equal(t, "db.wait_event", group.Group)
	})

	t.Run("falls through to a later rule", func(t *testing.T) {
		group := config.DimensionGroupFor("db.cache.blks_hit.avg")
		assert.NotNil(t, group)
		assert.Equal(t, "db.user", group.Group)
	})

	t.Run("returns nil when nothing matches", func(t *testing.T) {
		assert.Nil(t, config.DimensionGroupFor("os.cpuUtilization.user.avg"))
	})

	t.Run("returns nil when no rules are configured", func(t *testing.T) {
		assert.Nil(t, (&ParsedMetricsConfig{}).DimensionGroupFor("db.load.avg"))
	})
}

func TestDimensionSeriesStore(t *testing.T) {
	t.Run("records and returns dimension sets", func(t *testing.T) {
		store := NewDimensionSeriesStore()
		sets := []map[string]string{{}, {"db.wait_event.type": "IO"}}

		store.Record("db.load.avg", sets)

		got, found := store.Get("db.load.avg")
		assert.True(t, found)
		assert.Equal(t, sets, got)
	})

	t.Run("reports unknown metrics", func(t *testing.T) {
		store := NewDimensionSeriesStore()
		_, found := store.Get("db.load.avg")
		assert.False(t, found)
	})

	t.Run("a nil store is safe to use", func(t *testing.T) {
		var store *DimensionSeriesStore
		assert.NotPanics(t, func() { store.Record("db.load.avg", nil) })

		_, found := store.Get("db.load.avg")
		assert.False(t, found)
	})

	t.Run("recording replaces the previous sets", func(t *testing.T) {
		store := NewDimensionSeriesStore()
		store.Record("db.load.avg", []map[string]string{{"db.wait_event.type": "IO"}})
		store.Record("db.load.avg", []map[string]string{{"db.wait_event.type": "CPU"}})

		got, _ := store.Get("db.load.avg")
		assert.Equal(t, []map[string]string{{"db.wait_event.type": "CPU"}}, got)
	})
}

func TestMetricDataDimensionFingerprint(t *testing.T) {
	ungrouped := MetricData{Metric: "db.load.avg"}
	assert.Equal(t, "", ungrouped.DimensionFingerprint())

	grouped := MetricData{
		Metric:     "db.load.avg",
		Dimensions: map[string]string{"db.wait_event.type": "IO"},
	}
	assert.Equal(t, "db.wait_event.type=IO", grouped.DimensionFingerprint())
}
