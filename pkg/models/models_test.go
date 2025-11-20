package models

import (
	"testing"

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
		// Full string match tests
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
		// Partial match tests for Oracle (case-insensitive)
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
		// Partial match tests for SQL Server (case-insensitive)
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
		// Invalid cases
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.statistic.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
