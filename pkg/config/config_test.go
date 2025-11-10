package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "openshift-sts-installer.yaml")
	configContent := `releaseImage: quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64
awsRegion: us-east-2
pullSecretPath: ./pull-secret.json
privateBucket: true
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.ReleaseImage != "quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64" {
		t.Errorf("Expected ReleaseImage to be set, got %q", cfg.ReleaseImage)
	}
	// ClusterName should NOT be loaded from config file (yaml:"-" tag)
	if cfg.ClusterName != "" {
		t.Errorf("Expected ClusterName to be empty (not loaded from file), got %q", cfg.ClusterName)
	}
	if cfg.AwsRegion != "us-east-2" {
		t.Errorf("Expected AwsRegion to be 'us-east-2', got %q", cfg.AwsRegion)
	}
	if !cfg.PrivateBucket {
		t.Error("Expected PrivateBucket to be true")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	os.Setenv("OPENSHIFT_STS_RELEASE_IMAGE", "quay.io/test:4.11.0-x86_64")
	os.Setenv("OPENSHIFT_STS_AWS_REGION", "us-west-2")
	defer func() {
		os.Unsetenv("OPENSHIFT_STS_RELEASE_IMAGE")
		os.Unsetenv("OPENSHIFT_STS_AWS_REGION")
	}()

	cfg := LoadFromEnv()

	if cfg.ReleaseImage != "quay.io/test:4.11.0-x86_64" {
		t.Errorf("Expected ReleaseImage from env, got %q", cfg.ReleaseImage)
	}
	// ClusterName should NOT be loaded from env
	if cfg.ClusterName != "" {
		t.Errorf("Expected ClusterName to be empty (not loaded from env), got %q", cfg.ClusterName)
	}
	if cfg.AwsRegion != "us-west-2" {
		t.Errorf("Expected AwsRegion from env, got %q", cfg.AwsRegion)
	}
}

func TestConfigMerge(t *testing.T) {
	base := &Config{
		ReleaseImage: "base-image",
		ClusterName:  "base-cluster",
	}

	override := &Config{
		ClusterName: "override-cluster",
		AwsRegion:   "override-region",
	}

	base.Merge(override)

	if base.ReleaseImage != "base-image" {
		t.Error("Merge should not override non-empty base values when override is empty")
	}
	if base.ClusterName != "override-cluster" {
		t.Error("Merge should override base values")
	}
	if base.AwsRegion != "override-region" {
		t.Error("Merge should set empty base values")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		shouldError bool
	}{
		{
			name: "valid config",
			config: Config{
				ReleaseImage:   "quay.io/test:4.12.0-x86_64",
				ClusterName:    "test-cluster",
				AwsRegion:      "us-east-1",
				PullSecretPath: "pull-secret.json",
			},
			shouldError: false,
		},
		{
			name: "missing release image",
			config: Config{
				ClusterName:    "test-cluster",
				AwsRegion:      "us-east-1",
				PullSecretPath: "pull-secret.json",
			},
			shouldError: true,
		},
		{
			name: "missing cluster name is now an error",
			config: Config{
				ReleaseImage:   "quay.io/test:4.12.0-x86_64",
				AwsRegion:      "us-east-1",
				PullSecretPath: "pull-secret.json",
			},
			shouldError: true,
		},
		{
			name: "missing aws region is ok",
			config: Config{
				ReleaseImage:   "quay.io/test:4.12.0-x86_64",
				ClusterName:    "test-cluster",
				PullSecretPath: "pull-secret.json",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(&tt.config)
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
