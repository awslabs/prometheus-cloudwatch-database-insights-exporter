package mocks

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pi"
	pitypes "github.com/aws/aws-sdk-go-v2/service/pi/types"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/stretchr/testify/mock"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/testutils"
)

type MockRDSService struct {
	mock.Mock
}

func (mockRDSService *MockRDSService) DescribeDBInstancesPaginator(ctx context.Context) ([]rdstypes.DBInstance, error) {
	args := mockRDSService.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]rdstypes.DBInstance), args.Error(1)
}

// NewMockRDSDescribeInstances returns a slice of DBInstances for pagination testing
func NewMockRDSDescribeInstances() []rdstypes.DBInstance {
	return []rdstypes.DBInstance{
		{
			DBInstanceIdentifier:       aws.String("test-postgres-db"),
			DBInstanceArn:              aws.String("arn:aws:rds:us-west-2:123456789012:db:test-postgres-db"),
			InstanceCreateTime:         aws.Time(testutils.TestInstanceCreationTimePostgreSQL),
			DbiResourceId:              aws.String("db-TESTPOSTGRES"),
			Engine:                     aws.String("aurora-postgresql"),
			DBInstanceStatus:           aws.String("available"),
			DBInstanceClass:            aws.String("db.t3.micro"),
			AllocatedStorage:           aws.Int32(20),
			PerformanceInsightsEnabled: aws.Bool(true),
			TagList: []rdstypes.Tag{
				{Key: aws.String("Environment"), Value: aws.String("test")},
				{Key: aws.String("Team"), Value: aws.String("platform")},
			},
		},
		{
			DBInstanceIdentifier:       aws.String("test-mysql-db"),
			DBInstanceArn:              aws.String("arn:aws:rds:us-west-2:123456789012:db:test-mysql-db"),
			InstanceCreateTime:         aws.Time(testutils.TestInstanceCreationTimeMySQL),
			DbiResourceId:              aws.String("db-TESTMYSQL"),
			Engine:                     aws.String("aurora-mysql"),
			DBInstanceStatus:           aws.String("available"),
			DBInstanceClass:            aws.String("db.t3.small"),
			AllocatedStorage:           aws.Int32(50),
			PerformanceInsightsEnabled: aws.Bool(true),
			TagList: []rdstypes.Tag{
				{Key: aws.String("Environment"), Value: aws.String("production")},
				{Key: aws.String("Team"), Value: aws.String("data")},
			},
		},
	}
}

func NewMockRDSDescribeInstancesEmpty() []rdstypes.DBInstance {
	return []rdstypes.DBInstance{}
}

func NewMockRDSDescribeInstancesSingle() []rdstypes.DBInstance {
	return []rdstypes.DBInstance{
		{
			DBInstanceIdentifier:       aws.String("test-postgres-db"),
			DBInstanceArn:              aws.String("arn:aws:rds:us-west-2:123456789012:db:test-postgres-db"),
			InstanceCreateTime:         aws.Time(testutils.TestInstanceCreationTimePostgreSQL),
			DbiResourceId:              aws.String("db-TESTPOSTGRES"),
			Engine:                     aws.String("aurora-postgresql"),
			DBInstanceStatus:           aws.String("available"),
			DBInstanceClass:            aws.String("db.t3.micro"),
			AllocatedStorage:           aws.Int32(20),
			PerformanceInsightsEnabled: aws.Bool(true),
			TagList: []rdstypes.Tag{
				{Key: aws.String("Environment"), Value: aws.String("test")},
			},
		},
	}
}

type MockPIService struct {
	mock.Mock
}

func (mockPIService *MockPIService) ListAvailableResourceMetrics(ctx context.Context, resourceID string) (*pi.ListAvailableResourceMetricsOutput, error) {
	args := mockPIService.Called(ctx, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pi.ListAvailableResourceMetricsOutput), args.Error(1)
}

func (mockPIService *MockPIService) GetResourceMetrics(ctx context.Context, resourceID string, metricNames []string) (*pi.GetResourceMetricsOutput, error) {
	args := mockPIService.Called(ctx, resourceID, metricNames)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pi.GetResourceMetricsOutput), args.Error(1)
}

