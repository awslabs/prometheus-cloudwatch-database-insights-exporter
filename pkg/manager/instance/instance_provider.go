package instance

import (
	"context"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
)

type InstanceProvider interface {
	GetInstances(ctx context.Context) ([]models.Instance, error)
}
