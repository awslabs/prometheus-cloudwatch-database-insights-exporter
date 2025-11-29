package instance

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/clients/rds"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/models"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/utils"
)

const (
	MaxRetries          = 3
	BaseDelay           = time.Second
	InstanceTTL         = 5 * time.Minute
	MetricsTTL          = 60 * time.Minute
	ValidInstanceStatus = "available"
)

type RDSInstanceManager struct {
	rdsService           rds.RDSService
	Instances            []models.Instance
	InstancesLastUpdated time.Time
	InstanceTTL          time.Duration
	configuration        *models.ParsedConfig
}

type SafeInstanceFields struct {
	Engine                     string
	DBInstanceStatus           string
	PerformanceInsightsEnabled bool
	DbiResourceId              string
	DBInstanceIdentifier       string
	InstanceCreateTime         time.Time
}

// RDSInstanceManager handles discovery and caching of RDS database instances within a region.
// It provides instance discovery with TTL-based caching to minimize AWS API calls while ensuring data freshness for metric collection operations.
func NewRDSInstanceManager(rds rds.RDSService, config *models.ParsedConfig) (*RDSInstanceManager, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration parameter cannot be nil")
	}
	return &RDSInstanceManager{
		rdsService:    rds,
		InstanceTTL:   config.Discovery.Instances.InstanceTTL,
		configuration: config,
	}, nil
}

// GetInstances returns cached database instances, refreshing from AWS if TTL is expired.
func (instanceManager *RDSInstanceManager) GetInstances(ctx context.Context) ([]models.Instance, error) {
	if instanceManager.configuration == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	if instanceManager.Instances == nil || instanceManager.InstancesLastUpdated.IsZero() || time.Now().After(instanceManager.InstancesLastUpdated.Add(instanceManager.InstanceTTL)) {
		instances, err := instanceManager.discoverInstances(ctx)
		if err != nil {
			return nil, err
		}
		log.Printf("[INSTANCE] Discovered %d instances ", len(instances))

		maxInstances := instanceManager.configuration.Discovery.Instances.MaxInstances
		if len(instances) > maxInstances {
			instanceManager.Instances = instances[:maxInstances]
			log.Printf("[INSTANCE] Limited to %d instances ", len(instanceManager.Instances))
		} else {
			instanceManager.Instances = instances
		}
		instanceManager.InstancesLastUpdated = time.Now()
	}

	return instanceManager.Instances, nil
}

func (instanceManager *RDSInstanceManager) discoverInstances(ctx context.Context) ([]models.Instance, error) {
	discoveredInstances, err := utils.WithRetry(ctx, func() ([]types.DBInstance, error) {
		return instanceManager.rdsService.DescribeDBInstancesPaginator(ctx)
	}, MaxRetries, BaseDelay)
	if err != nil {
		log.Printf("[INSTANCE] Error discovering instances: %v", err)
		return nil, err
	}

	var instances []models.Instance
	for _, instance := range discoveredInstances {
		instanceFields, err := safeExtractInstanceFields(instance)
		if err != nil {
			log.Printf("[INSTANCE] Error extracting instance fields: %v", err)
			continue
		}

		var instance models.Instance
		engine := models.NewEngine(instanceFields.Engine)
		if instanceFields.PerformanceInsightsEnabled && engine != "" {
			instance = models.Instance{
				ResourceID:   instanceFields.DbiResourceId,
				Identifier:   instanceFields.DBInstanceIdentifier,
				Engine:       engine,
				CreationTime: instanceFields.InstanceCreateTime,
				Metrics: &models.Metrics{
					MetadataTTL: instanceManager.configuration.Discovery.Metrics.MetadataTTL,
				},
			}
		}

		instanceConfig := instanceManager.configuration.Discovery.Instances
		if !instanceConfig.ShouldIncludeInstance(instance) {
			continue
		}

		if instance.ResourceID == "" || instance.Identifier == "" {
			continue
		}

		instances = append(instances, instance)
	}

	sort.Slice(instances, func(i, j int) bool {
		return instances[i].CreationTime.Before(instances[j].CreationTime)
	})

	return instances, nil
}

func safeExtractInstanceFields(instance types.DBInstance) (*SafeInstanceFields, error) {
	fields := &SafeInstanceFields{}

	if instance.Engine == nil {
		return nil, fmt.Errorf("instance.Engine is nil for instance")
	}
	fields.Engine = *instance.Engine

	if instance.DBInstanceStatus == nil {
		return nil, fmt.Errorf("instance.DBInstanceStatus is nil for instance")
	}
	fields.DBInstanceStatus = *instance.DBInstanceStatus

	if instance.DbiResourceId == nil {
		return nil, fmt.Errorf("instance.DbiResourceId is nil for instance")
	}
	fields.DbiResourceId = *instance.DbiResourceId

	if instance.DBInstanceIdentifier == nil {
		return nil, fmt.Errorf("instance.DBInstanceIdentifier is nil for instance")
	}
	fields.DBInstanceIdentifier = *instance.DBInstanceIdentifier

	if instance.PerformanceInsightsEnabled != nil {
		fields.PerformanceInsightsEnabled = *instance.PerformanceInsightsEnabled
	} else {
		fields.PerformanceInsightsEnabled = false
	}

	if instance.InstanceCreateTime == nil {
		return nil, fmt.Errorf("instance.InstanceCreateTime is nil for instance")
	}

	if instance.InstanceCreateTime.IsZero() {
		return nil, fmt.Errorf("instance.InstanceCreateTime is zero for instance")
	}
	fields.InstanceCreateTime = *instance.InstanceCreateTime

	return fields, nil
}
