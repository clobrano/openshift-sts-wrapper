package steps

import (
	"os"
	"path/filepath"
	"testing"

	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/config"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/logger"
	"gitlab.cee.redhat.com/clobrano/ccoctl-sso/pkg/util"
)

func TestStep1ExtractCredReqs(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg := &config.Config{
		ReleaseImage: "quay.io/test:4.12.0-x86_64",
	}
	log := logger.New(logger.LevelQuiet, nil)
	executor := util.NewMockExecutor()

	step, err := NewStep1(cfg, log, executor)
	if err != nil {
		t.Fatalf("Failed to create step: %v", err)
	}

	if step.Name() != "Extract credentials requests" {
		t.Errorf("Unexpected step name: %s", step.Name())
	}

	err = step.Execute()
	if err != nil {
		t.Fatalf("Step execution failed: %v", err)
	}

	// Verify the command was executed
	if !executor.WasExecutedContaining("oc adm release extract --credentials-requests") {
		t.Error("Expected oc command to be executed")
	}

	// Verify directory was created
	credreqsPath := util.GetCredReqsPath("4.12.0-x86_64")
	if _, err := os.Stat(credreqsPath); os.IsNotExist(err) {
		t.Error("Credreqs directory was not created")
	}
}

func TestStep2ExtractBinaries(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg := &config.Config{
		ReleaseImage: "quay.io/test:4.12.0-x86_64",
	}
	log := logger.New(logger.LevelQuiet, nil)
	executor := util.NewMockExecutor()

	// Mock the CCO image output
	executor.SetOutput("oc adm release info --image-for=cloud-credential-operator quay.io/test:4.12.0-x86_64",
		"quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:abc123")

	step, err := NewStep2(cfg, log, executor)
	if err != nil {
		t.Fatalf("Failed to create step: %v", err)
	}

	err = step.Execute()
	if err != nil {
		t.Fatalf("Step execution failed: %v", err)
	}

	// Verify commands were executed
	if !executor.WasExecutedContaining("oc adm release extract --command=openshift-install") {
		t.Error("Expected openshift-install extraction command")
	}
	if !executor.WasExecutedContaining("oc image extract") {
		t.Error("Expected ccoctl extraction command")
	}
}

func TestStep3CreateConfig(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg := &config.Config{
		ReleaseImage: "quay.io/test:4.12.0-x86_64",
	}
	log := logger.New(logger.LevelQuiet, nil)
	executor := util.NewMockExecutor()

	// Create the binary path (simulating previous steps)
	binPath := filepath.Join("artifacts", "4.12.0-x86_64", "bin")
	os.MkdirAll(binPath, 0755)

	step, err := NewStep3(cfg, log, executor)
	if err != nil {
		t.Fatalf("Failed to create step: %v", err)
	}

	err = step.Execute()
	if err != nil {
		t.Fatalf("Step execution failed: %v", err)
	}

	if !executor.WasExecutedContaining("openshift-install") {
		t.Error("Expected openshift-install command to be executed")
	}
	if !executor.WasExecutedContaining("create install-config") {
		t.Error("Expected 'create install-config' in command")
	}
}

func TestStep4SetCredentialsMode(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg := &config.Config{
		ReleaseImage: "quay.io/test:4.12.0-x86_64",
		ClusterName:  "test-cluster",
	}
	log := logger.New(logger.LevelQuiet, nil)
	executor := util.NewMockExecutor()

	// Create install-config.yaml
	configPath := util.GetInstallConfigPath("4.12.0-x86_64", "test-cluster")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	os.WriteFile(configPath, []byte("apiVersion: v1\n"), 0644)

	step, err := NewStep4(cfg, log, executor)
	if err != nil {
		t.Fatalf("Failed to create step: %v", err)
	}

	err = step.Execute()
	if err != nil {
		t.Fatalf("Step execution failed: %v", err)
	}

	// Verify credentialsMode was added
	content, _ := os.ReadFile(configPath)
	if !util.FileContains(configPath, "credentialsMode: Manual") {
		t.Errorf("credentialsMode not added to config. Content: %s", string(content))
	}
}

func TestStep5CreateManifests(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg := &config.Config{
		ReleaseImage: "quay.io/test:4.12.0-x86_64",
	}
	log := logger.New(logger.LevelQuiet, nil)
	executor := util.NewMockExecutor()

	step, err := NewStep5(cfg, log, executor)
	if err != nil {
		t.Fatalf("Failed to create step: %v", err)
	}

	err = step.Execute()
	if err != nil {
		t.Fatalf("Step execution failed: %v", err)
	}

	if !executor.WasExecutedContaining("create manifests") {
		t.Error("Expected 'create manifests' command")
	}
}
