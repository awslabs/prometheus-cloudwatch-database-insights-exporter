package cache

import (
	"container/heap"
	"log"
	"sync"
	"time"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/utils"
)

// TimeProvider is an interface for getting the current time.
// This allows for dependency injection and time mocking in tests.
type TimeProvider interface {
	Now() time.Time
}

// RealTimeProvider implements TimeProvider using the actual system time.
type RealTimeProvider struct{}

// Now returns the current system time.
func (r *RealTimeProvider) Now() time.Time {
	return time.Now()
}

// CacheKey uniquely identifies a metric data entry in the cache.
// It includes instance identifier, metric name, and aggregation statistic.
// Note: Region is not included as each MetricManager is already scoped to a specific region.
type CacheKey struct {
	Instance   string
	MetricName string
	Statistic  string

	// Dimensions distinguishes the series of a metric queried with a dimension
	// group. It is the empty string for ungrouped metrics, so keys built before
	// dimension support are unaffected.
	Dimensions string
}

// MetricCacheEntry represents a cached metric value with its metadata.
type MetricCacheEntry struct {
	Value     float64   // The cached metric value
	Timestamp time.Time // Original metric timestamp from Performance Insights
	ExpiresAt time.Time // Cache expiration timestamp
}

// MetricCache defines the interface for metric data caching operations.
type MetricCache interface {
	// Get retrieves a metric entry from the cache.
	// Returns the entry and true if found and not expired, otherwise returns false.
	Get(key CacheKey) (MetricCacheEntry, bool)

	// Peek retrieves a metric entry from the cache without removing expired entries.
	// Returns the entry and true if found (even if expired), otherwise returns false.
	// Also returns whether the entry is expired.
	Peek(key CacheKey) (MetricCacheEntry, bool, bool)

	// Set stores a metric entry in the cache with the specified TTL.
	// If the cache is at maximum size, expired entries are evicted first.
	Set(key CacheKey, value float64, timestamp time.Time, ttl time.Duration)

	// EvictExpired removes up to maxToEvict expired entries from the cache.
	// Returns the number of entries actually evicted.
	EvictExpired(maxToEvict int) int

	// Size returns the current number of entries in the cache.
	Size() int
}

// heapEntry represents an entry in the expiration min-heap.
type heapEntry struct {
	key       CacheKey
	expiresAt time.Time
	index     int // Index in the heap (maintained by container/heap)
}

// expirationHeap implements heap.Interface for managing cache entries by expiration time.
type expirationHeap []*heapEntry

func (h expirationHeap) Len() int { return len(h) }

func (h expirationHeap) Less(i, j int) bool {
	return h[i].expiresAt.Before(h[j].expiresAt)
}

func (h expirationHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *expirationHeap) Push(x interface{}) {
	n := len(*h)
	entry := x.(*heapEntry)
	entry.index = n
	*h = append(*h, entry)
}

func (h *expirationHeap) Pop() interface{} {
	old := *h
	n := len(old)
	entry := old[n-1]
	old[n-1] = nil   // Avoid memory leak
	entry.index = -1 // Mark as removed
	*h = old[0 : n-1]
	return entry
}

// metricCache implements MetricCache with a dual-index data structure.
type metricCache struct {
	mu           sync.RWMutex
	data         map[CacheKey]MetricCacheEntry // Primary index: O(1) lookup
	expHeap      expirationHeap                // Expiration index: min-heap ordered by expiration time
	heapIndex    map[CacheKey]*heapEntry       // Maps cache keys to heap entries for O(log N) updates
	maxSize      int
	timeProvider TimeProvider
}

// NewMetricCache creates a new metric cache with the specified maximum size.
func NewMetricCache(maxSize int, timeProvider TimeProvider) MetricCache {
	if timeProvider == nil {
		timeProvider = &RealTimeProvider{}
	}

	c := &metricCache{
		data:         make(map[CacheKey]MetricCacheEntry),
		expHeap:      make(expirationHeap, 0),
		heapIndex:    make(map[CacheKey]*heapEntry),
		maxSize:      maxSize,
		timeProvider: timeProvider,
	}
	heap.Init(&c.expHeap)
	return c
}

// isExpired checks if a cache entry has expired based on the current time.
// An entry is expired if the current time is AFTER the expiration time (not at or after).
// This means if TTL is 60s, the entry is valid at second 60 and expires at second 61.
func (c *metricCache) isExpired(expiresAt time.Time) bool {
	now := c.timeProvider.Now()
	return now.After(expiresAt)
}