func NewMockPIListMetricsResponse() *pi.ListAvailableResourceMetricsOutput {
	return &pi.ListAvailableResourceMetricsOutput{
		Metrics: []pitypes.ResponseResourceMetric{
			{
				Metric:      aws.String("os.general.numVCPUs"),
				Description: aws.String("The number of virtual CPUs for the DB instance"),
				Unit:        aws.String("vCPUs"),
			},
			{
				Metric:      aws.String("os.cpuUtilization.guest"),
				Description: aws.String("The percentage of CPU in use by guest programs"),
				Unit:        aws.String("Percent"),
			},
			{
				Metric:      aws.String("os.cpuUtilization.idle"),
				Description: aws.String("The percentage of CPU that is idle"),
				Unit:        aws.String("Percent"),
			},
			{
				Metric:      aws.String("os.memory.total"),
				Description: aws.String("The total amount of memory in kilobytes"),
				Unit:        aws.String("KB"),
			},
			{
				Metric:      aws.String("db.User.max_connections"),
				Description: aws.String("The maximum number of connections allowed for a DB instance as configured in max_connections parameter"),
				Unit:        aws.String("Connections"),
			},
		},
	}
}

func NewMockPIListMetricsResponseEmpty() *pi.ListAvailableResourceMetricsOutput {
	return &pi.ListAvailableResourceMetricsOutput{
		Metrics: []pitypes.ResponseResourceMetric{},
	}
}

func NewMockPIGetResourceMetricsResponse() *pi.GetResourceMetricsOutput {
	return &pi.GetResourceMetricsOutput{
		MetricList: []pitypes.MetricKeyDataPoints{
			{
				Key: &pitypes.ResponseResourceMetricKey{
					Metric: aws.String("os.general.numVCPUs.avg"),
				},
				DataPoints: []pitypes.DataPoint{
					{
						Timestamp: aws.Time(testutils.TestTimestamp),
						Value:     aws.Float64(4.0),
					},
				},
			},
			{
				Key: &pitypes.ResponseResourceMetricKey{
					Metric: aws.String("os.cpuUtilization.guest.avg"),
				},
				DataPoints: []pitypes.DataPoint{
					{
						Timestamp: aws.Time(testutils.TestTimestamp),
						Value:     aws.Float64(25.5),
					},
				},
			},
			{
				Key: &pitypes.ResponseResourceMetricKey{
					Metric: aws.String("os.cpuUtilization.idle.avg"),
				},
				DataPoints: []pitypes.DataPoint{
					{
						Timestamp: aws.Time(testutils.TestTimestamp),
						Value:     aws.Float64(74.5),
					},
				},
			},
			{
				Key: &pitypes.ResponseResourceMetricKey{
					Metric: aws.String("os.memory.total.avg"),
				},
				DataPoints: []pitypes.DataPoint{
					{
						Timestamp: aws.Time(testutils.TestTimestamp),
						Value:     aws.Float64(16.0),
					},
				},
			},
			{
				Key: &pitypes.ResponseResourceMetricKey{
					Metric: aws.String("db.User.max_connections.avg"),
				},
				DataPoints: []pitypes.DataPoint{
					{
						Timestamp: aws.Time(testutils.TestTimestamp),
						Value:     aws.Float64(2.0),
					},
				},
			},
		},
	}
}

func NewMockPIGetResourceMetricsResponseEmpty() *pi.GetResourceMetricsOutput {
	return &pi.GetResourceMetricsOutput{
		MetricList: []pitypes.MetricKeyDataPoints{},
	}
}

func NewMockPIGetResourceMetricsResponseWithNilKeys() *pi.GetResourceMetricsOutput {
	return &pi.GetResourceMetricsOutput{
		MetricList: []pitypes.MetricKeyDataPoints{
			{
				Key: nil,
				DataPoints: []pitypes.DataPoint{
					{
						Timestamp: aws.Time(testutils.TestTimestamp),
						Value:     aws.Float64(4.0),
					},
				},
			},
			{
				Key: &pitypes.ResponseResourceMetricKey{
					Metric: nil,
				},
				DataPoints: []pitypes.DataPoint{
					{
						Timestamp: aws.Time(testutils.TestTimestamp),
						Value:     aws.Float64(25.5),
					},
				},
			},
			{
				Key: &pitypes.ResponseResourceMetricKey{
					Metric: aws.String("os.cpuUtilization.idle.avg"),
				},
				DataPoints: []pitypes.DataPoint{
					{
						Timestamp: aws.Time(testutils.TestTimestamp),
						Value:     aws.Float64(74.5),
					},
				},
			},
		},
	}
}
