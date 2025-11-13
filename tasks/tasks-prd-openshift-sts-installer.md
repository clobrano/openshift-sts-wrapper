# Tasks: OpenShift STS Installation Wrapper

## Relevant Files

### Core Application Files
- `cmd/root.go` ✅ - Root command setup with Cobra, global flags, and version info
- `cmd/install.go` ✅ - Main `install` command implementation
- `cmd/cleanup.go` ✅ - `cleanup` subcommand for removing AWS resources
- `main.go` ✅ - Application entry point

**Note:** Individual subcommands (`extract`, `create-manifests`, `create-aws-resources`, `deploy`) were not implemented as the `install` command orchestrates all steps and `--start-from-step` provides resume functionality.

### Configuration and Utilities
- `pkg/config/config.go` ✅ - Configuration struct and loading logic (flags, files, env vars, prompts)
- `pkg/config/validate.go` ✅ - Configuration validation functions and prerequisite checks
- `pkg/config/config_test.go` ✅ - Unit tests for configuration
- `pkg/config/validate_test.go` ✅ - Unit tests for validation
- `pkg/util/version.go` ✅ - Extract version-arch from release image URL
- `pkg/util/version_test.go` ✅ - Unit tests for version parsing
- `pkg/util/browser.go` ✅ - Open browser for pull-secret download (xdg-open)
- `pkg/util/filecheck.go` ✅ - File and directory existence checks
- `pkg/util/exec.go` ✅ - Command execution wrapper with mocking support

### Step Implementation
- `pkg/steps/detector.go` ✅ - Evidence-based step detection logic
- `pkg/steps/detector_test.go` ✅ - Unit tests for step detection
- `pkg/steps/steps.go` ✅ - Steps 1-5 implementation
- `pkg/steps/steps_aws.go` ✅ - Steps 6-10 implementation
- `pkg/steps/steps_test.go` ✅ - Unit tests for steps 1-5
- `pkg/steps/steps_aws_test.go` ✅ - Unit tests for steps 6-10

### Logging and Error Handling
- `pkg/logger/logger.go` ✅ - Logger with verbosity levels (quiet, normal, verbose)
- `pkg/logger/logger_test.go` ✅ - Unit tests for logger output
- `pkg/errors/handler.go` ✅ - Error summary and final status reporting
- `pkg/errors/handler_test.go` ✅ - Unit tests for error handling

### Build and Configuration
- `go.mod` ✅ - Go module definition with dependencies (cobra, viper)
- `go.sum` ✅ - Dependency checksums
- `Makefile` ✅ - Build targets (build, test, install, clean)
- `.gitignore` ✅ - Ignore artifacts/, _output/, binaries, etc.
- `README.md` ✅ - Comprehensive usage documentation and examples
- `openshift-sts-installer.yaml.example` ✅ - Example configuration file

### Notes

- Go tests are typically placed in the same package with `_test.go` suffix
- Use `go test ./...` to run all tests
- Use `go test -v ./pkg/steps` to run specific package tests with verbose output
- The tool creates version-specific directories: `artifacts/${RHOCP_version}-${Arch}/`
- All external commands (oc, ccoctl, openshift-install) are mocked in tests via CommandExecutor interface

## Tasks

- [x] 1.0 Project Setup and Foundation
  - [x] 1.1 Initialize Go module with `go mod init github.com/clobrano/openshift-sts-wrapper`
  - [x] 1.2 Add dependencies: `go get github.com/spf13/cobra github.com/spf13/viper gopkg.in/yaml.v3`
  - [x] 1.3 Create directory structure: `cmd/`, `pkg/config/`, `pkg/steps/`, `pkg/util/`, `pkg/logger/`, `pkg/errors/`
  - [x] 1.4 Create `main.go` as entry point that calls `cmd.Execute()`
  - [x] 1.5 Create `.gitignore` to exclude `artifacts/`, `_output/`, `manifests/`, binaries, etc.
  - [x] 1.6 Create comprehensive `README.md` with usage instructions

- [x] 2.0 Configuration Management System
  - [x] 2.1 Define `Config` struct in `pkg/config/config.go` with fields: ReleaseImage, ClusterName, AwsRegion, PullSecretPath, PrivateBucket, OutputDir, StartFromStep
  - [x] 2.2 Implement configuration loading with priority: flags > config file > env vars
  - [x] 2.3 Implement `LoadFromFile()` to read `openshift-sts-installer.yaml` from current directory or custom path
  - [x] 2.4 Implement `LoadFromEnv()` to read environment variables (e.g., `OPENSHIFT_STS_RELEASE_IMAGE`)
  - [x] 2.5 Implement config merging logic
  - [x] 2.6 Add validation in `pkg/config/validate.go`: validate release image, cluster name, region, pull-secret JSON format
  - [x] 2.7 Write unit tests in `pkg/config/config_test.go` and `pkg/config/validate_test.go` for config loading priority and validation

