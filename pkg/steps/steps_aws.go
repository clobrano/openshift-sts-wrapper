package steps

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/config"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/logger"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/util"
)

// Step7CreateAWSResources runs ccoctl to create AWS resources
type Step7CreateAWSResources struct {
	*BaseStep
}

func NewStep7(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step7CreateAWSResources, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step7CreateAWSResources{BaseStep: base}, nil
}

func (s *Step7CreateAWSResources) Name() string {
	return "Create AWS resources"
}

func (s *Step7CreateAWSResources) Execute() error {
	ccoctlBin := util.GetBinaryPath(s.versionArch, "ccoctl")
	credreqsPath := util.GetCredReqsPath(s.versionArch)

	// Cluster name and region should be available from config
	// (loaded after Step 4 from install-config.yaml if not specified)
	if s.cfg.ClusterName == "" || s.cfg.AwsRegion == "" {
		return fmt.Errorf("cluster name and AWS region are required. They should have been loaded from install-config.yaml after Step 4")
	}

	args := []string{
		"aws", "create-all",
		"--name", s.cfg.ClusterName,
		"--region", s.cfg.AwsRegion,
		"--credentials-requests-dir", credreqsPath,
		"--output-dir", s.cfg.OutputDir,
	}

	if s.cfg.PrivateBucket {
		args = append(args, "--create-private-s3-bucket")
	}

	// Get AWS credentials from profile and set as environment variables
	awsEnv, err := util.GetAWSEnvVars(s.cfg.AwsProfile)
	if err != nil {
		s.log.Debug(fmt.Sprintf("Could not read AWS credentials from profile '%s': %v", s.cfg.AwsProfile, err))
		s.log.Debug("Proceeding without setting AWS credentials from profile")
		return util.RunCommand(s.executor, ccoctlBin, args...)
	}

	return util.RunCommandWithEnv(s.executor, awsEnv, ccoctlBin, args...)
}

// Step8CopyManifests copies manifests from _output to manifests/
type Step8CopyManifests struct {
	*BaseStep
}

func NewStep8(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step8CopyManifests, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step8CopyManifests{BaseStep: base}, nil
}

func (s *Step8CopyManifests) Name() string {
	return "Copy manifests"
}

func (s *Step8CopyManifests) Execute() error {
	srcDir := filepath.Join(s.cfg.OutputDir, "manifests")
	dstDir := "manifests"

	if err := util.EnsureDir(dstDir); err != nil {
		return err
	}

	return copyDir(srcDir, dstDir)
}

// Step9CopyTLS copies TLS files from _output to ./
type Step9CopyTLS struct {
	*BaseStep
}

func NewStep9(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step9CopyTLS, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step9CopyTLS{BaseStep: base}, nil
}

func (s *Step9CopyTLS) Name() string {
	return "Copy TLS files"
}

func (s *Step9CopyTLS) Execute() error {
	srcDir := filepath.Join(s.cfg.OutputDir, "tls")
	dstDir := "tls"

	if err := util.EnsureDir(dstDir); err != nil {
		return err
	}

	return copyDir(srcDir, dstDir)
}

// Step10DeployCluster runs openshift-install create cluster
type Step10DeployCluster struct {
	*BaseStep
}

func NewStep10(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step10DeployCluster, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step10DeployCluster{BaseStep: base}, nil
}

func (s *Step10DeployCluster) Name() string {
	return "Deploy cluster"
}

func (s *Step10DeployCluster) Execute() error {
	versionDir := filepath.Join("artifacts", s.versionArch)
	installBin := util.GetBinaryPath(s.versionArch, "openshift-install")
	args := []string{"create", "cluster", "--dir", versionDir, "--log-level=debug"}

	// Get AWS credentials from profile and set as environment variables
	awsEnv, err := util.GetAWSEnvVars(s.cfg.AwsProfile)
	if err != nil {
		s.log.Debug(fmt.Sprintf("Could not read AWS credentials from profile '%s': %v", s.cfg.AwsProfile, err))
		s.log.Debug("Proceeding without setting AWS credentials from profile")
		// Use interactive execution to stream output in real-time
		return s.executor.ExecuteInteractive(installBin, args...)
	}

	// Use interactive execution with env vars to stream output in real-time
	return s.executor.ExecuteInteractiveWithEnv(installBin, awsEnv, args...)
}

// Step11Verify performs post-install verification
type Step11Verify struct {
	*BaseStep
}

func NewStep11(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step11Verify, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step11Verify{BaseStep: base}, nil
}

func (s *Step11Verify) Name() string {
	return "Verify installation"
}

func (s *Step11Verify) Execute() error {
	// Check 1: Root credentials should not exist
	_, err := s.executor.Execute("oc", "get", "secrets", "-n", "kube-system", "aws-creds")
	if err == nil {
		s.log.Error("WARNING: Root credentials secret exists (expected it to not exist)")
	} else {
		s.log.Info("✓ Root credentials secret does not exist (as expected)")
	}

	// Check 2: Components should use IAM roles
	output, err := s.executor.Execute("oc", "get", "secrets", "-n", "openshift-image-registry",
		"installer-cloud-credentials", "-o", "json")
	if err != nil {
		return fmt.Errorf("failed to check IAM role usage: %w", err)
	}

	if len(output) > 0 && (contains(output, "role_arn") || contains(output, "web_identity_token_file")) {
		s.log.Info("✓ Components are using IAM roles")
	} else {
		s.log.Error("WARNING: Components may not be using IAM roles correctly")
	}

	return nil
}

// Helper function to copy directories
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && haystack != "" && needle != "" &&
		findSubstring(haystack, needle)
}

func findSubstring(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
