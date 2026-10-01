package models

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type Metrics struct {
	MetricsDetails     map[string]MetricDetails
	MetricsList        []string // list of metricNames.statitic
	MetricsLastUpdated time.Time
	MetadataTTL        time.Duration

	// DimensionSeries records, per metric name, the dimension sets the last
	// response carried. Grouped metrics produce one series per dimension, so the
	// cache cannot be consulted by metric name alone; this is what lets a cache
	// hit replay every series the metric produced.
	//
	// It is a pointer so that Metrics stays copyable.
	DimensionSeries *DimensionSeriesStore
}

// DimensionSeriesStore remembers which dimension sets a grouped metric returned.
// Batches of one instance are collected by separate workers, so access is guarded.
type DimensionSeriesStore struct {
	mutex  sync.RWMutex
	series map[string][]map[string]string
}

func NewDimensionSeriesStore() *DimensionSeriesStore {
	return &DimensionSeriesStore{series: make(map[string][]map[string]string)}
}

// Record replaces the dimension sets stored for a metric.
func (store *DimensionSeriesStore) Record(metricName string, dimensionSets []map[string]string) {
	if store == nil {
		return
	}

	store.mutex.Lock()
	defer store.mutex.Unlock()

	if store.series == nil {
		store.series = make(map[string][]map[string]string)
	}
	store.series[metricName] = dimensionSets
}

// Get returns the dimension sets last recorded for a metric, and false when the
// metric has not been seen grouped yet.
func (store *DimensionSeriesStore) Get(metricName string) ([]map[string]string, bool) {
	if store == nil {
		return nil, false
	}

	store.mutex.RLock()
	defer store.mutex.RUnlock()

	dimensionSets, found := store.series[metricName]
	return dimensionSets, found
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

	// Dimensions is empty for ungrouped metrics. For a metric queried with a
	// dimension group it holds the dimension keys and values the API returned,
	// for example {"db.wait_event.type": "IO"}.
	Dimensions map[string]string

	// DimensionGroup is the PI group that produced Dimensions. It is empty for
	// ungrouped metrics and is used to keep grouped series under their own
	// exported metric name.
	DimensionGroup string

	// DimensionKeys is the configured key list for the group. Label names are
	// taken from it rather than from the response, so every series of a grouped
	// metric carries the same labels even when the API omits one; a Prometheus
	// descriptor requires a stable label set per metric name.
	DimensionKeys []string
}

// DimensionFingerprint renders Dimensions as a stable string so a metric's
// grouped series can be told apart in a map key. Keys are sorted, so the result
// does not depend on map iteration order.
func (metricData MetricData) DimensionFingerprint() string {
	return FingerprintDimensions(metricData.Dimensions)
}

// FingerprintDimensions renders a dimension set as a stable string. The empty
// set renders as the empty string, which keeps ungrouped metrics on the key they
// already used.
func FingerprintDimensions(dimensions map[string]string) string {
	if len(dimensions) == 0 {
		return ""
	}

	keys := make([]string, 0, len(dimensions))
	for key := range dimensions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for index, key := range keys {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(dimensions[key])
	}
	return builder.String()
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
