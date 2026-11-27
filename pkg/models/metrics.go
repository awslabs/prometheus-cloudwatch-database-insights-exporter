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

	// AvailableDimensions holds, per metric type such as db.load, the dimension
	// groups the instance supports. Some groups exist only for some engines, and
	// a GroupBy the engine does not support fails every metric in its request.
	// A metric type that is missing has not been looked up, or the lookup
	// failed, and its metrics are queried ungrouped.
	AvailableDimensions   map[string]DimensionGroupAvailability
	DimensionsLastUpdated time.Time
}

// DimensionGroupAvailability maps a dimension group to the dimensions it offers.
type DimensionGroupAvailability map[string]map[string]struct{}

// Supports reports whether the group exists and offers every one of the keys.
func (availability DimensionGroupAvailability) Supports(group string, keys []string) bool {
	dimensions, found := availability[group]
	if !found {
		return false
	}
	for _, key := range keys {
		if _, offered := dimensions[key]; !offered {
			return false
		}
	}
	return true
}

// SupportsDimensionGroup reports whether metrics of the given type can be
// queried with the group and keys on this instance.
func (metrics *Metrics) SupportsDimensionGroup(metricType string, group string, keys []string) bool {
	if metrics == nil {
		return false
	}
	availability, found := metrics.AvailableDimensions[metricType]
	return found && availability.Supports(group, keys)
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

// ProjectDimensions narrows a returned dimension set to the configured keys. A
// key the response left out maps to the empty string, the same value its label
// is exported with. The result identifies a grouped series: response entries
// that agree on every configured key end up with identical labels, so they have
// to be one series in the cache as well.
func ProjectDimensions(dimensions map[string]string, keys []string) map[string]string {
	projected := make(map[string]string, len(keys))
	for _, key := range keys {
		projected[key] = dimensions[key]
	}
	return projected
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
