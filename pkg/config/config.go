package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/logger"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/util"
)

type Config struct {
	ReleaseImage      string `yaml:"releaseImage"`
	ClusterName       string `yaml:"clusterName"`
	AwsRegion         string `yaml:"awsRegion"`
	AwsProfile        string `yaml:"awsProfile"`
	PullSecretPath    string `yaml:"pullSecretPath"`
	PrivateBucket     bool   `yaml:"privateBucket"`
	OutputDir         string `yaml:"outputDir"`
	StartFromStep     int    `yaml:"startFromStep"`
	ConfirmEachStep   bool   `yaml:"confirmEachStep"`
	InstanceType      string `yaml:"instanceType"`
	InstallConfigPath string `yaml:"installConfigPath"`
}

func LoadConfig(cfgFile string, log *logger.Logger) *Config {
	cfg := &Config{}

	// 1. Load from environment variables
	envCfg := LoadFromEnv()
	cfg.Merge(envCfg)

	// 2. Load from file
	configFile := cfgFile
	if configFile == "" {
		configFile = "openshift-sts-installer.yaml"
	}
	if util.FileExists(configFile) {
		fileCfg, err := LoadFromFile(configFile)
		if err != nil {
			log.Debug(fmt.Sprintf("Could not load config file: %v", err))
		} else {
			cfg.Merge(fileCfg)
		}
	}

	// 4. Set defaults
	cfg.SetDefaults()

	return cfg
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
		ReleaseImage:      os.Getenv("OPENSHIFT_STS_RELEASE_IMAGE"),
		ClusterName:       os.Getenv("OPENSHIFT_STS_CLUSTER_NAME"),
		AwsRegion:         os.Getenv("OPENSHIFT_STS_AWS_REGION"),
		AwsProfile:        os.Getenv("OPENSHIFT_STS_AWS_PROFILE"),
		PullSecretPath:    os.Getenv("OPENSHIFT_STS_PULL_SECRET_PATH"),
		PrivateBucket:     os.Getenv("OPENSHIFT_STS_PRIVATE_BUCKET") == "true",
		OutputDir:         os.Getenv("OPENSHIFT_STS_OUTPUT_DIR"),
		ConfirmEachStep:   os.Getenv("OPENSHIFT_STS_CONFIRM_EACH_STEP") == "true",
		InstanceType:      os.Getenv("OPENSHIFT_STS_INSTANCE_TYPE"),
		InstallConfigPath: os.Getenv("OPENSHIFT_STS_INSTALL_CONFIG_PATH"),
	}
}

// Merge merges another config into this one, with the other config taking precedence
func (c *Config) Merge(other *Config) {
	if other.ReleaseImage != "" {
		c.ReleaseImage = other.ReleaseImage
	}
	if other.ClusterName != "" {
		c.ClusterName = other.ClusterName
	}
	if other.AwsRegion != "" {
		c.AwsRegion = other.AwsRegion
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
	if other.OutputDir != "" {
		c.OutputDir = other.OutputDir
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
	if other.InstallConfigPath != "" {
		c.InstallConfigPath = other.InstallConfigPath
	}
}

// ValidateConfig validates that required fields are set
func ValidateConfig(cfg *Config) error {
	if cfg.ReleaseImage == "" {
		return fmt.Errorf("release image is required")
	}
	// ClusterName and AwsRegion are now optional - they can be read from install-config.yaml
	return nil
}

// SetDefaults sets default values for optional fields
func (c *Config) SetDefaults() {
	if c.OutputDir == "" {
		c.OutputDir = "_output"
	}
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
