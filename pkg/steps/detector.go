package steps

import (
	"path/filepath"

	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/config"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/util"
)

type Detector struct {
	cfg         *config.Config
	versionArch string
}

func NewDetector(cfg *config.Config) *Detector {
	versionArch, _ := util.ExtractVersionArch(cfg.ReleaseImage)
	return &Detector{
		cfg:         cfg,
		versionArch: versionArch,
	}
}

func (d *Detector) ShouldSkipStep(stepNum int) bool {
	// If StartFromStep is set, skip all steps before it
	if d.cfg.StartFromStep > 0 && stepNum < d.cfg.StartFromStep {
		return true
	}

	// Otherwise, check for evidence of completion
	switch stepNum {
	case 1:
		// Step 1: Extract credentials requests
		return util.DirExistsWithFiles(util.GetCredReqsPath(d.versionArch))
	case 2:
		// Step 2: Extract openshift-install binary
		return util.FileExists(util.GetBinaryPath(d.versionArch, "openshift-install"))
	case 3:
		// Step 3: Extract ccoctl binary
		return util.FileExists(util.GetBinaryPath(d.versionArch, "ccoctl"))
	case 4:
		// Step 4: Create install-config.yaml
		return util.FileExists(util.GetInstallConfigPath(d.versionArch))
	case 5:
		// Step 5: Set credentialsMode
		return util.FileContains(util.GetInstallConfigPath(d.versionArch), "credentialsMode: Manual")
	case 6:
		// Step 6: Create manifests
		return util.DirExistsWithFiles(filepath.Join("artifacts", d.versionArch, d.cfg.OutputDir, "manifests"))
	case 7:
		// Step 7: Create AWS resources
		return util.DirExistsWithFiles(filepath.Join("artifacts", d.versionArch, d.cfg.OutputDir, "manifests")) &&
			util.DirExistsWithFiles(filepath.Join("artifacts", d.versionArch, d.cfg.OutputDir, "tls"))
	case 8:
		// Step 8: Copy manifests
		return util.DirExistsWithFiles(filepath.Join("artifacts", d.versionArch, "manifests"))
	case 9:
		// Step 9: Copy TLS
		return util.DirExistsWithFiles(filepath.Join("artifacts", d.versionArch, "tls"))

	case 10:
		// Step 10: Deploy cluster
		// Always try to deploy the cluster, don't skip it
		return false
	case 11:
		// Step 11: Verify installation
		// Verification should always run, don't skip it
		return false
	default:
		return false
	}
}
