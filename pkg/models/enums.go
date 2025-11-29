package models

import "strings"

type MetricType string

const (
	MetricTypeDB           MetricType = "db"
	MetricTypeOS           MetricType = "os"
	MetricTypePerSQL       MetricType = "db.sql.stats"
	MetricTypePerSQLDigest MetricType = "db.sql_tokenized.stats"
)

type Engine string

const (
	AuroraPostgreSQL Engine = "aurora-postgresql"
	AuroraMySQL      Engine = "aurora-mysql"
	PostgreSQL       Engine = "postgres"
	MySQL            Engine = "mysql"
	MariaDB          Engine = "mariadb"
	Oracle           Engine = "oracle"
	SQLServer        Engine = "sqlserver"
)

type Statistic string

const (
	StatisticAvg Statistic = "avg"
	StatisticMin Statistic = "min"
	StatisticMax Statistic = "max"
	StatisticSum Statistic = "sum"
)

func NewEngine(engineString string) Engine {
	// Full string match for specific engines
	switch engineString {
	case string(AuroraPostgreSQL), string(AuroraMySQL), string(PostgreSQL), string(MySQL), string(MariaDB):
		return Engine(engineString)
	}

	// Partial match for Oracle and SQL Server (case-insensitive)
	lowerEngine := strings.ToLower(engineString)
	if strings.Contains(lowerEngine, "oracle") {
		return Oracle
	}
	if strings.Contains(lowerEngine, "sqlserver") {
		return SQLServer
	}

	return ""
}

func (engine Engine) IsValid() bool {
	switch engine {
	case AuroraPostgreSQL, AuroraMySQL, PostgreSQL, MySQL, MariaDB, Oracle, SQLServer:
		return true
	default:
		return false
	}
}

func NewStatistic(statisticString string) Statistic {
	statistic := Statistic(statisticString)
	if !statistic.IsValid() {
		return ""
	}
	return statistic
}

func (statistic Statistic) String() string {
	return string(statistic)
}

func (statistic Statistic) IsValid() bool {
	switch statistic {
	case StatisticAvg, StatisticMin, StatisticMax, StatisticSum:
		return true
	default:
		return false
	}
}

func GetAllStatistics() []Statistic {
	return []Statistic{StatisticAvg, StatisticMin, StatisticMax, StatisticSum}
}

type FilterType string

const (
	FilterTypeIdentifier FilterType = "identifier"
	FilterTypeEngine     FilterType = "engine"
	FilterTypeName       FilterType = "name"
	FilterTypeCategory   FilterType = "category"
	FilterTypeUnit       FilterType = "unit"
	FilterTypeTagPrefix  FilterType = "tag."
)

func (filterType FilterType) String() string {
	return string(filterType)
}

func (filterType FilterType) IsValid() bool {
	switch filterType {
	case FilterTypeIdentifier, FilterTypeEngine, FilterTypeName, FilterTypeCategory, FilterTypeUnit:
		return true
	default:
		return strings.HasPrefix(string(filterType), string(FilterTypeTagPrefix))
	}
}
