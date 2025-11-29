package testutils

import (
	"time"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
)

var (
	TestRegion = "us-west-2"
)

var (
	TestTimestamp    = time.Date(2025, 10, 28, 10, 0, 0, 0, time.UTC) // OCT 28, 2025 3PM PST
	TestTimestampNow = time.Now()
	TestTTL          = 5 * time.Minute
	TestTimestampNil = time.Time{}

	TestInstanceCreationTimeMySQL      = time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)  // JAN 5, 2024 OLD
	TestInstanceCreationTimePostgreSQL = time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC) // MAY 20, 2024 NEW
	TestInstanceCreationTimeExpired    = time.Date(2023, 10, 8, 0, 0, 0, 0, time.UTC) // OCT 8, 2023 OLDEST
	TestInstanceCreationTimeNoMetrics  = time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC) // NOV 21, 2025
)

var (
	TestEnginePostgreSQL = models.PostgreSQL
	TestEngineMySQL      = models.MySQL
)

var (
	TestMetricsDetails = map[string]models.MetricDetails{
		"os.general.numVCPUs": {
			Name:        "os.general.numVCPUs",
			Description: "The number of virtual CPUs for the DB instance",
			Unit:        "vCPUs",
			Statistics:  []models.Statistic{models.StatisticAvg},
		},
		"os.cpuUtilization.guest": {
			Name:        "os.cpuUtilization.guest",
			Description: "The percentage of CPU in use by guest programs",
			Unit:        "Percent",
			Statistics:  []models.Statistic{models.StatisticAvg},
		},
		"os.cpuUtilization.idle": {
			Name:        "os.cpuUtilization.idle",
			Description: "The percentage of CPU that is idle",
			Unit:        "Percent",
			Statistics:  []models.Statistic{models.StatisticAvg},
		},
		"os.memory.total": {
			Name:        "os.memory.total",
			Description: "The total amount of memory in kilobytes",
			Unit:        "KB",
			Statistics:  []models.Statistic{models.StatisticAvg},
		},
		"db.User.max_connections": {
			Name:        "db.User.max_connections",
			Description: "The maximum number of connections allowed for a DB instance as configured in max_connections parameter",
			Unit:        "Connections",
			Statistics:  []models.Statistic{models.StatisticAvg},
		},
	}

	TestMetricsDetailsSmall = map[string]models.MetricDetails{
		"os.general.numVCPUs":     TestMetricsDetails["os.general.numVCPUs"],
		"os.cpuUtilization.guest": TestMetricsDetails["os.cpuUtilization.guest"],
	}

	TestMetricsDetailsEmpty = map[string]models.MetricDetails{}
)

var (
	TestInstancePostgreSQL = models.Instance{
		ResourceID:   "db-TESTPOSTGRES",
		Identifier:   "test-postgres-db",
		Engine:       models.AuroraPostgreSQL,
		CreationTime: TestInstanceCreationTimePostgreSQL,
		Metrics: &models.Metrics{
			MetricsDetails:     TestMetricsDetails,
			MetricsList:        TestMetricNamesWithStats,
			MetricsLastUpdated: TestTimestampNow,
			MetadataTTL:        TestTTL,
		},
	}

	TestInstanceMySQL = models.Instance{
		ResourceID:   "db-TESTMYSQL",
		Identifier:   "test-mysql-db",
		Engine:       models.AuroraMySQL,
		CreationTime: TestInstanceCreationTimeMySQL,
		Metrics: &models.Metrics{
			MetricsDetails:     TestMetricsDetailsSmall,
			MetricsList:        TestMetricNamesWithStatsSmall,
			MetricsLastUpdated: TestTimestampNow,
			MetadataTTL:        TestTTL,
		},
	}

	TestInstancePostgreSQLExpired = models.Instance{
		ResourceID:   "db-TESTPOSTGRES-EXPIRED",
		Identifier:   "test-postgres-db-expired",
		Engine:       models.AuroraPostgreSQL,
		CreationTime: TestInstanceCreationTimeExpired,
		Metrics: &models.Metrics{
			MetricsDetails:     TestMetricsDetails,
			MetricsList:        TestMetricNamesWithStats,
			MetricsLastUpdated: TestTimestamp,
			MetadataTTL:        TestTTL,
		},
	}

	TestInstanceNoMetrics = models.Instance{
		ResourceID:   "db-TESTEMPTY",
		Identifier:   "test-empty-db",
		Engine:       models.AuroraPostgreSQL,
		CreationTime: TestInstanceCreationTimeNoMetrics,
		Metrics: &models.Metrics{
			MetricsDetails:     nil,
			MetricsList:        []string{},
			MetricsLastUpdated: time.Time{},
			MetadataTTL:        TestTTL,
		},
	}

	TestInstanceInvalid = models.Instance{
		ResourceID:   "db-TESTINVALID",
		Identifier:   "",
		Engine:       TestEngineMySQL,
		CreationTime: TestInstanceCreationTimeMySQL,
		Metrics: &models.Metrics{
			MetricsDetails:     TestMetricsDetails,
			MetricsList:        TestMetricNamesWithStats,
			MetricsLastUpdated: TestTimestampNow,
			MetadataTTL:        TestTTL,
		},
	}

	TestInstances = []models.Instance{
		TestInstanceMySQL,
		TestInstancePostgreSQL,
	}
)

