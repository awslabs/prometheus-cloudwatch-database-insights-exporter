package metric

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	awsPI "github.com/aws/aws-sdk-go-v2/service/pi"
	"github.com/aws/aws-sdk-go-v2/service/pi/types"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/cache"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/clients/pi"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/processing/formatting"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/utils"
)

const (
	MaxRetries = 3
	BaseDelay  = time.Second
)

type MetricManager struct {
	piService        pi.PIService
	configuration    *models.ParsedConfig
	registry         *utils.PerEngineMetricRegistry
	metricDataCache  cache.MetricCache
	ttlPolicyManager cache.TTLPolicyManager
	region           string
}

// MetricManager handles Performance Insights metric collection and caching for database instances.
// It coordinates between metric discovery and data collection to provide comprehensive database performance monitoring with efficient AWS API usage.
func NewMetricManager(pi pi.PIService, config *models.ParsedConfig, region string) (*MetricManager, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration parameter cannot be nil")
	}

	// Initialize metric data cache
	metricDataCache := cache.NewMetricCache(
		config.Discovery.Metrics.DataCacheMaxSize,
		&cache.RealTimeProvider{},
	)

	// Convert parsed pattern TTLs to cache.PatternTTL format
	patternTTLs := make([]cache.PatternTTL, len(config.Discovery.Metrics.DataCachePatterns))
	for i, p := range config.Discovery.Metrics.DataCachePatterns {
		patternTTLs[i] = cache.PatternTTL{
			Pattern: p.Pattern,
			TTL:     p.TTL,
		}
	}

	// Initialize TTL policy manager
	ttlPolicyManager, err := cache.NewTTLPolicyManager(patternTTLs)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTL policy manager: %w", err)
	}

	return &MetricManager{
		piService:        pi,
		configuration:    config,
		registry:         utils.NewPerEngineMetricRegistry(),
		metricDataCache:  metricDataCache,
		ttlPolicyManager: ttlPolicyManager,
		region:           region,
	}, nil
}

// GetMetricBatches retrieves and batches the metrics for an instance without collecting data.
// This method is used by the queue-based worker pool to generate all metric batch requests upfront.
func (metricManager *MetricManager) GetMetricBatches(ctx context.Context, instance models.Instance) ([][]string, error) {
	metricsList, err := metricManager.getMetrics(ctx, instance.ResourceID, instance.Engine, instance.Metrics)
	if err != nil {
		return nil, err
	}

	return utils.BatchMetricNames(metricsList, utils.BatchSize), nil
}

