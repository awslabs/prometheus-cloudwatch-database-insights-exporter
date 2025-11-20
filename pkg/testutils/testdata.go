package testutils

import (
	"time"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
)

var (
	TestRegion = "us-west-2"
)

var (
	TestTimestamp    = time.Date(2025, 10, 28, 10, 0, 0, 0, time.UTC) //OCTOBER 28th 2025 3PM PST
	TestTimestampNow = time.Now()
	TestTTL          = 5 * time.Minute
	TestTimestampNil = time.Time{}

	TestInstanceCreationTimeMySQL      = time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)  // JAN 5, 2024 OLD
	TestInstanceCreationTimePostgreSQL = time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC) // MAY 20, 2024 NEW
	TestInstanceCreationTimeExpired    = time.Date(2023, 10, 8, 0, 0, 0, 0, time.UTC) // OCT 8, 2023 OLDEST
	TestInstanceCreationTimeNoMetrics  = time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC) // MAR 10, 2024
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
			MetricsTTL:         TestTTL,
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
			MetricsTTL:         TestTTL,
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
			MetricsTTL:         TestTTL,
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
			MetricsTTL:         TestTTL,
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
			MetricsTTL:         TestTTL,
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
			MetricsTTL:         TestTTL,
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
			MetricsTTL:         TestTTL,
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
			MetricsTTL:         TestTTL,
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
			MetricsTTL:         TestTTL,
		},
	}
}

func CreateTestConfig(maxInstances int) *models.ParsedConfig {
	return &models.ParsedConfig{
		Discovery: models.ParsedDiscoveryConfig{
			Regions: []string{"us-west-2"},
			Instances: models.ParsedInstancesConfig{
				MaxInstances: maxInstances,
			},
			Metrics: models.ParsedMetricsConfig{
				Statistic: models.MetricStatisticConfig{
					Default: models.StatisticAvg,
				},
			},
		},
		Export: models.ParsedExportConfig{
			Port: 8081,
		},
	}
}

func CreateDefaultTestConfig() *models.ParsedConfig {
	return CreateTestConfig(TestMaxInstances)
}
