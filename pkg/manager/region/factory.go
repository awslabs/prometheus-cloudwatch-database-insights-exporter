package region

import (
	"context"
	"fmt"
	"log"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/clients/pi"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/clients/rds"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/manager/instance"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/manager/metric"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/utils"
)

// RegionManagerFactory creates and configures region managers for database insights collection.
// It implements the factory design pattern to encapsulate the initialization logic required to set up AWS service clients,
// instance discovery, and metric collection components.
type RegionManagerFactory struct {
}

func NewRegionManagerFactory() *RegionManagerFactory {
	return &RegionManagerFactory{}
}

// CreateRegionManager creates a multi-region manager to coordinate across configured regions.
func (factory *RegionManagerFactory) CreateRegionManager(config *models.ParsedConfig) (RegionManager, error) {
	multiRegionManager := NewMultiRegionManager()
	regions := config.Discovery.Regions
	for _, region := range regions {
		singleRegionManager, err := factory.createSingleRegionManager(region, config)
		if err != nil {
			return nil, err
		}

		multiRegionManager.AddRegionManager(region, singleRegionManager)
	}
	return multiRegionManager, nil
}

func (factory *RegionManagerFactory) createSingleRegionManager(region string, config *models.ParsedConfig) (RegionManager, error) {
	roleARN := config.Discovery.RoleARN
	externalID := config.Discovery.RoleARNExternalID

	var rdsClient *rds.RDSClient
	var piClient *pi.PIClient

	if roleARN != "" {
		cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(region))
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config for role assumption: %w", err)
		}
		stsClient := sts.NewFromConfig(cfg)
		cfg.Credentials = utils.NewAssumeRoleCredCache(stsClient, roleARN, externalID, "rds-pi-exporter")
		log.Printf("[FACTORY] Cross-account role configured: %s, region: %s", roleARN, region)
		rdsClient = rds.NewRDSClientFromConfig(cfg)
		piClient = pi.NewPIClientFromConfig(cfg)
	} else {
		var err error
		rdsClient, err = rds.NewRDSClient(region)
		if err != nil {
			return nil, err
		}
		piClient, err = pi.NewPIClient(region)
		if err != nil {
			return nil, err
		}
	}

	rdsInstanceManager, err := instance.NewRDSInstanceManager(rdsClient, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS instance manager: %w", err)
	}

	metricManager, err := metric.NewMetricManager(piClient, config, region)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric manager: %w", err)
	}

	return NewSingleRegionManager(region, rdsInstanceManager, metricManager, config.Discovery.Processing.Concurrency), nil
}
