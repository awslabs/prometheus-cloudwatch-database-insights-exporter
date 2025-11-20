package instance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/clients/rds"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils/mocks"
)

func TestNewRDSInstanceManager(t *testing.T) {
	testCases := []struct {
		name           string
		mockRDSService rds.RDSService
		config         *models.ParsedConfig
	}{
		{
			name:           "valid RDS service with default config",
			mockRDSService: &mocks.MockRDSService{},
			config:         testutils.CreateDefaultTestConfig(),
		},
		{
			name:           "nil RDS service with config",
			mockRDSService: nil,
			config:         testutils.CreateDefaultTestConfig(),
		},
		{
			name:           "valid RDS service with maxInstances config",
			mockRDSService: &mocks.MockRDSService{},
			config:         testutils.CreateTestConfig(testutils.TestMaxInstances),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager, err := NewRDSInstanceManager(tc.mockRDSService, tc.config)
			require.NoError(t, err)

			assert.NotNil(t, manager)
			assert.Equal(t, tc.mockRDSService, manager.rdsService)
			assert.Equal(t, tc.config, manager.Configuration)
			assert.Empty(t, manager.Instances)
			assert.True(t, manager.InstancesLastUpdated.Before(time.Now().Add(-5*time.Minute)))
		})
	}
}

func TestGetInstances(t *testing.T) {
	testCases := []struct {
		name          string
		setupManager  func() *RDSInstanceManager
		mockResponse  interface{}
		expectedError error
		shouldCallRDS bool
		expectedCount int
	}{
		{
			name: "get instances within instanceTTL",
			setupManager: func() *RDSInstanceManager {
				mockRDSService := &mocks.MockRDSService{}
				manager, _ := NewRDSInstanceManager(mockRDSService, testutils.CreateDefaultTestConfig())
				manager.Instances = testutils.TestInstances
				manager.InstancesLastUpdated = time.Now()
				return manager
			},
			mockResponse:  nil,
			expectedError: nil,
			shouldCallRDS: false,
			expectedCount: 2,
		},
		{
			name: "get instances with expired cache success",
			setupManager: func() *RDSInstanceManager {
				mockRDSService := &mocks.MockRDSService{}
				manager, _ := NewRDSInstanceManager(mockRDSService, testutils.CreateDefaultTestConfig())
				return manager
			},
			mockResponse:  mocks.NewMockRDSDescribeInstances(),
			expectedError: nil,
			shouldCallRDS: true,
			expectedCount: 2,
		},
		{
			name: "get instances with expired cache error",
			setupManager: func() *RDSInstanceManager {
				mockRDSService := &mocks.MockRDSService{}
				manager, _ := NewRDSInstanceManager(mockRDSService, testutils.CreateDefaultTestConfig())
				return manager
			},
			mockResponse:  nil,
			expectedError: errors.New("RDS API error"),
			shouldCallRDS: true,
			expectedCount: 0,
		},
		{
			name: "get instances with no cached data and empty RDS response",
			setupManager: func() *RDSInstanceManager {
				mockRDSService := &mocks.MockRDSService{}
				manager, _ := NewRDSInstanceManager(mockRDSService, testutils.CreateDefaultTestConfig())
				return manager
			},
			mockResponse:  mocks.NewMockRDSDescribeInstancesEmpty(),
			expectedError: nil,
			shouldCallRDS: true,
			expectedCount: 0,
		},
		{
			name: "get instances limits to maxInstances = 1 when more available",
			setupManager: func() *RDSInstanceManager {
				mockRDSService := &mocks.MockRDSService{}
				manager, _ := NewRDSInstanceManager(mockRDSService, testutils.CreateTestConfig(1))
				return manager
			},
			mockResponse:  mocks.NewMockRDSDescribeInstances(),
			expectedError: nil,
			shouldCallRDS: true,
			expectedCount: 1,
		},
		{
			name: "get instances returns all when fewer than maxInstances",
			setupManager: func() *RDSInstanceManager {
				mockRDSService := &mocks.MockRDSService{}
				manager, _ := NewRDSInstanceManager(mockRDSService, testutils.CreateTestConfig(testutils.TestMaxInstances/2))
				return manager
			},
			mockResponse:  mocks.NewMockRDSDescribeInstances(),
			expectedError: nil,
			shouldCallRDS: true,
			expectedCount: 2,
		},
		{
			name: "get instances with maxInstances = 0 (edge case) returns none",
			setupManager: func() *RDSInstanceManager {
				mockRDSService := &mocks.MockRDSService{}
				manager, _ := NewRDSInstanceManager(mockRDSService, testutils.CreateTestConfig(0))
				return manager
			},
			mockResponse:  mocks.NewMockRDSDescribeInstances(),
			expectedError: nil,
			shouldCallRDS: true,
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := tc.setupManager()

			if tc.shouldCallRDS {
				manager.rdsService.(*mocks.MockRDSService).On("DescribeDBInstancesPaginator", mock.Anything).
					Return(tc.mockResponse, tc.expectedError)
			}

			instances, err := manager.GetInstances(context.Background())

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, instances)
			} else {
				assert.NoError(t, err)
				assert.Len(t, instances, tc.expectedCount)

				if tc.shouldCallRDS && tc.expectedError == nil {
					assert.True(t, manager.InstancesLastUpdated.After(time.Now().Add(-1*time.Minute)))
				}
			}

			if tc.shouldCallRDS {
				manager.rdsService.(*mocks.MockRDSService).AssertExpectations(t)
			}
		})
	}
}

