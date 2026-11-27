package rds

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

type RDSClient struct {
	client *rds.Client
}

// AWS Relational Database Service (RDS) manages relational databases in the cloud.
// This client focuses on discovery instances for comprehensive database performance monitoring.

// RDSClient wraps the AWS RDS SDK with application-specific database discovery functionality.
// It provides methods for describing database instances.
func NewRDSClient(region string) (*RDSClient, error) {
	return NewRDSClientWithEndpoint(region, "")
}

// NewRDSClientFromConfig creates an RDS client from a pre-configured aws.Config.
// Use this when credentials (e.g. assumed-role) are set up externally, such as
// by the factory, so that multiple clients can share a single CredentialsCache.
func NewRDSClientFromConfig(cfg aws.Config) *RDSClient {
	return newRDSClientFromConfig(cfg, "")
}

func NewRDSClientWithEndpoint(region, endpoint string) (*RDSClient, error) {
	log.Println("[RDS] Creating new RDS client...")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Printf("[RDS] FATAL: Failed to load AWS config: %v", err)
		return nil, err
	}

	log.Printf("[RDS] AWS config loaded, region: %s", cfg.Region)
	return newRDSClientFromConfig(cfg, endpoint), nil
}

func newRDSClientFromConfig(cfg aws.Config, endpoint string) *RDSClient {
	client := rds.NewFromConfig(cfg)
	if endpoint != "" {
		client = rds.NewFromConfig(cfg, func(o *rds.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
		log.Printf("[RDS] Using custom endpoint: %s", endpoint)
	}
	return &RDSClient{client: client}
}

func (rdsClient *RDSClient) DescribeDBInstancesPaginator(ctx context.Context) ([]types.DBInstance, error) {
	input := &rds.DescribeDBInstancesInput{
		MaxRecords: aws.Int32(100),
	}

	var allInstances []types.DBInstance

	paginator := rds.NewDescribeDBInstancesPaginator(rdsClient.client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			log.Printf("[RDS] Failed to describe DB instances: %v", err)
			return nil, err
		}

		allInstances = append(allInstances, page.DBInstances...)
	}

	log.Printf("[RDS] Retrieved %d DB instances", len(allInstances))
	return allInstances, nil
}