- [x] 3.0 Core CLI Framework and Command Structure
  - [x] 3.1 Create `cmd/root.go` with Cobra root command, global flags (--config, --verbose, --quiet), and version info
  - [x] 3.2 Create `cmd/install.go` implementing the `install` command that orchestrates all steps
  - [x] 3.3 Add flags to `install` command: --release-image, --cluster-name, --region, --pull-secret, --private-bucket, --start-from-step
  - [x] 3.4-3.7 ~~Individual subcommands~~ (Skipped - `install` command with `--start-from-step` provides same functionality)
  - [x] 3.8 Create `cmd/cleanup.go` for the cleanup subcommand with confirmation prompt
  - [x] 3.9 Wire up commands to the root command in `cmd/root.go`

- [x] 4.0 Step Detection and Resume Logic
  - [x] 4.1 Implement `pkg/util/version.go` with `ExtractVersionArch(releaseImage string) string` to parse version-arch from image URL
  - [x] 4.2 Write unit tests in `pkg/util/version_test.go` for version parsing edge cases
  - [x] 4.3 Implement `pkg/steps/detector.go` with `ShouldSkipStep(stepNum int, config Config) bool` function
  - [x] 4.4 Add detection logic for Step 1-2: check if `artifacts/${version}/credreqs/` exists with files
  - [x] 4.5 Add detection logic for Step 3: check if `artifacts/${version}/bin/openshift-install` and `ccoctl` exist
  - [x] 4.6 Add detection logic for Step 4: check if `artifacts/${version}/install-config.yaml` exists
  - [x] 4.7 Add detection logic for Step 5: check if install-config.yaml contains `credentialsMode: Manual`
  - [x] 4.8 Add detection logic for Step 6: check if `manifests/` directory exists with files
  - [x] 4.9 Add detection logic for Step 7: check if `_output/manifests/` and `_output/tls/` exist with files
  - [x] 4.10 Add detection logic for Step 8-9: check if target files exist in `manifests/` and `tls/`
  - [x] 4.11 Add detection logic for Step 10: check if `.openshift_install.log` exists
  - [x] 4.12 Implement `--start-from-step` flag override logic to bypass detection
  - [x] 4.13 Write unit tests in `pkg/steps/detector_test.go` for each detection scenario

- [x] 5.0 Installation Steps Implementation (Steps 1-10)
  - [x] 5.1 Implement Step 1 in `pkg/steps/steps.go` running `oc adm release extract --credentials-requests`
  - [x] 5.2 Implement Step 2 running `oc adm release extract --command=openshift-install` and extracting ccoctl with `oc image extract`
  - [x] 5.3 Ensure binaries are extracted to `artifacts/${version}/bin/` and are executable (chmod +x)
  - [x] 5.4 Implement Step 3 running `artifacts/${version}/bin/openshift-install create install-config`
  - [x] 5.5 Implement Step 4 to append `credentialsMode: Manual` to install-config.yaml
  - [x] 5.6 Implement Step 5 running `openshift-install create manifests`
  - [x] 5.7 Implement Step 6 in `pkg/steps/steps_aws.go` running `ccoctl aws create-all` with optional `--create-private-s3-bucket` flag
  - [x] 5.8 Implement Step 7 copying files from `_output/manifests/*` to `manifests/`
  - [x] 5.9 Implement Step 8 copying `_output/tls/` directory to `./tls/`
  - [x] 5.10 Implement Step 9 running `openshift-install create cluster`
  - [x] 5.11 Implement Step 10 with post-install verification checks (check secrets, check IAM role usage)
  - [x] 5.12 Create `pkg/util/exec.go` with mock executor for testing
  - [x] 5.13 Ensure all step functions accept config, logger, and return descriptive errors

- [x] 6.0 Error Handling and User Interaction
  - [x] 6.1 Implement `pkg/errors/handler.go` with error summary functionality
  - [x] 6.2 Add interactive prompt in install command: "An error occurred in {stepName}. Continue anyway? (y/n)"
  - [x] 6.3 Track errors in summary for final report
  - [x] 6.4 Implement error summary with `String()` method
  - [x] 6.5 Display summary at end showing: successful steps, steps with errors, overall status
  - [x] 6.6 Implement pull-secret validation: check file exists, validate JSON format
  - [x] 6.7 Implement `pkg/util/browser.go` with `OpenBrowser(url string)` using `xdg-open` or equivalent
  - [x] 6.8 Add pull-secret prompt flow: detect missing file, display message, open browser, wait for user to provide path
  - [x] 6.9 Implement prerequisite validation: check `oc` command is available in PATH before starting

