package pi

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

// newPIClientWithSTSClient is a test-only helper that injects a mock STS client
// so unit tests can verify the AssumeRole credential chain without network calls.
func newPIClientWithSTSClient(cfg aws.Config, roleARN, externalID string, stsClient stscreds.AssumeRoleAPIClient) (*PIClient, *aws.CredentialsCache) {
	credCache := utils.NewAssumeRoleCredCache(stsClient, roleARN, externalID, "rds-pi-exporter")
	cfg.Credentials = credCache
	return newPIClientFromConfig(cfg, ""), credCache
}

func TestNewPIClient(t *testing.T) {
	t.Run("creates new PI client successfully", func(t *testing.T) {
		piClient, err := NewPIClient(testutils.TestRegion)
		assert.NoError(t, err)
		assert.NotNil(t, piClient)
		assert.NotNil(t, piClient.client)
	})
}

func TestNewPIClientWithSTSClient(t *testing.T) {
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

		piClient, credCache := newPIClientWithSTSClient(newCfg(t), "arn:aws:iam::123456789012:role/TestRole", "", mockSTS)
		require.NotNil(t, piClient)
		require.NotNil(t, piClient.client)

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

		_, credCache := newPIClientWithSTSClient(newCfg(t), "arn:aws:iam::123456789012:role/TestRole", "my-external-id", mockSTS)

		_, err := credCache.Retrieve(context.TODO())
		require.NoError(t, err)
		assert.Equal(t, "my-external-id", mockSTS.CapturedExternalIDValue())
	})

	t.Run("ExternalID is not sent when empty", func(t *testing.T) {
		mockSTS := &testutils.MockSTSClient{}

		_, credCache := newPIClientWithSTSClient(newCfg(t), "arn:aws:iam::123456789012:role/TestRole", "", mockSTS)

		_, err := credCache.Retrieve(context.TODO())
		require.NoError(t, err)
		assert.Empty(t, mockSTS.CapturedExternalIDValue())
	})

	t.Run("AssumeRole error is propagated through credential retrieval", func(t *testing.T) {
		mockSTS := &testutils.MockSTSClient{ReturnErr: fmt.Errorf("AccessDenied: not authorized to assume role")}

		_, credCache := newPIClientWithSTSClient(newCfg(t), "arn:aws:iam::123456789012:role/TestRole", "", mockSTS)

		_, err := credCache.Retrieve(context.TODO())
		assert.Error(t, err)
		assert.True(t, mockSTS.Called())
		assert.Contains(t, err.Error(), "AccessDenied")
	})
}

