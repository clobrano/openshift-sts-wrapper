package steps

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/config"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/logger"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/util"
	"gopkg.in/yaml.v3"
)

// Step represents a single installation step
type Step interface {
	Name() string
	Execute() error
}

// BaseStep contains common fields for all steps
type BaseStep struct {
	cfg         *config.Config
	log         *logger.Logger
	executor    util.CommandExecutor
	versionArch string
}

func newBaseStep(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*BaseStep, error) {
	versionArch, err := util.ExtractVersionArch(cfg.ReleaseImage)
	if err != nil {
		return nil, err
	}

	return &BaseStep{
		cfg:         cfg,
		log:         log,
		executor:    executor,
		versionArch: versionArch,
	}, nil
}

// Step1ExtractCredReqs extracts credentials requests from the release image
type Step1ExtractCredReqs struct {
	*BaseStep
}

func NewStep1(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step1ExtractCredReqs, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step1ExtractCredReqs{BaseStep: base}, nil
}

func (s *Step1ExtractCredReqs) Name() string {
	return "Extract credentials requests"
}

func (s *Step1ExtractCredReqs) Execute() error {
	credreqsPath := util.GetCredReqsPath(s.versionArch)
	if err := util.EnsureDir(credreqsPath); err != nil {
		return fmt.Errorf("failed to create credreqs directory: %w", err)
	}

	args := []string{
		"adm", "release", "extract",
		"--credentials-requests",
		"--cloud=aws",
		"--to=" + credreqsPath,
		s.cfg.ReleaseImage,
	}

	return util.RunCommand(s.executor, "oc", args...)
}

// Step2ExtractOpenshiftInstall extracts openshift-install binary
type Step2ExtractOpenshiftInstall struct {
	*BaseStep
}

func NewStep2(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step2ExtractOpenshiftInstall, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step2ExtractOpenshiftInstall{BaseStep: base}, nil
}

func (s *Step2ExtractOpenshiftInstall) Name() string {
	return "Extract openshift-install binary"
}

func (s *Step2ExtractOpenshiftInstall) Execute() error {
	binPath := filepath.Join("artifacts", s.versionArch, "bin")
	if err := util.EnsureDir(binPath); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Extract openshift-install
	installBinPath := filepath.Join(binPath, "openshift-install")
	args := []string{
		"adm", "release", "extract",
		"--command=openshift-install",
		"--to=" + binPath,
		s.cfg.ReleaseImage,
	}
	if err := util.RunCommand(s.executor, "oc", args...); err != nil {
		return fmt.Errorf("failed to extract openshift-install: %w", err)
	}

	// Make it executable
	os.Chmod(installBinPath, 0755)

	return nil
}

// Step3ExtractCcoctl extracts ccoctl binary
type Step3ExtractCcoctl struct {
	*BaseStep
}

func NewStep3(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step3ExtractCcoctl, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step3ExtractCcoctl{BaseStep: base}, nil
}

func (s *Step3ExtractCcoctl) Name() string {
	return "Extract ccoctl binary"
}

func (s *Step3ExtractCcoctl) Execute() error {
	binPath := filepath.Join("artifacts", s.versionArch, "bin")
	ccoctlPath := filepath.Join(binPath, "ccoctl")

	// Get CCO image
	ccoImageArgs := []string{"adm", "release", "info", "--image-for=cloud-credential-operator", s.cfg.ReleaseImage}
	ccoImage, err := s.executor.Execute("oc", ccoImageArgs...)
	if err != nil {
		return fmt.Errorf("failed to get CCO image: %w", err)
	}

	// Trim whitespace from CCO image reference
	ccoImage = strings.TrimSpace(ccoImage)

	// Extract ccoctl from CCO image (extracts to current directory)
	extractArgs := []string{
		"image", "extract",
		ccoImage,
		"--file=/usr/bin/ccoctl",
		"--registry-config=" + s.cfg.PullSecretPath,
	}
	if err := util.RunCommand(s.executor, "oc", extractArgs...); err != nil {
		return fmt.Errorf("failed to extract ccoctl: %w", err)
	}

	// Move ccoctl to the bin directory
	if err := os.Rename("ccoctl", ccoctlPath); err != nil {
		return fmt.Errorf("failed to move ccoctl to bin directory: %w", err)
	}

	// Make it executable
	os.Chmod(ccoctlPath, 0755)

	return nil
}

// Step4CreateConfig runs openshift-install create install-config
type Step4CreateConfig struct {
	*BaseStep
}

func NewStep4(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step4CreateConfig, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step4CreateConfig{BaseStep: base}, nil
}

func (s *Step4CreateConfig) Name() string {
	return "Create install-config.yaml"
}

