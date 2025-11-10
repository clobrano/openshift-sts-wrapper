package steps

import (
	"os"
	"path/filepath"
	"testing"

	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/config"
)

func TestShouldSkipStep(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	versionArch := "4.12.0-x86_64"
	clusterName := "test-cluster"
	cfg := &config.Config{
		ReleaseImage: "quay.io/test:4.12.0-x86_64",
		ClusterName:  clusterName,
	}

	detector := NewDetector(cfg)

	// Initially, no steps should be skipped (except steps 8 and 9 which check for non-existence)
	for i := 1; i <= 7; i++ {
		if detector.ShouldSkipStep(i) {
			t.Errorf("Step %d should not be skipped initially", i)
		}
	}

	// Create credreqs directory with a file (step 1) - shared path
	credreqsPath := filepath.Join("artifacts", "shared", versionArch, "credreqs")
	os.MkdirAll(credreqsPath, 0755)
	os.WriteFile(filepath.Join(credreqsPath, "test.yaml"), []byte("test"), 0644)

	detector = NewDetector(cfg) // Refresh detector
	if !detector.ShouldSkipStep(1) {
		t.Error("Step 1 should be skipped when credreqs exists")
	}

	// Create binaries (step 2, 3) - shared path
	binPath := filepath.Join("artifacts", "shared", versionArch, "bin")
	os.MkdirAll(binPath, 0755)
	os.WriteFile(filepath.Join(binPath, "openshift-install"), []byte("fake"), 0755)
	os.WriteFile(filepath.Join(binPath, "ccoctl"), []byte("fake"), 0755)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(2) {
		t.Error("Step 2 should be skipped when binaries exist")
	}
	if !detector.ShouldSkipStep(3) {
		t.Error("Step 3 should be skipped when ccoctl binary exists")
	}

	// Create install-config.yaml (step 4) - cluster-specific path
	configPath := filepath.Join("artifacts", "clusters", clusterName, "install-config.yaml")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	os.WriteFile(configPath, []byte("apiVersion: v1\n"), 0644)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(4) {
		t.Error("Step 4 should be skipped when install-config.yaml exists")
	}

	// Add credentialsMode to install-config.yaml (step 5)
	os.WriteFile(configPath, []byte("apiVersion: v1\ncredentialsMode: Manual\n"), 0644)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(5) {
		t.Error("Step 5 should be skipped when credentialsMode is set")
	}

	// Create ccoctl-output directories (step 6 and 7) - cluster-specific path
	ccoctlOutputPath := filepath.Join("artifacts", "clusters", clusterName, "ccoctl-output")
	os.MkdirAll(filepath.Join(ccoctlOutputPath, "manifests"), 0755)
	os.MkdirAll(filepath.Join(ccoctlOutputPath, "tls"), 0755)
	os.WriteFile(filepath.Join(ccoctlOutputPath, "manifests", "test.yaml"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(ccoctlOutputPath, "tls", "test.pem"), []byte("test"), 0644)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(6) {
		t.Error("Step 6 should be skipped when ccoctl-output/manifests exists")
	}

	// Step 7 should be skipped when both manifests and tls exist in ccoctl-output
	if !detector.ShouldSkipStep(7) {
		t.Error("Step 7 should be skipped when ccoctl-output has manifests and tls")
	}

	// Step 8 checks that ccoctl-output/manifests does NOT exist (we just created it, so step 8 should NOT be skipped)
	if detector.ShouldSkipStep(8) {
		t.Error("Step 8 should not be skipped when ccoctl-output/manifests still exists")
	}

	// Step 9 checks that ccoctl-output/tls does NOT exist (we just created it, so step 9 should NOT be skipped)
	if detector.ShouldSkipStep(9) {
		t.Error("Step 9 should not be skipped when ccoctl-output/tls still exists")
	}

	// Remove ccoctl-output to simulate steps 8 and 9 completing
	os.RemoveAll(ccoctlOutputPath)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(8) {
		t.Error("Step 8 should be skipped when ccoctl-output/manifests has been removed")
	}
	if !detector.ShouldSkipStep(9) {
		t.Error("Step 9 should be skipped when ccoctl-output/tls has been removed")
	}

	// Step 10 and 11 (deploy and verification) should never be skipped
	if detector.ShouldSkipStep(10) {
		t.Error("Step 10 should never be skipped")
	}
	if detector.ShouldSkipStep(11) {
		t.Error("Step 11 should never be skipped")
	}
}

func TestShouldSkipStepWithStartFromOverride(t *testing.T) {
	cfg := &config.Config{
		ReleaseImage:  "quay.io/test:4.12.0-x86_64",
		StartFromStep: 5,
	}

	detector := NewDetector(cfg)

	// Steps before startFromStep should be skipped
	if !detector.ShouldSkipStep(1) {
		t.Error("Step 1 should be skipped with StartFromStep=5")
	}
	if !detector.ShouldSkipStep(4) {
		t.Error("Step 4 should be skipped with StartFromStep=5")
	}

	// StartFromStep and later should not be skipped
	if detector.ShouldSkipStep(5) {
		t.Error("Step 5 should not be skipped with StartFromStep=5")
	}
	if detector.ShouldSkipStep(6) {
		t.Error("Step 6 should not be skipped with StartFromStep=5")
	}
}