// CollectMetricsForBatch collects metric data for a specific batch of metrics for an instance.
// This method is called by worker goroutines in the queue-based worker pool pattern.
func (metricManager *MetricManager) CollectMetricsForBatch(ctx context.Context, instance models.Instance, metricsBatch []string, ch chan<- prometheus.Metric) error {
	// Consult cache and filter out metrics with valid cache entries
	metricsToFetch, cachedMetrics := metricManager.filterCachedMetrics(instance, metricsBatch)

	// Return cached values for cache hits
	for _, metricDatum := range cachedMetrics {
		if err := formatting.ConvertToPrometheusMetric(ch, instance, metricDatum, metricManager.configuration.Export.Prometheus.MetricPrefix); err != nil {
			log.Printf("[METRIC MANAGER] Error converting cached metric data to prometheus metric: %v, error: %v", metricDatum, err)
			continue
		}
	}

	// Fetch fresh values for cache misses
	if len(metricsToFetch) > 0 {
		if utils.IsDebugEnabled() {
			log.Printf("[DEBUG] Metric-Data Cache Summary: instance=%s, expired_metrics=%d/%d, fetching from AWS",
				instance.Identifier, len(metricsToFetch), len(metricsBatch))
		}

		dimensionGroups := metricManager.dimensionGroupsFor(instance, metricsToFetch)

		metricDataResult, err := metricManager.getMetricData(ctx, instance.ResourceID, metricsToFetch, dimensionGroups)
		if err != nil {
			log.Printf("[METRIC MANAGER] Error getting metric data for these metrics: %v, error: %v", metricsToFetch, err)
			return err
		}

		// A grouped metric answers with the aggregate plus one entry per
		// dimension, so remember which sets came back. Without that, a later
		// cache hit could only replay the aggregate.
		returnedDimensionSets := make(map[string][]map[string]string)

		// Grouped entries are keyed by their dimensions projected onto the
		// configured keys, which is also what the labels are built from. Two
		// entries can share that projection, for instance when they differ only
		// in a dimension that is not a configured key, or when Performance
		// Insights truncated two long statements to the same value. They are
		// merged here, since exporting both would put two series with identical
		// labels into one scrape and fail the whole /metrics response.
		var collected []collectedSeries
		seriesIndex := make(map[string]int)

		// Process each metric from the response
		for _, metricKeyDataPoints := range metricDataResult.MetricList {
			if metricKeyDataPoints.Key == nil || metricKeyDataPoints.Key.Metric == nil {
				continue
			}

			metricName := *metricKeyDataPoints.Key.Metric
			dimensions := metricKeyDataPoints.Key.Dimensions
			group := dimensionGroups[metricName]
			if group != nil && len(dimensions) > 0 {
				dimensions = models.ProjectDimensions(dimensions, group.Keys)
			}

			// Get the latest 2 valid datapoints (for both latest value and dynamic TTL calculation)
			// This is more efficient than iterating twice
			validDataPoints := metricManager.getLatestNValidDataPoints(metricKeyDataPoints.DataPoints, 2)
			if len(validDataPoints) == 0 {
				continue
			}

			metricDatum := models.MetricData{
				Metric:     metricName,
				Timestamp:  *validDataPoints[0].Timestamp,
				Value:      *validDataPoints[0].Value,
				Dimensions: dimensions,
			}
			if group != nil && len(dimensions) > 0 {
				metricDatum.DimensionGroup = group.Group
				metricDatum.DimensionKeys = group.Keys
			}

			// Calculate dynamic TTL from the valid datapoints (requires at least 2)
			dynamicTTL := metricManager.calculateDynamicTTL(validDataPoints)

			seriesKey := metricName + "\x00" + metricDatum.DimensionFingerprint()
			if position, seen := seriesIndex[seriesKey]; seen {
				metricManager.mergeSeries(instance, &collected[position], metricDatum, dynamicTTL)
				continue
			}
			seriesIndex[seriesKey] = len(collected)
			collected = append(collected, collectedSeries{datum: metricDatum, ttl: dynamicTTL})
		}

		for _, series := range collected {
			metricDatum := series.datum
			if dimensionGroups[metricDatum.Metric] != nil {
				returnedDimensionSets[metricDatum.Metric] = append(returnedDimensionSets[metricDatum.Metric], metricDatum.Dimensions)
			}

			// Update cache with fetched value and dynamic TTL
			metricManager.updateCacheWithDynamicTTL(instance, metricDatum, series.ttl)

			if err := formatting.ConvertToPrometheusMetric(ch, instance, metricDatum, metricManager.configuration.Export.Prometheus.MetricPrefix); err != nil {
				log.Printf("[METRIC MANAGER] Error converting metric data to prometheus metric: %v, error: %v", metricDatum, err)
				continue
			}
		}

		if instance.Metrics != nil {
			for metricName, dimensionSets := range returnedDimensionSets {
				instance.Metrics.DimensionSeries.Record(metricName, dimensionSets)
			}
		}

		if utils.IsDebugEnabled() {
			log.Printf("[DEBUG] Metric-Data Cache Updated: instance=%s, new_entries=%d",
				instance.Identifier, len(metricDataResult.MetricList))
		}
	}

	return nil
}

func (metricManager *MetricManager) getMetrics(ctx context.Context, resourceID string, engine models.Engine, metrics *models.Metrics) ([]string, error) {
	if metrics == nil {
		return nil, fmt.Errorf("[METRIC MANAGER] Metrics not found for instance: %s", resourceID)
	}

	if metrics.MetricsDetails == nil || metrics.MetricsLastUpdated.IsZero() || time.Now().After(metrics.MetricsLastUpdated.Add(metrics.MetadataTTL)) {
		if utils.IsDebugEnabled() {
			log.Printf("[DEBUG] Metric-Metadata Cache Expired for instance: %s, fetching new metric definitions from AWS", resourceID)
		}

		availableMetrics, err := metricManager.getAvailableMetrics(ctx, resourceID, engine)
		if err != nil {
			return nil, err
		}

		filteredMetrics := make(map[string]models.MetricDetails)
		metricConfig := metricManager.configuration.Discovery.Metrics
		for metricName, metric := range availableMetrics {
			if metricConfig.ShouldIncludeMetric(metric) {
				filteredMetrics[metricName] = metric
			}
		}

		filteredMetricList := utils.GetMetricNamesWithStatistic(filteredMetrics)

		if utils.IsDebugEnabled() {
			log.Printf("[DEBUG] Metric-Metadata Cache Updated for instance: %s, cached %d metrics", resourceID, len(filteredMetrics))
		}

		metrics.MetricsDetails = filteredMetrics
		metrics.MetricsList = filteredMetricList
		metrics.MetricsLastUpdated = time.Now()
	}

	if len(metricManager.configuration.Discovery.Metrics.DimensionGroups) > 0 &&
		(metrics.DimensionsLastUpdated.IsZero() || time.Now().After(metrics.DimensionsLastUpdated.Add(metrics.MetadataTTL))) {
		metricManager.refreshAvailableDimensions(ctx, resourceID, metrics)
	}

	return metrics.MetricsList, nil
}

