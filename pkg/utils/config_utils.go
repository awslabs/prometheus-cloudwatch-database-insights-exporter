package utils

import (
	"cmp"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/filter"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"

	"gopkg.in/yaml.v2"
)

const (
	MaxInstances          = 25
	MaxInstancesHardLimit = 10_000
	BatchSize             = 15
	MaximumConcurrency    = 60
	DefaultConcurrency    = 4
	MinTTL                = time.Minute
	MaxTTL                = time.Hour * 24
	DefaultInstanceTTL    = time.Minute * 5
	DefaultMetadataTTL    = time.Minute * 60
	DefaultCacheMaxSize   = 100000
	MinCacheMaxSize       = 1
	ValidPrometheusName   = `^[a-zA-Z_:][a-zA-Z0-9_:]*$`
)

// validRoleARNPattern accepts exactly the three real AWS partitions: standard, China (aws-cn),
// and GovCloud (aws-us-gov). The role name/path segment uses IAM-allowed characters [\w+=,.@/-].
// The {1,1000} upper bound is deliberately generous; the empty-name guard below catches trailing slashes.
var validRoleARNPattern = regexp.MustCompile(`^arn:(aws|aws-cn|aws-us-gov):iam::\d{12}:role/[\w+=,.@/\-]{1,1000}$`)

// validExternalIDPattern matches the character class of the AWS STS ExternalId constraint: [\w+=,.@:/-].
// Length (2-1224) is checked separately because Go's RE2 engine caps repeat counts at 1000.
var validExternalIDPattern = regexp.MustCompile(`^[\w+=,.@:/\-]+$`)

func LoadConfig(filePath string) (*models.ParsedConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			config := createDefaultConfig()
			applyDefaults(&config)
			return parsedValidateConfig(&config)
		}
		return nil, err
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	applyDefaults(&config)

	return parsedValidateConfig(&config)
}

func createDefaultConfig() models.Config {
	return models.Config{
		Discovery: models.DiscoveryConfig{
			Regions: []string{},
			Instances: models.InstancesConfig{
				MaxInstances: 0,
				Cache: models.InstancesCacheConfig{
					TTL: "",
				},
			},
			Metrics: models.MetricsConfig{
				Statistic: "",
				Cache: models.MetricsCacheConfig{
					MetricMetadataTTL: "",
					MetricData: models.MetricDataCacheConfig{
						MaxSize: 0,
					},
				},
			},
			Processing: models.ProcessingConfig{
				Concurrency: 0,
			},
		},
		Export: models.ExportConfig{
			Port: 0,
			Prometheus: models.PrometheusConfig{
				MetricPrefix: "",
			},
		},
	}
}

func applyDefaults(config *models.Config) {
	if len(config.Discovery.Regions) == 0 {
		config.Discovery.Regions = []string{"us-west-2"}
	}

	if config.Discovery.Instances.MaxInstances <= 0 {
		config.Discovery.Instances.MaxInstances = MaxInstances
	}

	if config.Discovery.Metrics.Statistic == "" {
		config.Discovery.Metrics.Statistic = "avg"
	}

	if config.Discovery.Processing.Concurrency <= 0 {
		config.Discovery.Processing.Concurrency = DefaultConcurrency
	}

	if config.Export.Port == 0 {
		config.Export.Port = 8081
	}

	if config.Export.Prometheus.MetricPrefix == "" {
		config.Export.Prometheus.MetricPrefix = "dbi"
	}

	// Apply cache defaults
	if config.Discovery.Instances.Cache.TTL == "" {
		config.Discovery.Instances.Cache.TTL = "5m"
	}
	if config.Discovery.Metrics.Cache.MetricMetadataTTL == "" {
		config.Discovery.Metrics.Cache.MetricMetadataTTL = "60m"
	}
	if config.Discovery.Metrics.Cache.MetricData.MaxSize <= 0 {
		config.Discovery.Metrics.Cache.MetricData.MaxSize = DefaultCacheMaxSize
	}

}

