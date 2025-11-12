package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/config"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/errors"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/logger"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/steps"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/util"
)

var (
	releaseImage    string
	clusterName     string
	awsProfile      string
	pullSecretPath  string
	privateBucket   bool
	startFromStep   int
	confirmEachStep bool
	instanceType    string
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install OpenShift cluster with STS",
	Long:  `Executes the full OpenShift STS installation workflow`,
	Run:   runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().StringVar(&releaseImage, "release-image", "", "OpenShift release image URL (required)")
	installCmd.Flags().StringVar(&clusterName, "cluster-name", "", "Cluster name (required)")
	installCmd.Flags().StringVar(&awsProfile, "aws-profile", "", "AWS profile name (default: default)")
	installCmd.Flags().StringVar(&pullSecretPath, "pull-secret", "", "Path to pull secret file")
	installCmd.Flags().BoolVar(&privateBucket, "private-bucket", false, "Use private S3 bucket with CloudFront")
	installCmd.Flags().IntVar(&startFromStep, "start-from-step", 0, "Start from specific step number")
	installCmd.Flags().BoolVar(&confirmEachStep, "confirm-each-step", false, "Prompt for confirmation before executing each step")
	installCmd.Flags().StringVar(&instanceType, "instance-type", "m5.4xlarge", "AWS instance type for controlPlane and compute pools")
}

func runInstall(cmd *cobra.Command, args []string) {
	// Create logger
	log := logger.New(logger.Level(getLogLevel()), nil)

	// Check prerequisites
	if err := config.CheckPrerequisites(); err != nil {
		log.Error(fmt.Sprintf("Prerequisite check failed: %v", err))
		os.Exit(1)
	}

	// Load configuration with priority: flags > file > env > prompts
	cfg := loadConfig(log)

	// Validate configuration
	if err := config.ValidateConfig(cfg); err != nil {
		log.Error(fmt.Sprintf("Configuration error: %v", err))
		os.Exit(1)
	}

	// Validate AWS credentials
	log.Info(fmt.Sprintf("Validating AWS credentials for profile '%s'...", cfg.AwsProfile))
	if err := util.ValidateAWSCredentials(cfg.AwsProfile); err != nil {
		log.Error(fmt.Sprintf("AWS credential validation failed: %v", err))
		os.Exit(1)
	}
	log.Info("✓ AWS credentials are valid")

	// Verify pull secret
	if !util.FileExists(cfg.PullSecretPath) {
		handleMissingPullSecret(log, cfg)
	}

	// Validate pull secret format
	if err := config.ValidatePullSecret(cfg.PullSecretPath); err != nil {
		log.Error(fmt.Sprintf("Pull secret validation failed: %v", err))
		log.Info("Please ensure the pull secret is valid JSON format")
		os.Exit(1)
	}

	// Check if cluster directory already exists
	clusterDir := util.GetClusterPath(cfg.ClusterName, "")
	if util.DirExists(clusterDir) {
		log.Error(fmt.Sprintf("Cluster directory already exists: %s", clusterDir))
		log.Error(fmt.Sprintf("A cluster with name '%s' appears to already exist or was previously installed", cfg.ClusterName))
		log.Info("")
		log.Info("Options:")
		log.Info("  1. Use a different cluster name: --cluster-name=<new-name>")
		log.Info("  2. Clean up the existing cluster first:")
		log.Info("     openshift-sts-installer cleanup --help")
		os.Exit(1)
	}

	// Create command executor
	executor := &util.RealExecutor{}

	// Create step detector
	detector := steps.NewDetector(cfg)

	// Create error summary
	summary := errors.NewSummary()

	// Execute all steps
	allSteps := []struct {
		num     int
		factory func(*config.Config, *logger.Logger, util.CommandExecutor) (steps.Step, error)
	}{
		{1, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep1(c, l, e)
		}},
		{2, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep2(c, l, e)
		}},
		{3, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep3(c, l, e)
		}},
		{4, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep4(c, l, e)
		}},
		{5, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep5(c, l, e)
		}},
		{6, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep6(c, l, e)
		}},
		{7, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep7(c, l, e)
		}},
		{8, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep8(c, l, e)
		}},
		{9, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep9(c, l, e)
		}},
		{10, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep10(c, l, e)
		}},
		{11, func(c *config.Config, l *logger.Logger, e util.CommandExecutor) (steps.Step, error) {
			return steps.NewStep11(c, l, e)
		}},
	}

	for _, stepDef := range allSteps {
		// Create step to get its name
		step, err := stepDef.factory(cfg, log, executor)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to create step: %v", err))
			summary.AddError(fmt.Sprintf("Step %d", stepDef.num), err)
			continue
		}

		if detector.ShouldSkipStep(stepDef.num) {
			log.Info(fmt.Sprintf("⏭  Skipping [Step %d] %s (already completed)", stepDef.num, step.Name()))
			continue
		}

		// Optionally confirm before executing the step
		if cfg.ConfirmEachStep {
			if !confirm(fmt.Sprintf("Proceed with [Step %d] %s? [y/N] ", stepDef.num, step.Name())) {
				log.Info(fmt.Sprintf("⏭  Skipping [Step %d] %s (user choice)", stepDef.num, step.Name()))
				continue
			}
		}

		log.StartStep(fmt.Sprintf("[Step %d] %s", stepDef.num, step.Name()))

		if err := step.Execute(); err != nil {
			log.FailStep(fmt.Sprintf("[Step %d] %s", stepDef.num, step.Name()))
			summary.AddError(fmt.Sprintf("[Step %d] %s", stepDef.num, step.Name()), err)
			break
		} else {
			log.CompleteStep(fmt.Sprintf("[Step %d] %s", stepDef.num, step.Name()))
			summary.AddSuccess(fmt.Sprintf("[Step %d] %s", stepDef.num, step.Name()))

			// After Step 1, save installation metadata for cleanup purposes
			if stepDef.num == 1 {
				clusterDir := util.GetClusterPath(cfg.ClusterName, "")
				if err := util.SaveInstallMetadata(clusterDir, cfg.ReleaseImage); err != nil {
					log.Debug(fmt.Sprintf("Could not save install metadata: %v", err))
				} else {
					log.Debug(fmt.Sprintf("Saved installation metadata to %s/install-metadata.json", clusterDir))
				}
			}

			// After Step 5, backup install-config.yaml before Step 6 consumes it
			if stepDef.num == 5 {
				versionArch, err := util.ExtractVersionArch(cfg.ReleaseImage)
				if err == nil {
					installConfigPath := util.GetInstallConfigPath(versionArch, cfg.ClusterName)
					if util.FileExists(installConfigPath) {
						backupPath := installConfigPath + ".backup"
						if err := util.CopyFile(installConfigPath, backupPath); err != nil {
							log.Debug(fmt.Sprintf("Could not backup install-config.yaml: %v", err))
						} else {
							log.Debug(fmt.Sprintf("Backed up install-config.yaml to %s", backupPath))
						}
					}
				}
			}
		}
	}

	// Print summary
	fmt.Println(summary.String())

	if summary.HasErrors() {
		os.Exit(1)
	}
}

