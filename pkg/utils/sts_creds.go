package utils

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
)

const (
	// AssumeRoleSessionName is the RoleSessionName sent on sts:AssumeRole, visible in CloudTrail.
	AssumeRoleSessionName = "prometheus-cloudwatch-database-insights-exporter"
	// AssumeRoleDuration is the requested session length. One hour is the default role
	// maximum and the cap when chaining from IRSA; the SDK default is only 15 minutes.
	AssumeRoleDuration = time.Hour
	// AssumeRoleExpiryWindow refreshes credentials this long before they expire, so a
	// request signed just before expiry does not fail with ExpiredTokenException.
	AssumeRoleExpiryWindow = 5 * time.Minute
)

// NewAssumeRoleCredCache builds a CredentialsCache that assumes roleARN via stsClient.
// externalID is forwarded only when non-empty. sessionName is used for CloudTrail traceability.
func NewAssumeRoleCredCache(stsClient stscreds.AssumeRoleAPIClient, roleARN, externalID, sessionName string) *aws.CredentialsCache {
	creds := stscreds.NewAssumeRoleProvider(stsClient, roleARN, func(o *stscreds.AssumeRoleOptions) {
		o.RoleSessionName = sessionName
		o.Duration = AssumeRoleDuration
		if externalID != "" {
			o.ExternalID = aws.String(externalID)
		}
	})
	return aws.NewCredentialsCache(creds, func(o *aws.CredentialsCacheOptions) {
		o.ExpiryWindow = AssumeRoleExpiryWindow
	})
}