// refreshAvailableDimensions looks up which dimension groups the instance
// supports for every metric type a dimension rule applies to. It runs at most
// once per metadata TTL. A failed lookup is not fatal: the affected metrics are
// queried ungrouped until the next refresh, which is safer than sending a GroupBy
// the engine may reject along with every metric in its request.
func (metricManager *MetricManager) refreshAvailableDimensions(ctx context.Context, resourceID string, metrics *models.Metrics) {
	metricConfig := &metricManager.configuration.Discovery.Metrics

	metricTypes := make(map[string]struct{})
	for _, metricNameWithStat := range metrics.MetricsList {
		if metricConfig.DimensionGroupFor(metricNameWithStat) != nil {
			metricTypes[utils.TrimStatisticFromMetricName(metricNameWithStat)] = struct{}{}
		}
	}

	availableDimensions := make(map[string]models.DimensionGroupAvailability, len(metricTypes))
	for metricType := range metricTypes {
		availability, err := utils.WithRetry(ctx, func() (models.DimensionGroupAvailability, error) {
			return metricManager.piService.ListAvailableResourceDimensions(ctx, resourceID, metricType)
		}, MaxRetries, BaseDelay)
		if err != nil {
			log.Printf("[METRIC MANAGER] Error listing available dimensions for instance: %s, metric: %s, querying it ungrouped: %v", resourceID, metricType, err)
			continue
		}
		availableDimensions[metricType] = availability

		reported := make(map[*models.ParsedDimensionGroup]struct{})
		for _, metricNameWithStat := range metrics.MetricsList {
			if utils.TrimStatisticFromMetricName(metricNameWithStat) != metricType {
				continue
			}
			group := metricConfig.DimensionGroupFor(metricNameWithStat)
			if group == nil || availability.Supports(group.Group, group.Keys) {
				continue
			}
			if _, done := reported[group]; done {
				continue
			}
			reported[group] = struct{}{}
			log.Printf("[METRIC MANAGER] Dimension group %s with keys %v is not available for instance: %s, metric: %s, querying it ungrouped", group.Group, group.Keys, resourceID, metricType)
		}
	}

	metrics.AvailableDimensions = availableDimensions
	metrics.DimensionsLastUpdated = time.Now()
}

func (metricManager *MetricManager) getAvailableMetrics(ctx context.Context, resourceID string, engine models.Engine) (map[string]models.MetricDetails, error) {
	availableMetrics, err := utils.WithRetry(ctx, func() (*awsPI.ListAvailableResourceMetricsOutput, error) {
		return metricManager.piService.ListAvailableResourceMetrics(ctx, resourceID)
	}, MaxRetries, BaseDelay)
	if err != nil {
		return nil, err
	}

	return utils.BuildMetricDefinitionMap(availableMetrics.Metrics, &metricManager.configuration.Discovery.Metrics, engine, metricManager.registry)
}

func (metricManager *MetricManager) getMetricData(ctx context.Context, resourceID string, metricNamesWithStat []string, dimensionGroups map[string]*models.ParsedDimensionGroup) (*awsPI.GetResourceMetricsOutput, error) {
	metricDataResult, err := utils.WithRetry(ctx, func() (*awsPI.GetResourceMetricsOutput, error) {
		return metricManager.piService.GetResourceMetrics(ctx, resourceID, metricNamesWithStat, dimensionGroups)
	}, MaxRetries, BaseDelay)
	if err != nil {
		return nil, err
	}

	return metricDataResult, nil
}

