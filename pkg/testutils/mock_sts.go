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
// Set ReturnErr to simulate AssumeRole failures. Set CredentialsTTL to control how long
// the returned credentials stay valid (default: one hour).
var _ stscreds.AssumeRoleAPIClient = (*MockSTSClient)(nil)

type MockSTSClient struct {
	mu                      sync.Mutex
	callCount               int
	capturedRoleArn         string
	capturedExternalID      string
	externalIDSent          bool
	capturedRoleSessionName string
	capturedDurationSeconds int32
	ReturnErr               error
	CredentialsTTL          time.Duration
}

func (m *MockSTSClient) AssumeRole(_ context.Context, params *sts.AssumeRoleInput, _ ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	if params.RoleArn != nil {
		m.capturedRoleArn = *params.RoleArn
	}
	m.externalIDSent = params.ExternalId != nil
	if params.ExternalId != nil {
		m.capturedExternalID = *params.ExternalId
	}
	if params.RoleSessionName != nil {
		m.capturedRoleSessionName = *params.RoleSessionName
	}
	if params.DurationSeconds != nil {
		m.capturedDurationSeconds = *params.DurationSeconds
	}
	if m.ReturnErr != nil {
		return nil, m.ReturnErr
	}
	ttl := m.CredentialsTTL
	if ttl == 0 {
		ttl = time.Hour
	}
	return &sts.AssumeRoleOutput{
		Credentials: &stypes.Credentials{
			AccessKeyId:     aws.String("TEST-ACCESS-KEY-ID-0"),
			SecretAccessKey: aws.String("TEST-SECRET-KEY-NOT-REAL-000000000000"),
			SessionToken:    aws.String("session-token"),
			Expiration:      aws.Time(time.Now().Add(ttl)),
		},
	}, nil
}

func (m *MockSTSClient) Called() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount > 0
}

func (m *MockSTSClient) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
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

// ExternalIDWasSent reports whether the last AssumeRole call set ExternalId at all,
// which CapturedExternalIDValue cannot distinguish from an empty value.
func (m *MockSTSClient) ExternalIDWasSent() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.externalIDSent
}

func (m *MockSTSClient) CapturedDurationSecondsValue() int32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.capturedDurationSeconds
}

func (m *MockSTSClient) CapturedRoleSessionNameValue() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.capturedRoleSessionName
}