func parsedValidateConfig(config *models.Config) (*models.ParsedConfig, error) {
	var parsedConfig models.ParsedConfig

	if len(config.Discovery.Regions) > 1 {
		// Current version only supports single region exporter
		parsedConfig.Discovery.Regions = []string{config.Discovery.Regions[0]}
	} else {
		parsedConfig.Discovery.Regions = config.Discovery.Regions
	}

	instancesConfig, err := parseInstancesConfig(config.Discovery.Instances)
	if err != nil {
		return nil, err
	}
	parsedConfig.Discovery.Instances = instancesConfig

	metricsConfig, err := parsedMetricsConfig(config.Discovery.Metrics)
	if err != nil {
		return nil, err
	}
	parsedConfig.Discovery.Metrics = metricsConfig

	parsedConfig.Discovery.Processing = parseProcessingConfig(config.Discovery.Processing)

	if roleARN := config.Discovery.RoleARN; roleARN != "" {
		if !validRoleARNPattern.MatchString(roleARN) {
			return nil, fmt.Errorf("invalid discovery.role_arn %q: expected format arn:aws*:iam::<12-digit-account>:role/<name>", roleARN)
		}
		// SplitN on the structural ":role/" prefix. IAM role names themselves cannot
		// contain "/", so the last slash-delimited segment is always the role name.
		rolePathAndName := strings.SplitN(roleARN, ":role/", 2)[1]
		roleName := rolePathAndName
		if i := strings.LastIndex(rolePathAndName, "/"); i >= 0 {
			roleName = rolePathAndName[i+1:]
		}
		if roleName == "" {
			return nil, fmt.Errorf("invalid discovery.role_arn %q: role name segment must not be empty", roleARN)
		}
		if utf8.RuneCountInString(roleName) > 64 {
			return nil, fmt.Errorf("invalid discovery.role_arn: role name %q exceeds the AWS 64-character limit", roleName)
		}
		parsedConfig.Discovery.RoleARN = roleARN
		if extID := config.Discovery.RoleARNExternalID; extID != "" {
			if l := utf8.RuneCountInString(extID); l < 2 || l > 1224 {
				return nil, fmt.Errorf("invalid discovery.role_arn_external_id: length must be 2-1224 characters (got %d)", l)
			}
			if !validExternalIDPattern.MatchString(extID) {
				return nil, fmt.Errorf("invalid discovery.role_arn_external_id: must contain only [a-zA-Z0-9_+=,.@:/-]")
			}
			parsedConfig.Discovery.RoleARNExternalID = extID
		}
	} else if config.Discovery.RoleARNExternalID != "" {
		return nil, fmt.Errorf("discovery.role_arn_external_id requires discovery.role_arn to be set")
	}

	exportConfig, err := parseExportConfig(config.Export)
	if err != nil {
		return nil, err
	}
	parsedConfig.Export = exportConfig

	return &parsedConfig, nil
}

func getAllValidFilterFields() map[string]bool {
	validFields := make(map[string]bool)

	instance := models.Instance{}
	for fieldName := range instance.GetFilterableFields() {
		validFields[fieldName] = true
	}

	metric := models.MetricDetails{}
	for fieldName := range metric.GetFilterableFields() {
		validFields[fieldName] = true
	}

	return validFields
}

func isValidFilterField(fieldName string) bool {
	if strings.HasPrefix(fieldName, models.FilterTypeTagPrefix.String()) {
		return len(fieldName) > len(models.FilterTypeTagPrefix)
	}

	validFields := getAllValidFilterFields()
	return validFields[fieldName]
}

func compileFilterConfig(config models.FilterConfig) (filter.Patterns, error) {
	if config == nil {
		return nil, nil
	}

	filter := filter.Patterns{}
	for fieldName, patterns := range config {
		if !isValidFilterField(fieldName) {
			return nil, fmt.Errorf("invalid filter field '%s' in config.yml", fieldName)
		}

		compiledPatterns, err := compileRegexPatterns(patterns)
		if err != nil {
			return nil, fmt.Errorf("invalid filter patterns in config.yml: %v", err)
		}

		filter[fieldName] = compiledPatterns
	}

	return filter, nil
}

