package pi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
)

func TestNewPIClient(t *testing.T) {
	t.Run("creates new PI client successfully", func(t *testing.T) {
		piClient, err := NewPIClient(testutils.TestRegion)
		assert.NoError(t, err)
		assert.NotNil(t, piClient)
		assert.NotNil(t, piClient.client)
	})
}
