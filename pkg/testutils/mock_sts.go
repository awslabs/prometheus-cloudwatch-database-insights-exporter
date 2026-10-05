package testutils

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	stypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
)

// MockSTSClient implements stscreds.AssumeRoleAPIClient for testing.
// Set ReturnErr to simulate AssumeRole failures.
var _ stscreds.AssumeRoleAPIClient = (*MockSTSClient)(nil)

type MockSTSClient struct {
	mu                      sync.Mutex
	wasCalled               bool
	capturedRoleArn         string
	capturedExternalID      string
	capturedRoleSessionName string
	ReturnErr               error
}

func (m *MockSTSClient) AssumeRole(_ context.Context, params *sts.AssumeRoleInput, _ ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wasCalled = true
	if params.RoleArn != nil {
		m.capturedRoleArn = *params.RoleArn
	}
	if params.ExternalId != nil {
		m.capturedExternalID = *params.ExternalId
	}
	if params.RoleSessionName != nil {
		m.capturedRoleSessionName = *params.RoleSessionName
	}
	if m.ReturnErr != nil {
		return nil, m.ReturnErr
	}
	return &sts.AssumeRoleOutput{
		Credentials: &stypes.Credentials{
			AccessKeyId:     aws.String("TEST-ACCESS-KEY-ID-0"),
			SecretAccessKey: aws.String("TEST-SECRET-KEY-NOT-REAL-000000000000"),
			SessionToken:    aws.String("session-token"),
			Expiration:      aws.Time(time.Now().Add(time.Hour)),
		},
	}, nil
}

func (m *MockSTSClient) Called() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.wasCalled
}

func (m *MockSTSClient) CapturedRoleArnValue() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.capturedRoleArn
}

func (m *MockSTSClient) CapturedExternalIDValue() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.capturedExternalID
}

func (m *MockSTSClient) CapturedRoleSessionNameValue() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.capturedRoleSessionName
}