// Get retrieves a metric entry from the cache.
func (c *metricCache) Get(key CacheKey) (MetricCacheEntry, bool) {
	c.mu.RLock()
	entry, exists := c.data[key]
	c.mu.RUnlock()

	if !exists {
		return MetricCacheEntry{}, false
	}

	// Check if entry is expired
	if c.isExpired(entry.ExpiresAt) {
		// Entry is expired, remove it (lazy cleanup)
		c.mu.Lock()
		c.removeEntry(key)
		c.mu.Unlock()
		return MetricCacheEntry{}, false
	}

	return entry, true
}

// Peek retrieves a metric entry from the cache without removing expired entries.
// Returns the entry, whether it was found, and whether it's expired.
func (c *metricCache) Peek(key CacheKey) (MetricCacheEntry, bool, bool) {
	c.mu.RLock()
	entry, exists := c.data[key]
	c.mu.RUnlock()

	if !exists {
		return MetricCacheEntry{}, false, false
	}

	expired := c.isExpired(entry.ExpiresAt)
	return entry, true, expired
}

// Set stores a metric entry in the cache with the specified TTL.
// The expiration time is calculated based on the metric's timestamp (when it was measured),
// not the current time (when it was cached). This ensures cache expiration aligns with
// the actual age of the metric data.
func (c *metricCache) Set(key CacheKey, value float64, timestamp time.Time, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Calculate expiration based on metric timestamp, not current time
	expiresAt := timestamp.Add(ttl)

	// Lazy eviction: if at max size, evict expired entries first
	if len(c.data) >= c.maxSize {
		c.evictExpiredLocked(c.maxSize) // Try to evict all expired entries

		// If still at max size after evicting expired entries, evict oldest entry
		if len(c.data) >= c.maxSize && len(c.expHeap) > 0 {
			oldestEntry := heap.Pop(&c.expHeap).(*heapEntry)
			delete(c.data, oldestEntry.key)
			delete(c.heapIndex, oldestEntry.key)
		}
	}

	// If key already exists, remove old heap entry
	if oldHeapEntry, exists := c.heapIndex[key]; exists {
		heap.Remove(&c.expHeap, oldHeapEntry.index)
		delete(c.heapIndex, key)
	}

	// Insert/update entry in primary map
	c.data[key] = MetricCacheEntry{
		Value:     value,
		Timestamp: timestamp,
		ExpiresAt: expiresAt,
	}

	// Add entry to expiration heap
	heapEntry := &heapEntry{
		key:       key,
		expiresAt: expiresAt,
	}
	heap.Push(&c.expHeap, heapEntry)
	c.heapIndex[key] = heapEntry

	// Log cache entry update if debug mode is enabled
	if utils.IsDebugEnabled() {
		log.Printf("[DEBUG] Metric-Data Cache Entry Updated: key={instance=%s, metric=%s, stat=%s}, timestamp=%s, expires_at=%s",
			key.Instance, key.MetricName, key.Statistic, timestamp.Format(time.RFC3339), expiresAt.Format(time.RFC3339))
	}
}

// EvictExpired removes up to maxToEvict expired entries from the cache.
func (c *metricCache) EvictExpired(maxToEvict int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.evictExpiredLocked(maxToEvict)
}

// evictExpiredLocked is the internal implementation of EvictExpired.
// Caller must hold the write lock.
func (c *metricCache) evictExpiredLocked(maxToEvict int) int {
	evicted := 0

	for evicted < maxToEvict && len(c.expHeap) > 0 {
		// Peek at the earliest expiration
		earliest := c.expHeap[0]
		// Check if the earliest entry is expired
		if !c.isExpired(earliest.expiresAt) {
			// No more expired entries
			break
		}

		// Pop and remove the expired entry
		heapEntry := heap.Pop(&c.expHeap).(*heapEntry)
		delete(c.data, heapEntry.key)
		delete(c.heapIndex, heapEntry.key)
		evicted++
	}

	return evicted
}

// removeEntry removes an entry from both the primary map and the heap.
// Caller must hold the write lock.
func (c *metricCache) removeEntry(key CacheKey) {
	if heapEntry, exists := c.heapIndex[key]; exists {
		heap.Remove(&c.expHeap, heapEntry.index)
		delete(c.heapIndex, key)
	}
	delete(c.data, key)
}

// Size returns the current number of entries in the cache.
func (c *metricCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}