var (
	TestMetricData = []models.MetricData{
		{
			Metric:    "os.general.numVCPUs.avg",
			Timestamp: TestTimestamp,
			Value:     4.0,
		},
		{
			Metric:    "os.cpuUtilization.guest.avg",
			Timestamp: TestTimestamp,
			Value:     25.5,
		},
		{
			Metric:    "os.cpuUtilization.idle.avg",
			Timestamp: TestTimestamp,
			Value:     74.5,
		},
		{
			Metric:    "os.memory.total.avg",
			Timestamp: TestTimestamp,
			Value:     16.0,
		},
		{
			Metric:    "db.User.max_connections.avg",
			Timestamp: TestTimestamp,
			Value:     2.0,
		},
	}

	TestMetricDataSmall = []models.MetricData{
		TestMetricData[0],
		TestMetricData[1],
	}

	TestMetricDataEmpty = []models.MetricData{}
)

var (
	TestMetricNames = []string{
		"os.general.numVCPUs",
		"os.cpuUtilization.guest",
		"os.cpuUtilization.idle",
		"os.memory.total",
		"db.User.max_connections",
	}

	TestMetricNamesSmall = []string{
		"os.general.numVCPUs",
		"os.cpuUtilization.guest",
	}

	TestMetricNamesEmpty = []string{}

	TestMetricNamesWithStats = []string{
		"os.general.numVCPUs.avg",
		"os.cpuUtilization.guest.avg",
		"os.cpuUtilization.idle.avg",
		"os.memory.total.avg",
		"db.User.max_connections.avg",
	}

	TestMetricNamesWithStatsSmall = []string{
		"os.general.numVCPUs.avg",
		"os.cpuUtilization.guest.avg",
	}

	TestSnakeCaseMetricNamesWithStats = []string{
		"os_general_numvcpus_avg",
		"os_cpuutilization_guest_avg",
		"os_cpuutilization_idle_avg",
		"os_memory_total_avg",
		"db_user_max_connections_avg",
	}

	TestSnakeCaseMetricNamesWithStatsSmall = []string{
		"os_general_numvcpus_avg",
		"os_cpuutilization_guest_avg",
	}
)

var (
	TestInstancesResourceIDs = []string{
		"db-TESTPOSTGRES",
		"db-TESTMYSQL",
	}

	TestInstancesResourceIDsEmpty = []string{}
)

var (
	TestMaxInstances = 25
)

// TestConfigBuilder provides a fluent interface for building test configurations
type TestConfigBuilder struct {
	regions      []string
	maxInstances int
	instanceTTL  time.Duration
	statistic    models.Statistic
	metadataTTL  time.Duration
	concurrency  int
	port         int
	metricPrefix string
}

func NewTestInstance(resourceID, identifier string, engine models.Engine) models.Instance {
	return models.Instance{
		ResourceID:   resourceID,
		Identifier:   identifier,
		Engine:       engine,
		CreationTime: TestInstanceCreationTimeMySQL,
		Metrics: &models.Metrics{
			MetricsDetails:     TestMetricsDetails,
			MetricsList:        TestMetricNamesWithStats,
			MetricsLastUpdated: TestTimestampNow,
			MetadataTTL:        TestTTL,
		},
	}
}

func NewTestMetricData(metricName string, value float64) models.MetricData {
	return models.MetricData{
		Metric:    metricName,
		Timestamp: TestTimestamp,
		Value:     value,
	}
}

func NewTestMetricDetails(name, description, unit string) models.MetricDetails {
	return models.MetricDetails{
		Name:        name,
		Description: description,
		Unit:        unit,
		Statistics:  []models.Statistic{models.StatisticAvg},
	}
}

func NewTestInstancePostgreSQL() models.Instance {
	return models.Instance{
		ResourceID:   "db-TESTPOSTGRES",
		Identifier:   "test-postgres-db",
		Engine:       models.AuroraPostgreSQL,
		CreationTime: TestInstanceCreationTimePostgreSQL,
		Metrics: &models.Metrics{
			MetricsDetails:     TestMetricsDetails,
			MetricsList:        TestMetricNamesWithStats,
			MetricsLastUpdated: TestTimestampNow,
			MetadataTTL:        TestTTL,
		},
	}
}

