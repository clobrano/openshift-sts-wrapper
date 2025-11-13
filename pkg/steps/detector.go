package steps

import (
	"github.com/clobrano/openshift-sts-wrapper/pkg/config"
	"github.com/clobrano/openshift-sts-wrapper/pkg/util"
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
		// Step 1: Extract credentials requests (shared)
		return util.DirExistsWithFiles(util.GetSharedCredReqsPath(d.versionArch))
	case 2:
		// Step 2: Extract openshift-install binary (shared)
		return util.FileExists(util.GetSharedBinaryPath(d.versionArch, "openshift-install"))
	case 3:
		// Step 3: Extract ccoctl binary (shared)
		return util.FileExists(util.GetSharedBinaryPath(d.versionArch, "ccoctl"))
	case 4:
		// Step 4: Create install-config.yaml (cluster-specific)
		return util.FileExists(util.GetInstallConfigPath(d.versionArch, d.cfg.ClusterName))
	case 5:
		// Step 5: Set credentialsMode (cluster-specific)
		return util.FileContains(util.GetInstallConfigPath(d.versionArch, d.cfg.ClusterName), "credentialsMode: Manual")
	case 6:
		// Step 6: Create manifests (cluster-specific)
		return util.DirExistsWithFiles(util.GetClusterPath(d.cfg.ClusterName, "ccoctl-output/manifests"))
	case 7:
		// Step 7: Create AWS resources (cluster-specific)
		return util.DirExistsWithFiles(util.GetClusterPath(d.cfg.ClusterName, "ccoctl-output/manifests")) &&
			util.DirExistsWithFiles(util.GetClusterPath(d.cfg.ClusterName, "ccoctl-output/tls"))
	case 8:
		// Step 8: Copy manifests (cluster-specific)
		return !util.DirExistsWithFiles(util.GetClusterPath(d.cfg.ClusterName, "ccoctl-output/manifests"))
	case 9:
		// Step 9: Copy TLS (cluster-specific)
		return !util.DirExistsWithFiles(util.GetClusterPath(d.cfg.ClusterName, "ccoctl-output/tls"))
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