- [x] 7.0 Logging and Progress Indicators
  - [x] 7.1 Implement `pkg/logger/logger.go` with Logger struct and three verbosity levels: quiet, normal, verbose
  - [x] 7.2 Implement `Info(msg string)` (shown in normal/verbose), `Debug(msg string)` (shown only in verbose), `Error(msg string)` (shown always)
  - [x] 7.3 Implement progress indicators: `StartStep(name string)` displays "⏳ {name}..."
  - [x] 7.4 Implement `CompleteStep(name string)` displays "✓ {name}"
  - [x] 7.5 Implement `FailStep(name string)` displays "✗ {name}"
  - [x] 7.6 Add `--quiet` and `--verbose` flags to root command and configure logger accordingly
  - [x] 7.7 Pass logger instance to all step functions
  - [x] 7.8 Write unit tests in `pkg/logger/logger_test.go` testing output for each verbosity level

- [x] 8.0 Cleanup Command
  - [x] 8.1 Implement `cmd/cleanup.go` accepting --cluster-name and --region flags
  - [x] 8.2 Add confirmation prompt: "This will delete AWS resources for cluster {name}. Continue? (y/n)"
  - [x] 8.3 Run `ccoctl aws delete` command with appropriate parameters
  - [x] 8.4 Display messages about resource deletion
  - [x] 8.5 Handle errors gracefully if resources don't exist or deletion fails

- [x] 9.0 Post-Install Verification
  - [x] 9.1 Implement `pkg/steps/steps_aws.go` Step10Verify running `oc get secrets -n kube-system aws-creds` (should fail/not exist)
  - [x] 9.2 Add check for IAM role usage: `oc get secrets -n openshift-image-registry installer-cloud-credentials -o json`
  - [x] 9.3 Parse and validate that credentials contain `role_arn` and `web_identity_token_file`
  - [x] 9.4 Display verification results with clear pass/fail indicators
  - [x] 9.5 Return overall verification status

- [x] 10.0 Testing and Documentation
  - [x] 10.1 Write unit tests for configuration loading and priority (flags > file > env)
  - [x] 10.2 Write unit tests for version parsing from release image URLs
  - [x] 10.3 Write unit tests for step detection logic with various directory states
  - [x] 10.4 Write unit tests for logger output at different verbosity levels
  - [x] 10.5 Write unit tests for all 10 installation steps
  - [x] 10.6 Create `Makefile` with targets: `build`, `test`, `install`, `clean`, `fmt`, `vet`
  - [x] 10.7 Update `README.md` with installation instructions, usage examples, configuration file format
  - [x] 10.8 Add example `openshift-sts-installer.yaml.example` configuration file to repository
  - [x] 10.9 Document environment variables in README (e.g., `OPENSHIFT_STS_RELEASE_IMAGE`)
  - [x] 10.10 Add troubleshooting section to README with common issues and solutions
  - [x] 10.11 Run `go fmt ./...` and `go vet ./...` to ensure code quality
  - [x] 10.12 Build the binary with `go build -o openshift-sts-installer` and test manually

## Test Summary

All tests passing ✅

```
✓ pkg/config    - 9 tests (config loading, merging, validation, pull-secret validation)
✓ pkg/errors    - 3 tests (error summary functionality)
✓ pkg/logger    - 4 tests (verbosity levels, progress indicators)
✓ pkg/steps     - 13 tests (all 10 steps + detector + private bucket)
✓ pkg/util      - 5 tests (version parsing)
─────────────────────────────────────────────────────────────────────
Total: 34 tests, ALL PASSING
```

## Implementation Notes

### TDD Approach
- All code written test-first
- External commands (oc, ccoctl, openshift-install) are mocked via `CommandExecutor` interface
- No actual OpenShift/AWS tools required to run tests

### Simplifications from Original Plan
- **Individual subcommands skipped**: The `install` command with `--start-from-step` provides equivalent functionality with simpler UX
- **Interactive prompts for config**: Implemented config loading from multiple sources but skipped interactive prompts for missing values (user can use flags, file, or env vars)
- **Integration tests**: Skipped full integration tests as unit tests with mocks provide good coverage

### Key Features Delivered
✅ All 10 installation steps implemented and tested
✅ Version-aware directory structure for multi-version support
✅ Evidence-based step detection for resume capability
✅ Multi-source configuration (flags > file > env)
✅ Three verbosity levels with progress indicators
✅ Interactive error handling (continue/abort prompts)
✅ Comprehensive error summary
✅ Pull-secret validation and browser opening
✅ Prerequisite checking (oc command)
✅ Private S3 bucket support
✅ Cleanup command for AWS resources
✅ Complete documentation and examples
