package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
)

func TestParseDimensionGroups(t *testing.T) {
	t.Run("no rules configured leaves grouping off", func(t *testing.T) {
		parsed, err := parseDimensionGroups(nil)
		require.NoError(t, err)
		assert.Nil(t, parsed)
	})

	t.Run("compiles a valid rule", func(t *testing.T) {
		parsed, err := parseDimensionGroups([]models.DimensionGroupConfig{{
			Pattern: `^db\.load\..*`,
			Group:   "db.wait_event",
			Keys:    []string{"db.wait_event.type", "db.wait_event.name"},
			Limit:   10,
		}})
		require.NoError(t, err)
		require.Len(t, parsed, 1)

		assert.Equal(t, "db.wait_event", parsed[0].Group)
		assert.Equal(t, []string{"db.wait_event.type", "db.wait_event.name"}, parsed[0].Keys)
		assert.Equal(t, int32(10), parsed[0].Limit)
		assert.True(t, parsed[0].Pattern.MatchString("db.load.avg"))
		assert.False(t, parsed[0].Pattern.MatchString("os.cpuUtilization.user.avg"))
	})

	testCases := []struct {
		name          string
		config        models.DimensionGroupConfig
		expectedError string
	}{
		{
			name:          "rejects an uncompilable pattern",
			config:        models.DimensionGroupConfig{Pattern: `^db\.load\.[`, Group: "db.wait_event", Keys: []string{"db.wait_event.type"}, Limit: 10},
			expectedError: "dimensions[0].pattern",
		},
		{
			name:          "rejects an empty group",
			config:        models.DimensionGroupConfig{Pattern: `.*`, Group: "", Keys: []string{"db.wait_event.type"}, Limit: 10},
			expectedError: "dimensions[0].group",
		},
		{
			name:          "rejects an empty key list",
			config:        models.DimensionGroupConfig{Pattern: `.*`, Group: "db.wait_event", Keys: nil, Limit: 10},
			expectedError: "dimensions[0].keys",
		},
		{
			name:          "requires a limit rather than defaulting one",
			config:        models.DimensionGroupConfig{Pattern: `.*`, Group: "db.wait_event", Keys: []string{"db.wait_event.type"}, Limit: 0},
			expectedError: "dimensions[0].limit",
		},
		{
			name:          "rejects a limit above what one response can return",
			config:        models.DimensionGroupConfig{Pattern: `.*`, Group: "db.wait_event", Keys: []string{"db.wait_event.type"}, Limit: MaxDimensionLimit + 1},
			expectedError: "dimensions[0].limit",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := parseDimensionGroups([]models.DimensionGroupConfig{testCase.config})
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedError)
		})
	}

	t.Run("reports the index of the offending rule", func(t *testing.T) {
		_, err := parseDimensionGroups([]models.DimensionGroupConfig{
			{Pattern: `^db\.load\..*`, Group: "db.wait_event", Keys: []string{"db.wait_event.type"}, Limit: 5},
			{Pattern: `.*`, Group: "db.user", Keys: []string{"db.user.name"}, Limit: 0},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dimensions[1].limit")
	})
}
