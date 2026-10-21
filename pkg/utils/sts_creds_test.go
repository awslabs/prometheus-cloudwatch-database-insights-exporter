package utils

import (
	"context"
	"testing"
	"time"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAssumeRoleCredCache(t *testing.T) {
	const roleARN = "arn:aws:iam::123456789012:role/TestRole"

	t.Run("requests one-hour sessions", func(t *testing.T) {
		mockSTS := &testutils.MockSTSClient{}
		credCache := NewAssumeRoleCredCache(mockSTS, roleARN, "", AssumeRoleSessionName)

		_, err := credCache.Retrieve(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int32(3600), mockSTS.CapturedDurationSecondsValue())
	})

	t.Run("credentials outside the expiry window are reused", func(t *testing.T) {
		mockSTS := &testutils.MockSTSClient{CredentialsTTL: time.Hour}
		credCache := NewAssumeRoleCredCache(mockSTS, roleARN, "", AssumeRoleSessionName)

		for i := 0; i < 3; i++ {
			_, err := credCache.Retrieve(context.Background())
			require.NoError(t, err)
		}
		assert.Equal(t, 1, mockSTS.CallCount())
	})

	t.Run("credentials inside the expiry window are refreshed early", func(t *testing.T) {
		// Valid for 4 more minutes, inside the 5-minute window, so the cache treats them
		// as expired and assumes the role again rather than handing out soon-to-expire keys.
		mockSTS := &testutils.MockSTSClient{CredentialsTTL: 4 * time.Minute}
		credCache := NewAssumeRoleCredCache(mockSTS, roleARN, "", AssumeRoleSessionName)

		_, err := credCache.Retrieve(context.Background())
		require.NoError(t, err)
		_, err = credCache.Retrieve(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 2, mockSTS.CallCount())
	})

	t.Run("ExternalID is omitted when empty", func(t *testing.T) {
		mockSTS := &testutils.MockSTSClient{}
		credCache := NewAssumeRoleCredCache(mockSTS, roleARN, "", AssumeRoleSessionName)

		_, err := credCache.Retrieve(context.Background())
		require.NoError(t, err)
		assert.False(t, mockSTS.ExternalIDWasSent())
	})
}
