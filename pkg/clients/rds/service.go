package rds

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

type RDSService interface {
	DescribeDBInstancesPaginator(ctx context.Context) ([]types.DBInstance, error)
}
