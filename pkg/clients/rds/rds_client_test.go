package rds

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
)

// newRDSClientWithSTSClient is a test-only helper that injects a mock STS client
// so unit tests can verify the AssumeRole credential chain without network calls.
func newRDSClientWithSTSClient(cfg aws.Config, roleARN, externalID string, stsClient stscreds.AssumeRoleAPIClient) (*RDSClient, *aws.CredentialsCache) {
	credCache := utils.NewAssumeRoleCredCache(stsClient, roleARN, externalID, "rds-pi-exporter")
	cfg.Credentials = credCache
	return newRDSClientFromConfig(cfg, ""), credCache
}

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

func TestNewRDSClientWithSTSClient(t *testing.T) {
	newCfg := func(t *testing.T) aws.Config {
		t.Helper()
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(testutils.TestRegion),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKID", "SECRET", "TOKEN")),
		)
		require.NoError(t, err)
		return cfg
	}

	t.Run("AssumeRole is called on first credential retrieval and returns expected credentials", func(t *testing.T) {
		mockSTS := &testutils.MockSTSClient{}

		rdsClient, credCache := newRDSClientWithSTSClient(newCfg(t), "arn:aws:iam::123456789012:role/TestRole", "", mockSTS)
		require.NotNil(t, rdsClient)
		require.NotNil(t, rdsClient.client)

		assert.False(t, mockSTS.Called(), "STS should not be called before credential retrieval")

		creds, err := credCache.Retrieve(context.TODO())
		require.NoError(t, err)

		assert.True(t, mockSTS.Called(), "STS AssumeRole should have been called on credential retrieval")
		assert.Equal(t, "TEST-ACCESS-KEY-ID-0", creds.AccessKeyID)
		assert.Equal(t, "session-token", creds.SessionToken)
		assert.Equal(t, "arn:aws:iam::123456789012:role/TestRole", mockSTS.CapturedRoleArnValue())
		assert.Equal(t, "rds-pi-exporter", mockSTS.CapturedRoleSessionNameValue())
	})

	t.Run("ExternalID is forwarded to AssumeRole when set", func(t *testing.T) {
		mockSTS := &testutils.MockSTSClient{}

		_, credCache := newRDSClientWithSTSClient(newCfg(t), "arn:aws:iam::123456789012:role/TestRole", "my-external-id", mockSTS)

		_, err := credCache.Retrieve(context.TODO())
		require.NoError(t, err)
		assert.Equal(t, "my-external-id", mockSTS.CapturedExternalIDValue())
	})

	t.Run("ExternalID is not sent when empty", func(t *testing.T) {
		mockSTS := &testutils.MockSTSClient{}

		_, credCache := newRDSClientWithSTSClient(newCfg(t), "arn:aws:iam::123456789012:role/TestRole", "", mockSTS)

		_, err := credCache.Retrieve(context.TODO())
		require.NoError(t, err)
		assert.Empty(t, mockSTS.CapturedExternalIDValue())
	})

	t.Run("AssumeRole error is propagated through credential retrieval", func(t *testing.T) {
		mockSTS := &testutils.MockSTSClient{ReturnErr: fmt.Errorf("AccessDenied: not authorized to assume role")}

		_, credCache := newRDSClientWithSTSClient(newCfg(t), "arn:aws:iam::123456789012:role/TestRole", "", mockSTS)

		_, err := credCache.Retrieve(context.TODO())
		assert.Error(t, err)
		assert.True(t, mockSTS.Called())
		assert.Contains(t, err.Error(), "AccessDenied")
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
