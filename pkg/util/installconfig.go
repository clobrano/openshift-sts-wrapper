package util

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// InstallConfig represents the minimal structure we need from install-config.yaml
type InstallConfig struct {
	BaseDomain string `yaml:"baseDomain"`
	SSHKey     string `yaml:"sshKey"`
	PullSecret string `yaml:"pullSecret"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Platform struct {
		AWS struct {
			Region string `yaml:"region"`
		} `yaml:"aws"`
	} `yaml:"platform"`
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

// ExtractedConfig contains all fields extracted from install-config.yaml
type ExtractedConfig struct {
	ClusterName string
	AwsRegion   string
	BaseDomain  string
	SSHKey      string
	PullSecret  string
}

// ExtractAllFields reads install-config.yaml and returns all relevant fields
func ExtractAllFields(installConfigPath string) (*ExtractedConfig, error) {
	config, err := ReadInstallConfig(installConfigPath)
	if err != nil {
		return nil, err
	}

	return &ExtractedConfig{
		ClusterName: config.Metadata.Name,
		AwsRegion:   config.Platform.AWS.Region,
		BaseDomain:  config.BaseDomain,
		SSHKey:      config.SSHKey,
		PullSecret:  config.PullSecret,
	}, nil
}

// GenerateInstallConfig generates a complete install-config.yaml file from provided values
func GenerateInstallConfig(path string, clusterName, baseDomain, awsRegion, sshKey, pullSecret, instanceType string) error {
	// Use default instance type if not specified
	if instanceType == "" {
		instanceType = "m5.4xlarge"
	}

	installConfig := map[string]interface{}{
		"additionalTrustBundlePolicy": "Proxyonly",
		"apiVersion":                  "v1",
		"baseDomain":                  baseDomain,
		"compute": []interface{}{
			map[string]interface{}{
				"architecture":   "amd64",
				"hyperthreading": "Enabled",
				"name":           "worker",
				"platform": map[string]interface{}{
					"aws": map[string]interface{}{
						"type": instanceType,
					},
				},
				"replicas": 3,
			},
		},
		"controlPlane": map[string]interface{}{
			"architecture":   "amd64",
			"hyperthreading": "Enabled",
			"name":           "master",
			"platform": map[string]interface{}{
				"aws": map[string]interface{}{
					"type": instanceType,
				},
			},
			"replicas": 3,
		},
		"metadata": map[string]interface{}{
			"creationTimestamp": nil,
			"name":              clusterName,
		},
		"networking": map[string]interface{}{
			"clusterNetwork": []interface{}{
				map[string]interface{}{
					"cidr":       "10.128.0.0/14",
					"hostPrefix": 23,
				},
			},
			"machineNetwork": []interface{}{
				map[string]interface{}{
					"cidr": "10.0.0.0/16",
				},
			},
			"networkType": "OVNKubernetes",
			"serviceNetwork": []interface{}{
				"172.30.0.0/16",
			},
		},
		"platform": map[string]interface{}{
			"aws": map[string]interface{}{
				"region": awsRegion,
				"vpc":    map[string]interface{}{},
			},
		},
		"publish":    "External",
		"pullSecret": pullSecret,
		"sshKey":     sshKey,
	}

	data, err := yaml.Marshal(installConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal install-config: %w", err)
	}

	// Post-process to format SSH key with literal block scalar (|)
	// The YAML library outputs: sshKey: <key content>
	// We want: sshKey: |\n    <key content>
	yamlStr := string(data)
	yamlStr = strings.Replace(yamlStr, "sshKey: "+sshKey, "sshKey: |\n    "+sshKey, 1)

	if err := os.WriteFile(path, []byte(yamlStr), 0644); err != nil {
		return fmt.Errorf("failed to write install-config.yaml: %w", err)
	}

	return nil
}
