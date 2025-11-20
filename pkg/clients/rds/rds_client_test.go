package rds

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
)

func TestNewRDSClient(t *testing.T) {
	t.Run("creates new RDS client successfully", func(t *testing.T) {
		rdsClient, err := NewRDSClient(testutils.TestRegion)
		assert.NoError(t, err)
		assert.NotNil(t, rdsClient)
		assert.NotNil(t, rdsClient.client)
	})

	t.Run("creates new RDS client with valid region", func(t *testing.T) {
		regions := []string{"us-west-2", "us-east-1", "eu-west-1"}
		for _, region := range regions {
			rdsClient, err := NewRDSClient(region)
			assert.NoError(t, err)
			assert.NotNil(t, rdsClient)
			assert.NotNil(t, rdsClient.client)
		}
	})
}

func TestDescribeDBInstancesPaginatorIntegration(t *testing.T) {
	testCases := []struct {
		name            string
		region          string
		expectError     bool
		skipIntegration bool
	}{
		{
			name:            "integration test - describe instances with pagination in us-west-2",
			region:          "us-west-2",
			expectError:     false,
			skipIntegration: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipIntegration {
				t.Skip("Skipping integration test - requires AWS credentials and actual RDS instances")
			}

			rdsClient, err := NewRDSClient(tc.region)
			assert.NoError(t, err)

			instances, err := rdsClient.DescribeDBInstancesPaginator(context.Background())
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, instances)
				t.Logf("Retrieved %d DB instances from %s", len(instances), tc.region)
			}
		})
	}
}
