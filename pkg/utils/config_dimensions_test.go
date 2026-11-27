package utils

import (
	"fmt"
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
		{
			name: "rejects more keys than one GroupBy accepts",
			config: models.DimensionGroupConfig{Pattern: `.*`, Group: "db.sql", Limit: 10, Keys: []string{
				"db.sql.k1", "db.sql.k2", "db.sql.k3", "db.sql.k4", "db.sql.k5", "db.sql.k6",
				"db.sql.k7", "db.sql.k8", "db.sql.k9", "db.sql.k10", "db.sql.k11",
			}},
			expectedError: "at most 10 keys",
		},
		{
			name:          "rejects a key from another group",
			config:        models.DimensionGroupConfig{Pattern: `.*`, Group: "db.wait_event", Keys: []string{"db.user.name"}, Limit: 10},
			expectedError: "does not belong to group 'db.wait_event'",
		},
		{
			name:          "rejects a key that only shares the group name as a prefix",
			config:        models.DimensionGroupConfig{Pattern: `.*`, Group: "db.wait_event", Keys: []string{"db.wait_event_type.name"}, Limit: 10},
			expectedError: "does not belong to group 'db.wait_event'",
		},
		{
			name:          "rejects the bare group prefix",
			config:        models.DimensionGroupConfig{Pattern: `.*`, Group: "db.wait_event", Keys: []string{"db.wait_event."}, Limit: 10},
			expectedError: "does not belong to group",
		},
		{
			name:          "rejects a duplicate key",
			config:        models.DimensionGroupConfig{Pattern: `.*`, Group: "db.wait_event", Keys: []string{"db.wait_event.type", "db.wait_event.type"}, Limit: 10},
			expectedError: "listed more than once",
		},
		{
			name:          "rejects keys that map to the same label",
			config:        models.DimensionGroupConfig{Pattern: `.*`, Group: "db.wait_event", Keys: []string{"db.wait_event.type", "db.wait_event.Type"}, Limit: 10},
			expectedError: "both map to label 'db_wait_event_type'",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := parseDimensionGroups([]models.DimensionGroupConfig{testCase.config})
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedError)
		})
	}

	t.Run("accepts the keys of a group that are not prefixed with its name", func(t *testing.T) {
		parsed, err := parseDimensionGroups([]models.DimensionGroupConfig{{
			Pattern: `^db\.load\..*`,
			Group:   "db.query_tokenized",
			Keys:    []string{"db.query.tokenized.id", "db.query.tokenized.statement"},
			Limit:   10,
		}})
		require.NoError(t, err)
		assert.Len(t, parsed, 1)
	})

	t.Run("accepts the maximum number of keys", func(t *testing.T) {
		keys := make([]string, MaxDimensionKeys)
		for index := range keys {
			keys[index] = fmt.Sprintf("db.sql.k%d", index)
		}
		_, err := parseDimensionGroups([]models.DimensionGroupConfig{{Pattern: `.*`, Group: "db.sql", Keys: keys, Limit: 10}})
		require.NoError(t, err)
	})

	t.Run("reports the index of the offending rule", func(t *testing.T) {
		_, err := parseDimensionGroups([]models.DimensionGroupConfig{
			{Pattern: `^db\.load\..*`, Group: "db.wait_event", Keys: []string{"db.wait_event.type"}, Limit: 5},
			{Pattern: `.*`, Group: "db.user", Keys: []string{"db.user.name"}, Limit: 0},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dimensions[1].limit")
	})
}
