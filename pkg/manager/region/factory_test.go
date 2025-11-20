package region

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
)

func TestNewRegionManagerFactory(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "creates new factory successfully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewRegionManagerFactory()

			assert.NotNil(t, factory)
		})
	}
}

func TestCreateRegionManager(t *testing.T) {
	testCases := []struct {
		name           string
		config         *models.ParsedConfig
		expectedType   string
		expectedRegion int
		shouldError    bool
	}{
		{
			name:           "creates multi region manager with single region",
			config:         testutils.CreateDefaultTestConfig(),
			expectedType:   "*region.MultiRegionManager",
			expectedRegion: 1,
			shouldError:    false,
		},
		{
			name: "creates multi region manager with multiple regions",
			config: &models.ParsedConfig{
				Discovery: models.ParsedDiscoveryConfig{
					Regions: []string{"us-west-2", "us-east-1"},
					Instances: models.ParsedInstancesConfig{
						MaxInstances: testutils.TestMaxInstances,
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
			},
			expectedType:   "*region.MultiRegionManager",
			expectedRegion: 2,
			shouldError:    false,
		},
		{
			name: "creates multi region manager with no regions",
			config: &models.ParsedConfig{
				Discovery: models.ParsedDiscoveryConfig{
					Regions: []string{},
					Instances: models.ParsedInstancesConfig{
						MaxInstances: testutils.TestMaxInstances,
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
			},
			expectedType:   "*region.MultiRegionManager",
			expectedRegion: 0,
			shouldError:    false,
		},
		{
			name:           "creates multi region manager with maxInstances",
			config:         testutils.CreateTestConfig(testutils.TestMaxInstances),
			expectedType:   "*region.MultiRegionManager",
			expectedRegion: 1,
			shouldError:    false,
		},
		{
			name:           "creates multi region manager with maxInstances = 1",
			config:         testutils.CreateTestConfig(1),
			expectedType:   "*region.MultiRegionManager",
			expectedRegion: 1,
			shouldError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewRegionManagerFactory()

			regionManager, err := factory.CreateRegionManager(tc.config)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, regionManager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, regionManager)

				multiRM, ok := regionManager.(*MultiRegionManager)
				assert.True(t, ok, "Expected MultiRegionManager type")
				assert.Len(t, multiRM.RegionManagers, tc.expectedRegion)
			}
		})
	}
}

func TestCreateSingleRegionManager(t *testing.T) {
	testCases := []struct {
		name        string
		region      string
		config      *models.ParsedConfig
		shouldError bool
	}{
		{
			name:        "creates single region manager for us-west-2",
			region:      "us-west-2",
			config:      testutils.CreateDefaultTestConfig(),
			shouldError: false,
		},
		{
			name:        "creates single region manager for us-east-1",
			region:      "us-east-1",
			config:      testutils.CreateDefaultTestConfig(),
			shouldError: false,
		},
		{
			name:        "creates single region manager with maxInstances",
			region:      "us-west-2",
			config:      testutils.CreateTestConfig(testutils.TestMaxInstances),
			shouldError: false,
		},
		{
			name:        "creates single region manager with maxInstances = 1",
			region:      "eu-west-1",
			config:      testutils.CreateTestConfig(1),
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewRegionManagerFactory()

			regionManager, err := factory.createSingleRegionManager(tc.region, tc.config)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, regionManager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, regionManager)

				singleRM, ok := regionManager.(*SingleRegionManager)
				assert.True(t, ok, "Expected SingleRegionManager type")
				assert.Equal(t, tc.region, singleRM.region)
				assert.NotNil(t, singleRM.instanceManager)
				assert.NotNil(t, singleRM.metricManager)
			}
		})
	}
}