func parseInstancesConfig(config models.InstancesConfig) (models.ParsedInstancesConfig, error) {
	maxInstances := config.MaxInstances
	if maxInstances <= 0 {
		log.Printf("[CONFIG] max-instances %d is outside the allowed range [1, %d], setting to %d", maxInstances, MaxInstancesHardLimit, MaxInstances)
		maxInstances = MaxInstances
	} else if maxInstances > MaxInstancesHardLimit {
		log.Printf("[CONFIG] max-instances %d is outside the allowed range [1, %d], setting to %d", maxInstances, MaxInstancesHardLimit, MaxInstancesHardLimit)
		maxInstances = MaxInstancesHardLimit
	}

	// Parse instance discovery cache TTL
	cacheTTL := DefaultInstanceTTL
	if config.Cache.TTL != "" {
		ttl, err := time.ParseDuration(config.Cache.TTL)
		if err != nil {
			return models.ParsedInstancesConfig{}, fmt.Errorf("invalid discovery.instances.cache.ttl format '%s' in config.yml: %v", config.Cache.TTL, err)
		}
		if ttl < 0 {
			return models.ParsedInstancesConfig{}, fmt.Errorf("invalid discovery.instances.cache.ttl in config.yml: TTL must be positive, got %s", config.Cache.TTL)
		}
		cacheTTL = GetOrDefault(ttl, MinTTL, MaxTTL, DefaultInstanceTTL, "discovery.instances.cache.ttl")
	}

	includePatterns, err := compileFilterConfig(config.Include)
	if err != nil {
		return models.ParsedInstancesConfig{}, fmt.Errorf("invalid instance.include patterns in config.yml: %v", err)
	}

	excludePatterns, err := compileFilterConfig(config.Exclude)
	if err != nil {
		return models.ParsedInstancesConfig{}, fmt.Errorf("invalid instance.exclude patterns in config.yml: %v", err)
	}

	var instanceFilter filter.Filter
	if len(includePatterns) > 0 || len(excludePatterns) > 0 {
		instanceFilter = filter.NewPatternFilter(includePatterns, excludePatterns)
	}

	return models.ParsedInstancesConfig{
		MaxInstances: maxInstances,
		CacheTTL:     cacheTTL,
		Filter:       instanceFilter,
	}, nil
}

func extractMetricAndStatistic(pattern string) (string, string) {
	for _, statistic := range models.GetAllStatistics() {
		suffix := "." + statistic.String()
		if strings.HasSuffix(strings.ToLower(pattern), suffix) {
			metricName := strings.TrimSuffix(pattern, suffix)
			return metricName, statistic.String()
		}
	}
	return "", ""
}

