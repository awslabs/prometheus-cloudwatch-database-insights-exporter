package region

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			config:         testutils.CreateDefaultParsedTestConfig(),
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
						Statistic: models.StatisticAvg,
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
						Statistic: models.StatisticAvg,
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
			config:         testutils.CreateParsedTestConfig(testutils.TestMaxInstances),
			expectedType:   "*region.MultiRegionManager",
			expectedRegion: 1,
			shouldError:    false,
		},
		{
			name:           "creates multi region manager with maxInstances = 1",
			config:         testutils.CreateParsedTestConfig(1),
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
			config:      testutils.CreateDefaultParsedTestConfig(),
			shouldError: false,
		},
		{
			name:        "creates single region manager for us-east-1",
			region:      "us-east-1",
			config:      testutils.CreateDefaultParsedTestConfig(),
			shouldError: false,
		},
		{
			name:        "creates single region manager with maxInstances",
			region:      "us-west-2",
			config:      testutils.CreateParsedTestConfig(testutils.TestMaxInstances),
			shouldError: false,
		},
		{
			name:        "creates single region manager with maxInstances = 1",
			region:      "eu-west-1",
			config:      testutils.CreateParsedTestConfig(1),
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

func TestCreateSingleRegionManagerRoleARNWiring(t *testing.T) {
	const stsXML = `<AssumeRoleResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><AssumeRoleResult><Credentials><AccessKeyId>TEST-ACCESS-KEY-ID-0</AccessKeyId><SecretAccessKey>TEST-SECRET-KEY-NOT-REAL-000000000000</SecretAccessKey><SessionToken>assumed-session-token</SessionToken><Expiration>2030-01-01T00:00:00Z</Expiration></Credentials><AssumedRoleUser><Arn>arn:aws:sts::123456789012:assumed-role/TestRole/rds-pi-exporter</Arn><AssumedRoleId>AROATEST:rds-pi-exporter</AssumedRoleId></AssumedRoleUser></AssumeRoleResult><ResponseMetadata><RequestId>test</RequestId></ResponseMetadata></AssumeRoleResponse>`
	const rdsXML = `<DescribeDBInstancesResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/"><DescribeDBInstancesResult><DBInstances/></DescribeDBInstancesResult><ResponseMetadata><RequestId>test</RequestId></ResponseMetadata></DescribeDBInstancesResponse>`

	newMockServer := func(t *testing.T, capture *string, mu *sync.Mutex) *httptest.Server {
		t.Helper()
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), "AssumeRole") {
				mu.Lock()
				*capture = string(body)
				mu.Unlock()
				w.Header().Set("Content-Type", "text/xml")
				fmt.Fprint(w, stsXML)
			} else {
				w.Header().Set("Content-Type", "text/xml")
				fmt.Fprint(w, rdsXML)
			}
		}))
	}

	t.Run("RoleARN from config reaches STS AssumeRole", func(t *testing.T) {
		var mu sync.Mutex
		var stsBody string
		server := newMockServer(t, &stsBody, &mu)
		defer server.Close()

		t.Setenv("AWS_ENDPOINT_URL", server.URL)
		t.Setenv("AWS_ACCESS_KEY_ID", "AKIDBASE")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "SECRETBASE")
		t.Setenv("AWS_SESSION_TOKEN", "")

		cfg := &models.ParsedConfig{
			Discovery: models.ParsedDiscoveryConfig{
				RoleARN: "arn:aws:iam::123456789012:role/TestRole",
				Instances: models.ParsedInstancesConfig{MaxInstances: testutils.TestMaxInstances},
				Metrics:   models.ParsedMetricsConfig{Statistic: models.StatisticAvg},
			},
		}

		factory := NewRegionManagerFactory()
		rm, err := factory.createSingleRegionManager("us-west-2", cfg)
		require.NoError(t, err)

		// GetInstances triggers the first credential retrieval, which calls STS AssumeRole.
		_, _ = rm.(*SingleRegionManager).instanceManager.GetInstances(context.Background())

		mu.Lock()
		defer mu.Unlock()
		require.NotEmpty(t, stsBody, "STS AssumeRole should have been called")
		values, err := url.ParseQuery(stsBody)
		require.NoError(t, err)
		assert.Equal(t, "arn:aws:iam::123456789012:role/TestRole", values.Get("RoleArn"))
	})

	t.Run("ExternalID from config reaches STS AssumeRole", func(t *testing.T) {
		var mu sync.Mutex
		var stsBody string
		server := newMockServer(t, &stsBody, &mu)
		defer server.Close()

		t.Setenv("AWS_ENDPOINT_URL", server.URL)
		t.Setenv("AWS_ACCESS_KEY_ID", "AKIDBASE")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "SECRETBASE")
		t.Setenv("AWS_SESSION_TOKEN", "")

		cfg := &models.ParsedConfig{
			Discovery: models.ParsedDiscoveryConfig{
				RoleARN:           "arn:aws:iam::123456789012:role/TestRole",
				RoleARNExternalID: "my-external-id",
				Instances:         models.ParsedInstancesConfig{MaxInstances: testutils.TestMaxInstances},
				Metrics:           models.ParsedMetricsConfig{Statistic: models.StatisticAvg},
			},
		}

		factory := NewRegionManagerFactory()
		rm, err := factory.createSingleRegionManager("us-west-2", cfg)
		require.NoError(t, err)

		_, _ = rm.(*SingleRegionManager).instanceManager.GetInstances(context.Background())

		mu.Lock()
		defer mu.Unlock()
		require.NotEmpty(t, stsBody)
		values, err := url.ParseQuery(stsBody)
		require.NoError(t, err)
		assert.Equal(t, "my-external-id", values.Get("ExternalId"))
	})

	t.Run("RoleSessionName reaches STS AssumeRole", func(t *testing.T) {
		var mu sync.Mutex
		var stsBody string
		server := newMockServer(t, &stsBody, &mu)
		defer server.Close()

		t.Setenv("AWS_ENDPOINT_URL", server.URL)
		t.Setenv("AWS_ACCESS_KEY_ID", "AKIDBASE")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "SECRETBASE")
		t.Setenv("AWS_SESSION_TOKEN", "")

		cfg := &models.ParsedConfig{
			Discovery: models.ParsedDiscoveryConfig{
				RoleARN:   "arn:aws:iam::123456789012:role/TestRole",
				Instances: models.ParsedInstancesConfig{MaxInstances: testutils.TestMaxInstances},
				Metrics:   models.ParsedMetricsConfig{Statistic: models.StatisticAvg},
			},
		}

		factory := NewRegionManagerFactory()
		rm, err := factory.createSingleRegionManager("us-west-2", cfg)
		require.NoError(t, err)

		_, _ = rm.(*SingleRegionManager).instanceManager.GetInstances(context.Background())

		mu.Lock()
		defer mu.Unlock()
		require.NotEmpty(t, stsBody)
		values, err := url.ParseQuery(stsBody)
		require.NoError(t, err)
		assert.Equal(t, "rds-pi-exporter", values.Get("RoleSessionName"))
	})

	t.Run("single STS AssumeRole call is shared by both RDS and PI clients", func(t *testing.T) {
		var mu sync.Mutex
		stsCallCount := 0
		const piJSON = `{"Metrics":[]}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-amz-json") {
				w.Header().Set("Content-Type", "application/x-amz-json-1.1")
				fmt.Fprint(w, piJSON)
				return
			}
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), "AssumeRole") {
				mu.Lock()
				stsCallCount++
				mu.Unlock()
				w.Header().Set("Content-Type", "text/xml")
				fmt.Fprint(w, stsXML)
			} else {
				w.Header().Set("Content-Type", "text/xml")
				fmt.Fprint(w, rdsXML)
			}
		}))
		defer server.Close()

		t.Setenv("AWS_ENDPOINT_URL", server.URL)
		t.Setenv("AWS_ACCESS_KEY_ID", "AKIDBASE")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "SECRETBASE")
		t.Setenv("AWS_SESSION_TOKEN", "")

		cfg := &models.ParsedConfig{
			Discovery: models.ParsedDiscoveryConfig{
				RoleARN:   "arn:aws:iam::123456789012:role/TestRole",
				Instances: models.ParsedInstancesConfig{MaxInstances: testutils.TestMaxInstances},
				Metrics:   models.ParsedMetricsConfig{Statistic: models.StatisticAvg},
			},
		}

		factory := NewRegionManagerFactory()
		rm, err := factory.createSingleRegionManager("us-west-2", cfg)
		require.NoError(t, err)

		singleRM := rm.(*SingleRegionManager)

		// Trigger RDS credential retrieval via instance discovery.
		_, _ = singleRM.instanceManager.GetInstances(context.Background())

		// Trigger PI credential retrieval: Metrics field being non-nil but empty forces
		// GetMetricBatches to call ListAvailableResourceMetrics on the PI client.
		dummyInstance := models.Instance{
			ResourceID: "db-TESTRESOURCEID",
			Engine:     models.Engine("mysql"),
			Metrics:    &models.Metrics{},
		}
		_, _ = singleRM.metricManager.GetMetricBatches(context.Background(), dummyInstance)

		// Both clients share one CredentialsCache, so STS is called exactly once for both.
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 1, stsCallCount, "shared CredentialsCache must result in exactly one STS AssumeRole call")
	})

	t.Run("no RoleARN in config does not call STS AssumeRole", func(t *testing.T) {
		var mu sync.Mutex
		var stsBody string
		server := newMockServer(t, &stsBody, &mu)
		defer server.Close()

		t.Setenv("AWS_ENDPOINT_URL", server.URL)
		t.Setenv("AWS_ACCESS_KEY_ID", "AKIDBASE")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "SECRETBASE")
		t.Setenv("AWS_SESSION_TOKEN", "")

		cfg := &models.ParsedConfig{
			Discovery: models.ParsedDiscoveryConfig{
				Instances: models.ParsedInstancesConfig{MaxInstances: testutils.TestMaxInstances},
				Metrics:   models.ParsedMetricsConfig{Statistic: models.StatisticAvg},
			},
		}

		factory := NewRegionManagerFactory()
		rm, err := factory.createSingleRegionManager("us-west-2", cfg)
		require.NoError(t, err)

		_, _ = rm.(*SingleRegionManager).instanceManager.GetInstances(context.Background())

		mu.Lock()
		defer mu.Unlock()
		assert.Empty(t, stsBody, "STS AssumeRole must not be called when RoleARN is empty")
	})
}
