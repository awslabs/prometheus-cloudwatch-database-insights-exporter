package utils

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
)

// NewAssumeRoleCredCache builds a CredentialsCache that assumes roleARN via stsClient.
// externalID is forwarded only when non-empty. sessionName is used for CloudTrail traceability.
func NewAssumeRoleCredCache(stsClient stscreds.AssumeRoleAPIClient, roleARN, externalID, sessionName string) *aws.CredentialsCache {
	creds := stscreds.NewAssumeRoleProvider(stsClient, roleARN, func(o *stscreds.AssumeRoleOptions) {
		o.RoleSessionName = sessionName
		if externalID != "" {
			o.ExternalID = aws.String(externalID)
		}
	})
	return aws.NewCredentialsCache(creds)
}