func TestDiscoverInstances(t *testing.T) {
	testCases := []struct {
		name              string
		mockResponse      interface{}
		expectedInstances []models.Instance
		expectedError     error
		expectedCount     int
		shouldCallRDS     bool
	}{
		{
			name:              "discover instances success with multiple instances",
			mockResponse:      mocks.NewMockRDSDescribeInstances(),
			expectedInstances: testutils.TestInstances,
			expectedError:     nil,
			expectedCount:     2,
			shouldCallRDS:     true,
		},
		{
			name:              "discover instances success with single instance",
			mockResponse:      mocks.NewMockRDSDescribeInstancesSingle(),
			expectedInstances: []models.Instance{testutils.TestInstancePostgreSQL},
			expectedError:     nil,
			expectedCount:     1,
			shouldCallRDS:     true,
		},
		{
			name:              "discover instances success with empty response",
			mockResponse:      mocks.NewMockRDSDescribeInstancesEmpty(),
			expectedInstances: []models.Instance{},
			expectedError:     nil,
			expectedCount:     0,
			shouldCallRDS:     true,
		},
		{
			name:              "discover instances with RDS error",
			mockResponse:      nil,
			expectedInstances: nil,
			expectedError:     errors.New("RDS describe failed"),
			expectedCount:     0,
			shouldCallRDS:     true,
		},
		{
			name:         "discover instances returns sorted by creation time (oldest first)",
			mockResponse: mocks.NewMockRDSDescribeInstances(),
			expectedInstances: []models.Instance{
				testutils.TestInstanceMySQL,
				testutils.TestInstancePostgreSQL,
			},
			expectedError: nil,
			expectedCount: 2,
			shouldCallRDS: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRDS := &mocks.MockRDSService{}
			manager, _ := NewRDSInstanceManager(mockRDS, testutils.CreateDefaultTestConfig())

			if tc.shouldCallRDS {
				mockRDS.On("DescribeDBInstancesPaginator", mock.Anything).
					Return(tc.mockResponse, tc.expectedError)
			}

			instances, err := manager.discoverInstances(context.Background())

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, instances)
			} else {
				assert.NoError(t, err)
				assert.Len(t, instances, tc.expectedCount)

				for i, instance := range instances {
					assert.Equal(t, tc.expectedInstances[i].ResourceID, instance.ResourceID)
					assert.Equal(t, tc.expectedInstances[i].Identifier, instance.Identifier)
					assert.Equal(t, tc.expectedInstances[i].Engine, instance.Engine)
					assert.Equal(t, tc.expectedInstances[i].CreationTime, instance.CreationTime)

					if i > 0 {
						assert.True(t, instances[i-1].CreationTime.Before(instances[i].CreationTime) || instances[i-1].CreationTime.Equal(instances[i].CreationTime),
							"Instances should be sorted by CreationTime (oldest first): %v should be before or equal to %v",
							instances[i-1].CreationTime, instances[i].CreationTime)
					}
				}
			}

			mockRDS.AssertExpectations(t)
		})
	}
}
