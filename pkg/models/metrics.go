package models

import (
	"strings"
	"time"
)

type Metrics struct {
	MetricsDetails     map[string]MetricDetails
	MetricsList        []string // list of metricNames.statitic
	MetricsLastUpdated time.Time
	MetadataTTL        time.Duration
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

func (metric MetricDetails) GetFilterableFields() map[string]string {
	category := DeriveMetricCategory(metric.Name)
	return map[string]string{
		"name":     metric.Name,
		"category": category,
		"unit":     metric.Unit,
	}
}

func (metric MetricDetails) GetFilterableTags() map[string]string {
	// Metrics don't have tags, returns empty
	return make(map[string]string)
}

func DeriveMetricCategory(metricName string) string {
	if strings.HasPrefix(metricName, "os.") {
		return "os"
	}
	if strings.HasPrefix(metricName, "db.") {
		return "db"
	}
	return "other"
}
