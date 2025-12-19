package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// AWSClients bundles all service clients for convenience.
type AWSClients struct {
	DynamoDB   DynamoDBAPI
	SQS        SQSAPI
	CloudWatch CloudWatchAPI
}

// NewAWSClients loads AWS config and returns concrete service clients that implement our interfaces.
func NewAWSClients(ctx context.Context) (*AWSClients, error) {
	cfg, err := LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &AWSClients{
		DynamoDB:   dynamodb.NewFromConfig(cfg),
		SQS:        sqs.NewFromConfig(cfg),
		CloudWatch: cloudwatch.NewFromConfig(cfg),
	}, nil
}
