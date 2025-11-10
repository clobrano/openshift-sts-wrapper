package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ReleaseImage    string `yaml:"releaseImage"`
	ClusterName     string `yaml:"-"` // Not loaded from config file - must be provided via CLI flag
	AwsRegion       string `yaml:"awsRegion"`
	BaseDomain      string `yaml:"baseDomain"`
	SSHKeyPath      string `yaml:"sshKeyPath,omitempty"`
	AwsProfile      string `yaml:"awsProfile"`
	PullSecretPath  string `yaml:"pullSecretPath"`
	PrivateBucket   bool   `yaml:"privateBucket"`
	StartFromStep   int    `yaml:"startFromStep"`
	ConfirmEachStep bool   `yaml:"confirmEachStep"`
	InstanceType    string `yaml:"instanceType"`
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	return &Config{
		ReleaseImage: os.Getenv("OPENSHIFT_STS_RELEASE_IMAGE"),
		// ClusterName is not loaded from env - must be provided via CLI flag
		AwsRegion:       os.Getenv("OPENSHIFT_STS_AWS_REGION"),
		BaseDomain:      os.Getenv("OPENSHIFT_STS_BASE_DOMAIN"),
		SSHKeyPath:      os.Getenv("OPENSHIFT_STS_SSH_KEY_PATH"),
		AwsProfile:      os.Getenv("OPENSHIFT_STS_AWS_PROFILE"),
		PullSecretPath:  os.Getenv("OPENSHIFT_STS_PULL_SECRET_PATH"),
		PrivateBucket:   os.Getenv("OPENSHIFT_STS_PRIVATE_BUCKET") == "true",
		ConfirmEachStep: os.Getenv("OPENSHIFT_STS_CONFIRM_EACH_STEP") == "true",
		InstanceType:    os.Getenv("OPENSHIFT_STS_INSTANCE_TYPE"),
	}
}

// Merge merges another config into this one, with the other config taking precedence
func (c *Config) Merge(other *Config) {
	if other.ReleaseImage != "" {
		c.ReleaseImage = other.ReleaseImage
	}
	// ClusterName is explicitly set from CLI flag, not merged from config sources
	if other.ClusterName != "" {
		c.ClusterName = other.ClusterName
	}
	if other.AwsRegion != "" {
		c.AwsRegion = other.AwsRegion
	}
	if other.BaseDomain != "" {
		c.BaseDomain = other.BaseDomain
	}
	if other.SSHKeyPath != "" {
		c.SSHKeyPath = other.SSHKeyPath
	}
	if other.AwsProfile != "" {
		c.AwsProfile = other.AwsProfile
	}
	if other.PullSecretPath != "" {
		c.PullSecretPath = other.PullSecretPath
	}
	if other.PrivateBucket {
		c.PrivateBucket = other.PrivateBucket
	}
	if other.StartFromStep > 0 {
		c.StartFromStep = other.StartFromStep
	}
	if other.ConfirmEachStep {
		c.ConfirmEachStep = other.ConfirmEachStep
	}
	if other.InstanceType != "" {
		c.InstanceType = other.InstanceType
	}
}

// ValidateConfig validates that required fields are set
func ValidateConfig(cfg *Config) error {
	if cfg.ReleaseImage == "" {
		return fmt.Errorf("release image is required")
	}
	if cfg.ClusterName == "" {
		return fmt.Errorf("cluster name is required (use --cluster-name flag)")
	}
	// AwsRegion is optional - can be read from install-config.yaml
	return nil
}

// SetDefaults sets default values for optional fields
func (c *Config) SetDefaults() {
	if c.PullSecretPath == "" {
		c.PullSecretPath = "pull-secret.json"
	}
	if c.AwsProfile == "" {
		c.AwsProfile = "default"
	}
	if c.InstanceType == "" {
		c.InstanceType = "m5.4xlarge"
	}
}

// SaveToFile saves configuration to a YAML file
func SaveToFile(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// HasCompleteInstallConfigData checks if config has all required fields for install-config.yaml
func (c *Config) HasCompleteInstallConfigData() (bool, []string) {
	var missing []string

	if c.ClusterName == "" {
		missing = append(missing, "clusterName")
	}
	if c.AwsRegion == "" {
		missing = append(missing, "awsRegion")
	}
	if c.BaseDomain == "" {
		missing = append(missing, "baseDomain")
	}
	if c.SSHKeyPath == "" {
		missing = append(missing, "sshKeyPath")
	}
	if c.PullSecretPath == "" {
		missing = append(missing, "pullSecretPath")
	}

	return len(missing) == 0, missing
}