func (s *Step4CreateConfig) Execute() error {
	// Ensure the version-specific directory exists
	versionDir := filepath.Join("artifacts", s.versionArch)
	if err := util.EnsureDir(versionDir); err != nil {
		return err
	}

	installConfigPath := util.GetInstallConfigPath(s.versionArch)

	// Check if we have complete saved configuration
	complete, missing := s.cfg.HasCompleteInstallConfigData()
	if complete {
		// Read pull secret from file
		pullSecretContent, err := os.ReadFile(s.cfg.PullSecretPath)
		if err != nil {
			s.log.Info(fmt.Sprintf("Cannot read pull secret file: %v, running interactive mode", err))
		} else {
			// Read SSH key from file
			sshKeyContent, err := os.ReadFile(s.cfg.SSHKeyPath)
			if err != nil {
				s.log.Info(fmt.Sprintf("Cannot read SSH key file: %v, running interactive mode", err))
			} else {
				// Display saved configuration to user
				s.log.Info("Found saved configuration:")
				s.log.Info(fmt.Sprintf("  Cluster Name: %s", s.cfg.ClusterName))
				s.log.Info(fmt.Sprintf("  AWS Region: %s", s.cfg.AwsRegion))
				s.log.Info(fmt.Sprintf("  Base Domain: %s", s.cfg.BaseDomain))
				s.log.Info(fmt.Sprintf("  SSH Key: %s", s.cfg.SSHKeyPath))
				s.log.Info(fmt.Sprintf("  Pull Secret: %s", s.cfg.PullSecretPath))

				// Ask user for confirmation
				if confirm(s.log, "Reuse this configuration") {
					s.log.Info("Generating install-config.yaml from saved configuration...")
					err := util.GenerateInstallConfig(
						installConfigPath,
						s.cfg.ClusterName,
						s.cfg.BaseDomain,
						s.cfg.AwsRegion,
						strings.TrimSpace(string(sshKeyContent)),
						string(pullSecretContent),
					)
					if err != nil {
						return fmt.Errorf("failed to generate install-config.yaml: %w", err)
					}
					s.log.Info("✓ install-config.yaml generated from saved configuration")
					return nil
				}
				s.log.Info("User chose to enter configuration interactively")
			}
		}
	} else {
		// Log which fields are missing
		s.log.Info(fmt.Sprintf("Saved configuration is incomplete, running interactive mode (missing: %s)", strings.Join(missing, ", ")))
	}

	// Run openshift-install create install-config (interactive)
	installBin := util.GetBinaryPath(s.versionArch, "openshift-install")
	args := []string{"create", "install-config", "--dir", versionDir}

	// Get AWS credentials from profile and pass them as environment variables
	envVars, err := util.GetAWSEnvVars(s.cfg.AwsProfile)
	if err != nil {
		s.log.Debug(fmt.Sprintf("Could not read AWS credentials: %v", err))
		s.log.Debug("Proceeding without explicit AWS credential injection")
		return s.executor.ExecuteInteractive(installBin, args...)
	}

	return s.executor.ExecuteInteractiveWithEnv(installBin, envVars, args...)
}

// maskString masks a string showing only first and last n characters
func maskString(s string, showChars int) string {
	if len(s) <= showChars*2 {
		return strings.Repeat("*", len(s))
	}
	return s[:showChars] + "..." + s[len(s)-showChars:]
}

// confirm prompts the user with a yes/no question
func confirm(log *logger.Logger, prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s? [y/N] ", prompt)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	return strings.ToLower(answer) == "y"
}

// Step5SetCredentialsMode appends credentialsMode: Manual to install-config.yaml
type Step5SetCredentialsMode struct {
	*BaseStep
}

func NewStep5(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step5SetCredentialsMode, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step5SetCredentialsMode{BaseStep: base}, nil
}

func (s *Step5SetCredentialsMode) Name() string {
	return "Set credentialsMode to Manual"
}

func (s *Step5SetCredentialsMode) Execute() error {
	configPath := util.GetInstallConfigPath(s.versionArch)

	// Read existing config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read install-config.yaml: %w", err)
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return fmt.Errorf("failed to parse install-config.yaml: %w", err)
	}

	// Ensure credentialsMode: Manual exists at top-level
	if _, exists := doc["credentialsMode"]; !exists {
		doc["credentialsMode"] = "Manual"
	}

	// Helper to ensure platform.aws.type is set in a machine pool-like object
	desiredType := s.cfg.InstanceType
	if strings.TrimSpace(desiredType) == "" {
		desiredType = "m5.4xlarge"
	}

	ensurePoolType := func(pool map[string]interface{}) {
		platform, ok := pool["platform"].(map[string]interface{})
		if !ok {
			platform = map[string]interface{}{}
			pool["platform"] = platform
		}
		aws, ok := platform["aws"].(map[string]interface{})
		if !ok {
			aws = map[string]interface{}{}
			platform["aws"] = aws
		}
		if _, ok := aws["type"]; !ok || aws["type"] == "" {
			aws["type"] = desiredType
		}
	}

	// controlPlane
	if cpRaw, ok := doc["controlPlane"]; ok {
		if cp, ok := cpRaw.(map[string]interface{}); ok {
			ensurePoolType(cp)
		}
	}

	// compute (list of pools)
	if compsRaw, ok := doc["compute"]; ok {
		if comps, ok := compsRaw.([]interface{}); ok {
			for i := range comps {
				if pool, ok := comps[i].(map[string]interface{}); ok {
					ensurePoolType(pool)
				}
			}
			// assign back in case underlying slice was modified
			doc["compute"] = comps
		}
	}

	// Marshal back to YAML
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to serialize install-config.yaml: %w", err)
	}
	if err := os.WriteFile(configPath, out, 0644); err != nil {
		return fmt.Errorf("failed to write install-config.yaml: %w", err)
	}

	return nil
}

// Step6CreateManifests runs openshift-install create manifests
type Step6CreateManifests struct {
	*BaseStep
}

func NewStep6(cfg *config.Config, log *logger.Logger, executor util.CommandExecutor) (*Step6CreateManifests, error) {
	base, err := newBaseStep(cfg, log, executor)
	if err != nil {
		return nil, err
	}
	return &Step6CreateManifests{BaseStep: base}, nil
}

func (s *Step6CreateManifests) Name() string {
	return "Create manifests"
}

func (s *Step6CreateManifests) Execute() error {
	versionDir := filepath.Join("artifacts", s.versionArch)
	installBin := util.GetBinaryPath(s.versionArch, "openshift-install")
	args := []string{"create", "manifests", "--dir", versionDir}

	return util.RunCommand(s.executor, installBin, args...)
}

// Additional steps will follow the same pattern...
// For brevity, I'll implement the remaining steps in separate files
