package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// InstallConfig represents the minimal structure we need from install-config.yaml
type InstallConfig struct {
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Platform struct {
		AWS struct {
			Region string `yaml:"region"`
		} `yaml:"aws"`
	} `yaml:"platform"`
	SSHKey     string `yaml:"sshKey"`
	PullSecret string `yaml:"pullSecret"`
}

// ReadInstallConfig reads and parses install-config.yaml
func ReadInstallConfig(path string) (*InstallConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read install-config.yaml: %w", err)
	}

	var config InstallConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse install-config.yaml: %w", err)
	}

	return &config, nil
}

// ExtractClusterNameAndRegion reads install-config.yaml and returns the cluster name and region
func ExtractClusterNameAndRegion(installConfigPath string) (clusterName string, region string, err error) {
	config, err := ReadInstallConfig(installConfigPath)
	if err != nil {
		return "", "", err
	}

	if config.Metadata.Name == "" {
		return "", "", fmt.Errorf("cluster name not found in install-config.yaml")
	}

	if config.Platform.AWS.Region == "" {
		return "", "", fmt.Errorf("AWS region not found in install-config.yaml")
	}

	return config.Metadata.Name, config.Platform.AWS.Region, nil
}

// CopyInstallConfig copies an existing install-config.yaml to the target location
func CopyInstallConfig(sourcePath, targetPath string) error {
	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := EnsureDir(targetDir); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source install-config: %w", err)
	}
	defer sourceFile.Close()

	// Create target file
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target install-config: %w", err)
	}
	defer targetFile.Close()

	// Copy file contents
	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy install-config: %w", err)
	}

	return nil
}

// UpdateInstallConfig updates an install-config.yaml with new values
func UpdateInstallConfig(configPath string, clusterName, region, sshKey, pullSecret string) error {
	// Read existing config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read install-config.yaml: %w", err)
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return fmt.Errorf("failed to parse install-config.yaml: %w", err)
	}

	// Update cluster name if provided
	if clusterName != "" {
		if metadata, ok := doc["metadata"].(map[string]interface{}); ok {
			metadata["name"] = clusterName
		} else {
			doc["metadata"] = map[string]interface{}{
				"name": clusterName,
			}
		}
	}

	// Update region if provided
	if region != "" {
		if platform, ok := doc["platform"].(map[string]interface{}); ok {
			if aws, ok := platform["aws"].(map[string]interface{}); ok {
				aws["region"] = region
			} else {
				platform["aws"] = map[string]interface{}{
					"region": region,
				}
			}
		} else {
			doc["platform"] = map[string]interface{}{
				"aws": map[string]interface{}{
					"region": region,
				},
			}
		}
	}

	// Update SSH key if provided
	if sshKey != "" {
		doc["sshKey"] = sshKey
	}

	// Update pull secret if provided
	if pullSecret != "" {
		doc["pullSecret"] = pullSecret
	}

	// Marshal back to YAML
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to serialize install-config.yaml: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(configPath, out, 0644); err != nil {
		return fmt.Errorf("failed to write install-config.yaml: %w", err)
	}

	return nil
}
