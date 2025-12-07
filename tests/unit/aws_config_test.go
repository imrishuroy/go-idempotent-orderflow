package unit

import (
	"context"
	"os"
	"testing"

	internalaws "github.com/imrishuroy/go-idempotent-orderflow/internal/aws"
)

func TestLoadAWSConfig_DefaultRegion(t *testing.T) {
	os.Unsetenv("AWS_ENDPOINT_OVERRIDE")
	os.Setenv("AWS_REGION", "")

	cfg, err := internalaws.LoadAWSConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Region != "us-east-1" {
		t.Fatalf("expected default region 'us-east-1', got %s", cfg.Region)
	}
}

func TestLoadAWSConfig_WithEndpointOverride(t *testing.T) {
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_ENDPOINT_OVERRIDE", "http://localhost:4566")

	cfg, err := internalaws.LoadAWSConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// we can't guarantee exact endpoints here, but we can ensure no error.
	if cfg.Region != "us-east-1" {
		t.Fatalf("region mismatch, got %s", cfg.Region)
	}
}