func loadConfig(log *logger.Logger) *config.Config {
	cfg := &config.Config{}

	// 1. Load from environment variables
	envCfg := config.LoadFromEnv()
	cfg.Merge(envCfg)

	// 2. Load from file
	configFile := cfgFile
	if configFile == "" {
		configFile = "openshift-sts-installer.yaml"
	}
	if util.FileExists(configFile) {
		fileCfg, err := config.LoadFromFile(configFile)
		if err != nil {
			log.Debug(fmt.Sprintf("Could not load config file: %v", err))
		} else {
			cfg.Merge(fileCfg)
		}
	}

	// 3. Merge flags
	flagCfg := &config.Config{
		ReleaseImage:    releaseImage,
		ClusterName:     clusterName,
		AwsProfile:      awsProfile,
		PullSecretPath:  pullSecretPath,
		PrivateBucket:   privateBucket,
		StartFromStep:   startFromStep,
		ConfirmEachStep: confirmEachStep,
		InstanceType:    instanceType,
	}
	cfg.Merge(flagCfg)

	// 4. Set defaults
	cfg.SetDefaults()

	return cfg
}

func handleMissingPullSecret(log *logger.Logger, cfg *config.Config) {
	log.Error("Pull-secret is required but not found.")
	log.Info("Please download it from: https://cloud.redhat.com/openshift/install/pull-secret")

	// Try to open browser
	if err := util.OpenBrowser("https://cloud.redhat.com/openshift/install/pull-secret"); err != nil {
		log.Debug(fmt.Sprintf("Could not open browser: %v", err))
	}

	// Wait for user to provide path
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter path to pull-secret file: ")
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)

	if !util.FileExists(path) {
		log.Error("File does not exist. Exiting.")
		os.Exit(1)
	}

	cfg.PullSecretPath = path
}

// confirm prompts the user with a yes/no question and returns true only for 'y' or 'Y'.
func confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	return strings.ToLower(answer) == "y"
}