func (metricManager *MetricManager) filterLatestValidMetricData(result *awsPI.GetResourceMetricsOutput) []models.MetricData {
	var filteredData []models.MetricData

	for _, metricData := range result.MetricList {
		if metricData.Key == nil || metricData.Key.Metric == nil {
			continue
		}

		latestDataPoint := metricManager.getLatestValidDataPoint(metricData.DataPoints)
		if latestDataPoint != nil && latestDataPoint.Value != nil && latestDataPoint.Timestamp != nil {
			filteredData = append(filteredData, models.MetricData{
				Metric:    *metricData.Key.Metric,
				Timestamp: *latestDataPoint.Timestamp,
				Value:     *latestDataPoint.Value,
			})
		}
	}

	return filteredData
}

// getLatestNValidDataPoints extracts the N most recent valid datapoints from the array.
// Returns datapoints in reverse chronological order (most recent first).
// This is more efficient than multiple iterations when we need both the latest point
// and multiple points for TTL calculation.
func (metricManager *MetricManager) getLatestNValidDataPoints(dataPoints []types.DataPoint, n int) []types.DataPoint {
	if len(dataPoints) == 0 || n <= 0 {
		return nil
	}

	validPoints := make([]types.DataPoint, 0, n)
	
	// Iterate from most recent to oldest
	for i := len(dataPoints) - 1; i >= 0; i-- {
		dp := dataPoints[i]
		if dp.Value != nil && dp.Timestamp != nil {
			validPoints = append(validPoints, dp)
			// Return early once we have N valid datapoints
			if len(validPoints) == n {
				return validPoints
			}
		}
	}

	return validPoints
}

// getLatestValidDataPoint returns the most recent valid datapoint.
// This is a convenience wrapper around getLatestNValidDataPoints for backward compatibility.
func (metricManager *MetricManager) getLatestValidDataPoint(dataPoints []types.DataPoint) *types.DataPoint {
	validPoints := metricManager.getLatestNValidDataPoints(dataPoints, 1)
	if len(validPoints) == 0 {
		return nil
	}
	return &validPoints[0]
}

// filterCachedMetrics consults the cache and separates metrics into those that need fetching
// and those that can be served from cache.
func (metricManager *MetricManager) filterCachedMetrics(instance models.Instance, metricsBatch []string) ([]string, []models.MetricData) {
	var metricsToFetch []string
	var cachedMetrics []models.MetricData

	for _, metricNameWithStat := range metricsBatch {
		// Extract statistic from metric name
		statistic := metricManager.extractStatistic(metricNameWithStat)

		// A grouped metric is cached as several series, one per dimension set the
		// last response carried. It can only be served from cache when every one
		// of them is still fresh, otherwise the metric is refetched as a whole.
		if metricManager.dimensionGroupFor(instance, metricNameWithStat) != nil {
			replayed, complete := metricManager.cachedDimensionSeries(instance, metricNameWithStat, statistic)
			if complete {
				cachedMetrics = append(cachedMetrics, replayed...)
			} else {
				metricsToFetch = append(metricsToFetch, metricNameWithStat)
			}
			continue
		}

		// Build cache key (region not needed since MetricManager is region-scoped)
		cacheKey := cache.CacheKey{
			Instance:   instance.Identifier,
			MetricName: metricNameWithStat,
			Statistic:  statistic,
		}

		// Check cache
		if entry, found := metricManager.metricDataCache.Get(cacheKey); found {
			// Cache hit - use cached value
			cachedMetrics = append(cachedMetrics, models.MetricData{
				Metric:    metricNameWithStat,
				Timestamp: entry.Timestamp,
				Value:     entry.Value,
			})
		} else {
			// Cache miss - need to fetch
			metricsToFetch = append(metricsToFetch, metricNameWithStat)

			// Log individual expired/missing metric if debug mode is enabled
			if utils.IsDebugEnabled() {
				// Peek at the cache to get expired entry details for logging
				if expiredEntry, exists, isExpired := metricManager.metricDataCache.Peek(cacheKey); exists && isExpired {
					// Entry exists but is expired - log with timestamps
					log.Printf("[DEBUG] Metric-Data Cache Expired: instance=%s, metric=%s, stat=%s, timestamp=%s, expires_at=%s",
						instance.Identifier, metricNameWithStat, statistic,
						expiredEntry.Timestamp.Format(time.RFC3339), expiredEntry.ExpiresAt.Format(time.RFC3339))
				} else {
					// Entry doesn't exist - log without timestamps
					log.Printf("[DEBUG] Metric-Data Cache Expired: instance=%s, metric=%s, stat=%s",
						instance.Identifier, metricNameWithStat, statistic)
				}
			}
		}
	}

	return metricsToFetch, cachedMetrics
}

