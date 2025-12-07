package aws

import (
	"context"
	"fmt"
	"os"

	sdkaws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func LoadAWSConfig(ctx context.Context) (sdkaws.Config, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1" // default fallback
	}

	var cfg sdkaws.Config
	var err error

	cfg, err = config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)

	if err != nil {
		return cfg, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return cfg, nil
}
