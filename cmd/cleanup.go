package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/config"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/logger"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/util"
)

var (
	cleanupClusterName  string
	cleanupAwsRegion    string
	cleanupReleaseImage string
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up AWS resources after a failed installation",
	Long:  `Removes AWS resources (infrastructure, DNS, S3 bucket, IAM roles) created during installation`,
	Run:   runCleanup,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	cleanupCmd.Flags().StringVar(&cleanupClusterName, "cluster-name", "", "Cluster/infrastructure name")
	cleanupCmd.Flags().StringVar(&cleanupAwsRegion, "region", "", "AWS region")
	cleanupCmd.Flags().StringVar(&cleanupReleaseImage, "release-image", "", "OpenShift release image (to find correct version directory)")
	cleanupCmd.MarkFlagRequired("cluster-name")
	cleanupCmd.MarkFlagRequired("region")
}

func runCleanup(cmd *cobra.Command, args []string) {
	log := logger.New(logger.Level(getLogLevel()), nil)

	// Load config to get AWS profile
	cfg := &config.Config{}
	configFile := cfgFile
	if configFile == "" {
		configFile = "openshift-sts-installer.yaml"
	}
	if util.FileExists(configFile) {
		fileCfg, err := config.LoadFromFile(configFile)
		if err != nil {
			log.Debug(fmt.Sprintf("Could not load config file: %v", err))
		} else {
			cfg = fileCfg
		}
	}
	cfg.SetDefaults()

	// Validate AWS credentials before proceeding
	log.Info(fmt.Sprintf("Validating AWS credentials for profile '%s'...", cfg.AwsProfile))
	if err := util.ValidateAWSCredentials(cfg.AwsProfile); err != nil {
		log.Error(fmt.Sprintf("AWS credential validation failed: %v", err))
		os.Exit(1)
	}
	log.Info("✓ AWS credentials are valid")

	// Confirm with user
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("This will delete AWS resources for cluster '%s' in region '%s'.\n", cleanupClusterName, cleanupAwsRegion)
	fmt.Print("Continue? (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		log.Info("Cleanup cancelled.")
		return
	}

	executor := &util.RealExecutor{}

	// Step 1: Run openshift-install destroy if we have the version directory
	if cleanupReleaseImage != "" {
		versionArch, err := util.ExtractVersionArch(cleanupReleaseImage)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to extract version from release image: %v", err))
		} else {
			versionDir := fmt.Sprintf("artifacts/%s", versionArch)
			stateFile := fmt.Sprintf("%s/.openshift_install_state.json", versionDir)
			installBin := util.GetBinaryPath(versionArch, "openshift-install")

			// Check if state file exists
			if util.FileExists(stateFile) {
				log.StartStep("Destroying OpenShift infrastructure")

				destroyArgs := []string{"destroy", "cluster", "--dir", versionDir, "--log-level=debug"}

				// TODO: we should pass AWS credentials to ExecuteInteractive
				if err := executor.ExecuteInteractive(installBin, destroyArgs...); err != nil {
					log.FailStep("Destroy infrastructure")
					log.Error(fmt.Sprintf("Failed to destroy infrastructure: %v", err))
					log.Info("Continuing with ccoctl cleanup...")
				} else {
					log.CompleteStep("Destroy infrastructure")
				}
			} else {
				log.Info(fmt.Sprintf("No state file found at %s, skipping openshift-install destroy", stateFile))
			}
		}
	} else {
		log.Info("No --release-image provided, skipping openshift-install destroy")
		log.Info("If you have orphaned infrastructure, run: ./artifacts/<version>/bin/openshift-install destroy cluster --dir artifacts/<version>/")
	}

	// Step 2: Run ccoctl aws delete to clean up IAM roles and S3 bucket
	log.StartStep("Cleaning up IAM roles and S3 bucket")

	// Find ccoctl binary
	ccoctlPath := "ccoctl"
	if cleanupReleaseImage != "" {
		versionArch, err := util.ExtractVersionArch(cleanupReleaseImage)
		if err == nil {
			versionCcoctl := util.GetBinaryPath(versionArch, "ccoctl")
			if util.FileExists(versionCcoctl) {
				ccoctlPath = versionCcoctl
			}
		}
	}
	if ccoctlPath == "ccoctl" && util.FileExists("artifacts/bin/ccoctl") {
		ccoctlPath = "artifacts/bin/ccoctl"
	}

	args_cleanup := []string{
		"aws", "delete",
		"--name", cleanupClusterName,
		"--region", cleanupAwsRegion,
	}

	if err := util.RunCommand(executor, ccoctlPath, args_cleanup...); err != nil {
		log.FailStep("Cleanup IAM/S3")
		log.Error(fmt.Sprintf("Failed to clean up IAM/S3: %v", err))
		log.Info("You may need to manually delete AWS resources.")
		os.Exit(1)
	}

	log.CompleteStep("Cleanup IAM/S3")
	log.Info("All AWS resources have been deleted.")
}
