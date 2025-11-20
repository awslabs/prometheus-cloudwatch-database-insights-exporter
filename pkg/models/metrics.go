package models

import (
	"time"
)

type Metrics struct {
	MetricsDetails     map[string]MetricDetails
	MetricsList        []string // list of metricNames.statitic
	MetricsLastUpdated time.Time
	MetricsTTL         time.Duration
}

type MetricDetails struct {
	Name        string
	Description string
	Unit        string
	Statistics  []Statistic
}

type MetricData struct {
	Metric    string
	Timestamp time.Time
	Value     float64
}
