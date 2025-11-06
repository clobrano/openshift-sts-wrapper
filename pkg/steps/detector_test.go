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
	cfg := &config.Config{
		ReleaseImage: "quay.io/test:4.12.0-x86_64",
	}

	detector := NewDetector(cfg)

	// Initially, no steps should be skipped
	for i := 1; i <= 10; i++ {
		if detector.ShouldSkipStep(i) {
			t.Errorf("Step %d should not be skipped initially", i)
		}
	}

	// Create credreqs directory with a file (step 1)
	credreqsPath := filepath.Join("artifacts", versionArch, "credreqs")
	os.MkdirAll(credreqsPath, 0755)
	os.WriteFile(filepath.Join(credreqsPath, "test.yaml"), []byte("test"), 0644)

	detector = NewDetector(cfg) // Refresh detector
	if !detector.ShouldSkipStep(1) {
		t.Error("Step 1 should be skipped when credreqs exists")
	}

	// Create binaries (step 2)
	binPath := filepath.Join("artifacts", versionArch, "bin")
	os.MkdirAll(binPath, 0755)
	os.WriteFile(filepath.Join(binPath, "openshift-install"), []byte("fake"), 0755)
	os.WriteFile(filepath.Join(binPath, "ccoctl"), []byte("fake"), 0755)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(2) {
		t.Error("Step 2 should be skipped when binaries exist")
	}

	// Create install-config.yaml (step 3)
	configPath := filepath.Join("artifacts", versionArch, "install-config.yaml")
	os.WriteFile(configPath, []byte("apiVersion: v1\n"), 0644)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(3) {
		t.Error("Step 3 should be skipped when install-config.yaml exists")
	}

	// Add credentialsMode to install-config.yaml (step 4)
	os.WriteFile(configPath, []byte("apiVersion: v1\ncredentialsMode: Manual\n"), 0644)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(4) {
		t.Error("Step 4 should be skipped when credentialsMode is set")
	}

	// Create manifests directory (step 5)
	manifestsPath := "manifests"
	os.MkdirAll(manifestsPath, 0755)
	os.WriteFile(filepath.Join(manifestsPath, "test.yaml"), []byte("test"), 0644)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(5) {
		t.Error("Step 5 should be skipped when manifests exist")
	}

	// Create ccoctl-output directories (step 6 and 7)
	ccoctlOutputPath := filepath.Join("artifacts", versionArch, "ccoctl-output")
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

	// Create copied manifests directory (step 8)
	copiedManifestsPath := filepath.Join("artifacts", versionArch, "manifests")
	os.MkdirAll(copiedManifestsPath, 0755)
	os.WriteFile(filepath.Join(copiedManifestsPath, "test.yaml"), []byte("test"), 0644)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(8) {
		t.Error("Step 8 should be skipped when artifacts/*/manifests exists")
	}

	// Create copied tls directory (step 9)
	copiedTlsPath := filepath.Join("artifacts", versionArch, "tls")
	os.MkdirAll(copiedTlsPath, 0755)
	os.WriteFile(filepath.Join(copiedTlsPath, "ca.pem"), []byte("test"), 0644)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(9) {
		t.Error("Step 9 should be skipped when artifacts/*/tls exists")
	}

	// Create .openshift_install.log (step 9)
	os.WriteFile(".openshift_install.log", []byte("log"), 0644)

	detector = NewDetector(cfg)
	if !detector.ShouldSkipStep(9) {
		t.Error("Step 9 should be skipped when install log exists")
	}

	// Step 10 (verification) should never be skipped
	if detector.ShouldSkipStep(10) {
		t.Error("Step 10 should never be skipped")
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