func parsedMetricsConfig(config models.MetricsConfig) (models.ParsedMetricsConfig, error) {
	defaultStatistic := models.NewStatistic(config.Statistic)
	if defaultStatistic == "" {
		return models.ParsedMetricsConfig{}, fmt.Errorf("invalid statistic %s provided in config.yml", config.Statistic)
	}

	// Parse metric metadata cache TTL
	metadataCacheTTL := DefaultMetadataTTL
	if config.Cache.MetricMetadataTTL != "" {
		ttl, err := time.ParseDuration(config.Cache.MetricMetadataTTL)
		if err != nil {
			return models.ParsedMetricsConfig{}, fmt.Errorf("invalid discovery.metrics.cache.metric-metadata-ttl format '%s' in config.yml: %v", config.Cache.MetricMetadataTTL, err)
		}
		if ttl < 0 {
			return models.ParsedMetricsConfig{}, fmt.Errorf("invalid discovery.metrics.cache.metric-metadata-ttl in config.yml: TTL must be positive, got %s", config.Cache.MetricMetadataTTL)
		}
		metadataCacheTTL = GetOrDefault(ttl, MinTTL, MaxTTL, DefaultMetadataTTL, "discovery.metrics.cache.metric-metadata-ttl")
	}

	// Parse metric data cache configuration
	dataCacheMaxSize := config.Cache.MetricData.MaxSize
	if dataCacheMaxSize < MinCacheMaxSize {
		return models.ParsedMetricsConfig{}, fmt.Errorf("invalid discovery.metrics.cache.metric-data.max-size in config.yml: max size must be >= %d, got %d", MinCacheMaxSize, dataCacheMaxSize)
	}

	// Parse and validate pattern TTLs
	var parsedPatternTTLs []models.ParsedPatternTTL
	for i, patternTTL := range config.Cache.MetricData.PatternTTLs {
		// Validate pattern regex
		compiledPattern, err := regexp.Compile(patternTTL.Pattern)
		if err != nil {
			return models.ParsedMetricsConfig{}, fmt.Errorf("invalid discovery.metrics.cache.metric-data.pattern-ttls[%d].pattern in config.yml: %v", i, err)
		}

		// Validate and parse TTL
		if patternTTL.TTL == "" {
			return models.ParsedMetricsConfig{}, fmt.Errorf("invalid discovery.metrics.cache.metric-data.pattern-ttls[%d].ttl in config.yml: TTL cannot be empty", i)
		}

		ttl, err := time.ParseDuration(patternTTL.TTL)
		if err != nil {
			return models.ParsedMetricsConfig{}, fmt.Errorf("invalid discovery.metrics.cache.metric-data.pattern-ttls[%d].ttl format '%s' in config.yml: %v", i, patternTTL.TTL, err)
		}

		// Reject negative TTLs
		if ttl < 0 {
			return models.ParsedMetricsConfig{}, fmt.Errorf("invalid discovery.metrics.cache.metric-data.pattern-ttls[%d].ttl in config.yml: TTL must be positive, got %s", i, patternTTL.TTL)
		}

		parsedPatternTTLs = append(parsedPatternTTLs, models.ParsedPatternTTL{
			Pattern: compiledPattern.String(),
			TTL:     ttl,
		})
	}

	includePatterns, err := compileFilterConfig(config.Include)
	if err != nil {
		return models.ParsedMetricsConfig{}, fmt.Errorf("invalid metrics.include patterns in config.yml: %v", err)
	}

	excludePatterns, err := compileFilterConfig(config.Exclude)
	if err != nil {
		return models.ParsedMetricsConfig{}, fmt.Errorf("invalid metrics.exclude patterns in config.yml: %v", err)
	}

	var metricFilter filter.Filter
	if len(includePatterns) > 0 || len(excludePatterns) > 0 {
		metricFilter = filter.NewPatternFilter(includePatterns, excludePatterns)
	}

	return models.ParsedMetricsConfig{
		Statistic:         defaultStatistic,
		MetadataCacheTTL:  metadataCacheTTL,
		DataCacheMaxSize:  dataCacheMaxSize,
		DataCachePatterns: parsedPatternTTLs,
		Filter:            metricFilter,
		Include:           config.Include,
		Exclude:           config.Exclude,
	}, nil
}

func parseProcessingConfig(config models.ProcessingConfig) models.ParsedProcessingConfig {
	concurrency := GetOrDefault(config.Concurrency, 1, MaximumConcurrency, MaximumConcurrency, "concurrency")

	return models.ParsedProcessingConfig{
		Concurrency: concurrency,
	}
}

func parseExportConfig(config models.ExportConfig) (models.ParsedExportConfig, error) {
	port := config.Port
	if port <= 0 || port > 65535 {
		port = 8081
	}

	if !isPortAvailable(port) {
		return models.ParsedExportConfig{}, fmt.Errorf("invalid export.port in config.yml, port %d is not available", port)
	}

	metricPrefix := config.Prometheus.MetricPrefix
	if err := validatePrometheusMetricPrefix(metricPrefix); err != nil {
		return models.ParsedExportConfig{}, err
	}

	return models.ParsedExportConfig{
		Port: port,
		Prometheus: models.ParsedPrometheusConfig{
			MetricPrefix: metricPrefix,
		},
	}, nil
}

func isPortAvailable(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf(":%d", port), time.Second)
	if err != nil {
		return true
	}
	conn.Close()
	return false
}

func validatePrometheusMetricPrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("invalid prometheus.metric-prefix in config.yml, prefix cannot be empty")
	}

	validName := regexp.MustCompile(ValidPrometheusName)
	if !validName.MatchString(prefix) {
		return fmt.Errorf("invalid prometheus.metric-prefix in config.yml, prefix '%s' is not valid", prefix)
	}

	if strings.HasPrefix(prefix, "_") {
		return fmt.Errorf("invalid prometheus.metric-prefix in config.yml, prefix '%s' cannot start with '_'", prefix)
	}

	return nil
}

func GetOrDefault[T cmp.Ordered](value, min, max, defaultValue T, fieldName string) T {
	if value < min || value > max {
		log.Printf("[CONFIG] %s %v is outside the allowed range [%v, %v], setting to %v", fieldName, value, min, max, defaultValue)
		return defaultValue
	}
	return value
}