// collectedSeries is one series of a response, held back until every entry of
// the response has been seen so that entries with the same labels can be merged.
type collectedSeries struct {
	datum models.MetricData
	ttl   time.Duration
}

// mergeSeries folds a response entry into the series already collected under the
// same labels.
//
// Load is additive across dimensions, so averages and sums add up. The max or
// min of a sum cannot be derived from the max or min of its parts, but load is
// never negative, so the sum is at least as large as either part and the larger
// value is the closest lower bound available.
func (metricManager *MetricManager) mergeSeries(instance models.Instance, series *collectedSeries, metricDatum models.MetricData, ttl time.Duration) {
	switch models.Statistic(metricManager.extractStatistic(metricDatum.Metric)) {
	case models.StatisticAvg, models.StatisticSum:
		series.datum.Value += metricDatum.Value
	default:
		series.datum.Value = math.Max(series.datum.Value, metricDatum.Value)
	}

	if metricDatum.Timestamp.After(series.datum.Timestamp) {
		series.datum.Timestamp = metricDatum.Timestamp
	}
	if ttl > 0 && (series.ttl == 0 || ttl < series.ttl) {
		series.ttl = ttl
	}

	if utils.IsDebugEnabled() {
		log.Printf("[DEBUG] Merged response entries with identical labels: instance=%s, metric=%s, dimensions=%s",
			instance.Identifier, metricDatum.Metric, metricDatum.DimensionFingerprint())
	}
}

// dimensionGroupFor returns the dimension group a metric is broken down by on
// this instance, or nil when no rule matches or the instance does not support
// the matching rule's group and keys.
func (metricManager *MetricManager) dimensionGroupFor(instance models.Instance, metricNameWithStat string) *models.ParsedDimensionGroup {
	group := metricManager.configuration.Discovery.Metrics.DimensionGroupFor(metricNameWithStat)
	if group == nil {
		return nil
	}
	if !instance.Metrics.SupportsDimensionGroup(utils.TrimStatisticFromMetricName(metricNameWithStat), group.Group, group.Keys) {
		return nil
	}
	return group
}

// dimensionGroupsFor maps each metric that is broken down on this instance to
// its dimension group. Metrics absent from the result are queried ungrouped.
func (metricManager *MetricManager) dimensionGroupsFor(instance models.Instance, metricNamesWithStat []string) map[string]*models.ParsedDimensionGroup {
	var dimensionGroups map[string]*models.ParsedDimensionGroup

	for _, metricNameWithStat := range metricNamesWithStat {
		group := metricManager.dimensionGroupFor(instance, metricNameWithStat)
		if group == nil {
			continue
		}
		if dimensionGroups == nil {
			dimensionGroups = make(map[string]*models.ParsedDimensionGroup)
		}
		dimensionGroups[metricNameWithStat] = group
	}

	return dimensionGroups
}

// cachedDimensionSeries rebuilds every series a grouped metric produced from the
// cache. It reports false as soon as one series is missing or expired, because a
// partial breakdown would understate the metric rather than simply go stale.
func (metricManager *MetricManager) cachedDimensionSeries(instance models.Instance, metricNameWithStat string, statistic string) ([]models.MetricData, bool) {
	if instance.Metrics == nil {
		return nil, false
	}

	dimensionSets, known := instance.Metrics.DimensionSeries.Get(metricNameWithStat)
	if !known || len(dimensionSets) == 0 {
		return nil, false
	}

	group := metricManager.dimensionGroupFor(instance, metricNameWithStat)
	replayed := make([]models.MetricData, 0, len(dimensionSets))

	for _, dimensions := range dimensionSets {
		cacheKey := cache.CacheKey{
			Instance:   instance.Identifier,
			MetricName: metricNameWithStat,
			Statistic:  statistic,
			Dimensions: models.FingerprintDimensions(dimensions),
		}

		entry, found := metricManager.metricDataCache.Get(cacheKey)
		if !found {
			return nil, false
		}

		metricDatum := models.MetricData{
			Metric:     metricNameWithStat,
			Timestamp:  entry.Timestamp,
			Value:      entry.Value,
			Dimensions: dimensions,
		}
		if group != nil && len(dimensions) > 0 {
			metricDatum.DimensionGroup = group.Group
			metricDatum.DimensionKeys = group.Keys
		}
		replayed = append(replayed, metricDatum)
	}

	return replayed, true
}

