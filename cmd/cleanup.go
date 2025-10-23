package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/config"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/logger"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/util"
)

var (
	cleanupClusterName string
	cleanupAwsRegion   string
	cleanupAwsProfile  string
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up AWS resources after a failed installation",
	Long:  `Removes AWS resources (S3 bucket, IAM roles) created during installation`,
	Run:   runCleanup,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	cleanupCmd.Flags().StringVar(&cleanupClusterName, "cluster-name", "", "Cluster/infrastructure name")
	cleanupCmd.Flags().StringVar(&cleanupAwsRegion, "region", "", "AWS region")
	cleanupCmd.Flags().StringVar(&cleanupAwsProfile, "aws-profile", "", "AWS profile name (default: default)")
}

func runCleanup(cmd *cobra.Command, args []string) {
	log := logger.New(logger.Level(getLogLevel()), nil)

	// Load configuration with priority: flags > file > env > prompts
	cfg := config.LoadConfig(cfgFile, log)

	// 3. Merge flags
	flagCfg := &config.Config{
		ClusterName: cleanupClusterName,
		AwsRegion:   cleanupAwsRegion,
		AwsProfile:  cleanupAwsProfile,
	}
	cfg.Merge(flagCfg)

	// Confirm with user
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("This will delete AWS resources for cluster '%s' in region '%s'.\n", cfg.ClusterName, cfg.AwsRegion)
	fmt.Print("Continue? (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		log.Info("Cleanup cancelled.")
		return
	}

	log.StartStep("Cleaning up AWS resources")

	executor := &util.RealExecutor{}

	// Find ccoctl binary - check common locations
	ccoctlPath := "ccoctl"
	if util.FileExists("artifacts/bin/ccoctl") {
		ccoctlPath = "artifacts/bin/ccoctl"
	}

	args_cleanup := []string{
		"aws", "delete",
		"--name", cfg.ClusterName,
		"--region", cfg.AwsRegion,
	}

	if err := util.RunCommand(executor, ccoctlPath, args_cleanup...); err != nil {
		log.FailStep("Cleanup")
		log.Error(fmt.Sprintf("Failed to clean up: %v", err))
		log.Info("You may need to manually delete AWS resources.")
		os.Exit(1)
	}

	log.CompleteStep("Cleanup")
	log.Info("AWS resources have been deleted.")

	// Remove outputDir after successful cleanup
	if cfg.OutputDir != "" && cfg.OutputDir != "." {
		outputPath := filepath.Clean(cfg.OutputDir)
		if util.FileExists(outputPath) {
			log.Info(fmt.Sprintf("Removing output directory: %s", outputPath))
			if err := os.RemoveAll(outputPath); err != nil {
				log.Error(fmt.Sprintf("Failed to remove output directory %s: %v", outputPath, err))
				log.Info("You may need to manually remove the output directory.")
			} else {
				log.Info("Output directory removed successfully.")
			}
		}
	}
}
