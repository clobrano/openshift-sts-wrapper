package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateInstallConfigCompatibility(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		yamlContent   string
		shouldError   bool
		errorContains string
	}{
		{
			name: "valid install-config",
			yamlContent: `apiVersion: v1
baseDomain: example.com
metadata:
  name: test-cluster
platform:
  aws:
    region: us-east-2
`,
			shouldError: false,
		},
		{
			name: "missing cluster name",
			yamlContent: `apiVersion: v1
baseDomain: example.com
metadata:
  name: ""
platform:
  aws:
    region: us-east-2
`,
			shouldError:   true,
			errorContains: "metadata.name",
		},
		{
			name: "missing aws region",
			yamlContent: `apiVersion: v1
baseDomain: example.com
metadata:
  name: test-cluster
platform:
  aws:
    region: ""
`,
			shouldError:   true,
			errorContains: "platform.aws.region",
		},
		{
			name:          "invalid YAML",
			yamlContent:   `invalid: [yaml: content: that: cannot: be: parsed`,
			shouldError:   true,
			errorContains: "failed to read install-config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary install-config.yaml file
			installConfigPath := filepath.Join(tmpDir, tt.name+"-install-config.yaml")
			err := os.WriteFile(installConfigPath, []byte(tt.yamlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test install-config.yaml: %v", err)
			}

			// Test validation
			err = ValidateInstallConfigCompatibility(installConfigPath)
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.shouldError && err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
			}
		})
	}
}

func TestValidateInstallConfigCompatibilityNonExistent(t *testing.T) {
	err := ValidateInstallConfigCompatibility("/nonexistent/install-config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error to mention file not found, got: %v", err)
	}
}

func TestExtractClusterNameAndRegion(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name            string
		yamlContent     string
		expectedCluster string
		expectedRegion  string
		shouldError     bool
	}{
		{
			name: "valid config",
			yamlContent: `apiVersion: v1
baseDomain: example.com
metadata:
  name: my-cluster
platform:
  aws:
    region: us-west-2
`,
			expectedCluster: "my-cluster",
			expectedRegion:  "us-west-2",
			shouldError:     false,
		},
		{
			name: "missing cluster name",
			yamlContent: `apiVersion: v1
baseDomain: example.com
metadata:
  name: ""
platform:
  aws:
    region: us-west-2
`,
			shouldError: true,
		},
		{
			name: "missing region",
			yamlContent: `apiVersion: v1
baseDomain: example.com
metadata:
  name: my-cluster
platform:
  aws:
    region: ""
`,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installConfigPath := filepath.Join(tmpDir, tt.name+"-install-config.yaml")
			err := os.WriteFile(installConfigPath, []byte(tt.yamlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test install-config.yaml: %v", err)
			}

			cluster, region, err := ExtractClusterNameAndRegion(installConfigPath)
			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.shouldError {
				if cluster != tt.expectedCluster {
					t.Errorf("Expected cluster name %q, got %q", tt.expectedCluster, cluster)
				}
				if region != tt.expectedRegion {
					t.Errorf("Expected region %q, got %q", tt.expectedRegion, region)
				}
			}
		})
	}
}