// updateCacheWithDynamicTTL stores a fetched metric value in the cache with pattern-based or dynamic TTL.
// If a pattern matches and its TTL is larger than the dynamic TTL, uses the pattern's TTL.
// Otherwise, uses the dynamic TTL (which adapts to the actual metric granularity).
// If dynamic TTL is 0 (couldn't be calculated), skips caching.
func (metricManager *MetricManager) updateCacheWithDynamicTTL(instance models.Instance, metricDatum models.MetricData, dynamicTTL time.Duration) {
	// Extract statistic from metric name
	statistic := metricManager.extractStatistic(metricDatum.Metric)

	// Get TTL from policy manager using just the metric name (returns 0 if no pattern matches)
	patternTTL := metricManager.ttlPolicyManager.GetTTL(metricDatum.Metric)
	
	// Choose the appropriate TTL:
	// 1. If no pattern matched (patternTTL == 0), use dynamic TTL
	// 2. If pattern matched but dynamic TTL is larger, use dynamic TTL (respects actual metric granularity)
	// 3. Otherwise, use pattern TTL
	var ttl time.Duration
	var ttlSource string
	
	if patternTTL == 0 {
		// No pattern match - use dynamic TTL
		ttl = dynamicTTL
		ttlSource = "dynamic"
	} else if dynamicTTL > patternTTL {
		// Pattern matched but dynamic TTL is larger - use dynamic to respect metric granularity
		ttl = dynamicTTL
		ttlSource = "dynamic (overriding pattern)"
	} else {
		// Pattern matched and is >= dynamic TTL - use pattern
		ttl = patternTTL
		ttlSource = "pattern"
	}
	
	// If TTL is still 0 (no pattern and couldn't calculate dynamic), skip caching
	if ttl == 0 {
		if utils.IsDebugEnabled() {
			log.Printf("[DEBUG] Skipping cache for metric %s: no pattern match and insufficient datapoints for dynamic TTL",
				metricDatum.Metric)
		}
		return
	}

	// Build cache key (region not needed since MetricManager is region-scoped)
	cacheKey := cache.CacheKey{
		Instance:   instance.Identifier,
		MetricName: metricDatum.Metric,
		Statistic:  statistic,
		Dimensions: metricDatum.DimensionFingerprint(),
	}

	// Store in cache
	metricManager.metricDataCache.Set(cacheKey, metricDatum.Value, metricDatum.Timestamp, ttl)
	
	if utils.IsDebugEnabled() {
		log.Printf("[DEBUG] Metric cached: instance=%s, metric=%s, ttl=%s (source=%s)",
			instance.Identifier, metricDatum.Metric, ttl, ttlSource)
	}
}

// extractStatistic extracts the statistic suffix from a metric name.
// For example, "db.load.avg" returns "avg".
func (metricManager *MetricManager) extractStatistic(metricNameWithStat string) string {
	for _, stat := range models.GetAllStatistics() {
		suffix := "." + stat.String()
		if len(metricNameWithStat) > len(suffix) && metricNameWithStat[len(metricNameWithStat)-len(suffix):] == suffix {
			return stat.String()
		}
	}
	return ""
}

// calculateDynamicTTL calculates TTL based on the time difference between the last two valid datapoints.
// This allows the cache to adapt to the actual metric granularity automatically.
// Takes pre-extracted valid datapoints (most recent first) to avoid redundant iteration.
// Returns the calculated TTL, or 0 if unable to calculate (less than 2 datapoints).
func (metricManager *MetricManager) calculateDynamicTTL(validDataPoints []types.DataPoint) time.Duration {
	// Need at least 2 valid datapoints to calculate TTL
	if len(validDataPoints) < 2 {
		return 0
	}
	
	// validDataPoints[0] is the most recent, validDataPoints[1] is the second most recent
	timeDiff := validDataPoints[0].Timestamp.Sub(*validDataPoints[1].Timestamp)
	
	// Ensure TTL is positive
	if timeDiff <= 0 {
		return 0
	}
	
	return timeDiff
}