func NewTestInstancePostgreSQLExpired() models.Instance {
	return models.Instance{
		ResourceID:   "db-TESTPOSTGRES-EXPIRED",
		Identifier:   "test-postgres-db-expired",
		Engine:       models.AuroraPostgreSQL,
		CreationTime: TestInstanceCreationTimeExpired,
		Metrics: &models.Metrics{
			MetricsDetails:     TestMetricsDetails,
			MetricsList:        TestMetricNamesWithStats,
			MetricsLastUpdated: TestTimestamp,
			MetadataTTL:        TestTTL,
		},
	}
}

func NewTestInstanceNoMetrics() models.Instance {
	return models.Instance{
		ResourceID:   "db-TESTEMPTY",
		Identifier:   "test-empty-db",
		Engine:       models.AuroraPostgreSQL,
		CreationTime: TestInstanceCreationTimeNoMetrics,
		Metrics: &models.Metrics{
			MetricsDetails:     nil,
			MetricsList:        []string{},
			MetricsLastUpdated: time.Time{},
			MetadataTTL:        TestTTL,
		},
	}
}

func NewTestConfigBuilder() *TestConfigBuilder {
	return &TestConfigBuilder{
		regions:      []string{"us-west-2"},
		maxInstances: TestMaxInstances,
		instanceTTL:  5 * time.Minute,
		statistic:    models.StatisticAvg,
		metadataTTL:  60 * time.Minute,
		concurrency:  4,
		port:         8081,
		metricPrefix: "dbi",
	}
}

func (b *TestConfigBuilder) WithRegions(regions []string) *TestConfigBuilder {
	b.regions = regions
	return b
}

func (b *TestConfigBuilder) WithMaxInstances(maxInstances int) *TestConfigBuilder {
	b.maxInstances = maxInstances
	return b
}

func (b *TestConfigBuilder) WithInstanceTTL(ttl time.Duration) *TestConfigBuilder {
	b.instanceTTL = ttl
	return b
}

func (b *TestConfigBuilder) WithStatistic(statistic models.Statistic) *TestConfigBuilder {
	b.statistic = statistic
	return b
}

func (b *TestConfigBuilder) WithMetadataTTL(ttl time.Duration) *TestConfigBuilder {
	b.metadataTTL = ttl
	return b
}

func (b *TestConfigBuilder) WithConcurrency(concurrency int) *TestConfigBuilder {
	b.concurrency = concurrency
	return b
}

func (b *TestConfigBuilder) WithPort(port int) *TestConfigBuilder {
	b.port = port
	return b
}

func (b *TestConfigBuilder) WithMetricPrefix(prefix string) *TestConfigBuilder {
	b.metricPrefix = prefix
	return b
}

func (b *TestConfigBuilder) Build() *models.ParsedConfig {
	return &models.ParsedConfig{
		Discovery: models.ParsedDiscoveryConfig{
			Regions: b.regions,
			Instances: models.ParsedInstancesConfig{
				MaxInstances: b.maxInstances,
				InstanceTTL:  b.instanceTTL,
			},
			Metrics: models.ParsedMetricsConfig{
				Statistic:   b.statistic,
				MetadataTTL: b.metadataTTL,
			},
			Processing: models.ParsedProcessingConfig{
				Concurrency: b.concurrency,
			},
		},
		Export: models.ParsedExportConfig{
			Port: b.port,
			Prometheus: models.ParsedPrometheusConfig{
				MetricPrefix: b.metricPrefix,
			},
		},
	}
}

func CreateParsedTestConfig(maxInstances int) *models.ParsedConfig {
	return NewTestConfigBuilder().WithMaxInstances(maxInstances).Build()
}

func CreateDefaultParsedTestConfig() *models.ParsedConfig {
	return NewTestConfigBuilder().Build()
}

func CreateTestConfig(overrides ...map[string]interface{}) *models.Config {
	cfg := &models.Config{
		Discovery: models.DiscoveryConfig{
			Regions: []string{"us-west-2"},
			Instances: models.InstancesConfig{
				MaxInstances: TestMaxInstances,
				InstanceTTL:  "5m",
			},
			Metrics: models.MetricsConfig{
				Statistic:   "avg",
				MetadataTTL: "60m",
			},
		},
		Export: models.ExportConfig{
			Port: 8081,
			Prometheus: models.PrometheusConfig{
				MetricPrefix: "dbi",
			},
		},
	}

	var override map[string]interface{}
	if len(overrides) > 0 {
		override = overrides[0]
	} else {
		override = map[string]interface{}{}
	}

	if stat, ok := override["statistic"].(string); ok {
		cfg.Discovery.Metrics.Statistic = stat
	}
	if port, ok := override["port"].(int); ok {
		cfg.Export.Port = port
	}
	if regions, ok := override["regions"].([]string); ok {
		cfg.Discovery.Regions = regions
	}

	return cfg
}
