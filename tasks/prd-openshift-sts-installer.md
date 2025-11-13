# PRD: OpenShift STS Installation Wrapper

## Introduction/Overview

This tool provides an automated wrapper for the multi-step process of installing an OpenShift cluster with AWS Security Token Service (STS) authentication. The current installation process requires 10+ manual steps involving binary extraction, manifest generation, AWS resource creation, and cluster deployment. This wrapper consolidates these steps into a single, guided workflow that reduces human error and simplifies the installation experience for users of all skill levels.

The tool solves the key pain points of: too many manual steps prone to errors, complexity of extracting binaries and credentials, and difficulty remembering the correct sequence of operations.

## Goals

1. **Reduce Installation Complexity**: Consolidate the 10-step STS installation process into a single, automated workflow
2. **Minimize Human Error**: Eliminate manual file copying, path management, and command sequencing errors
3. **Improve User Experience**: Provide interactive guidance with clear progress indicators and error handling
4. **Support Flexibility**: Allow users to provide inputs via multiple methods (flags, config files, environment variables, interactive prompts)
5. **Enable Recovery**: Support resuming installations that were stopped mid-process (if feasible within reasonable complexity)
6. **Validate Prerequisites**: Check for required tools and validate inputs before starting the installation

## User Stories

1. **As a DevOps engineer**, I want to install an OpenShift cluster with STS in a single command, so that I don't have to manually track and execute 10+ sequential steps.

2. **As a platform engineer managing multiple clusters**, I want to provide configuration via a config file or environment variables, so that I can automate deployments without interactive prompts.

3. **As a junior developer new to OpenShift**, I want clear interactive prompts with validation, so that I understand what information is needed at each step.

4. **As any user**, I want the tool to validate prerequisites before starting, so that I discover missing dependencies or configuration issues early.

5. **As any user**, I want the tool to handle the pull-secret retrieval gracefully, so that I'm guided through the one manual step that cannot be automated.

6. **As any user**, I want to control the verbosity of output, so that I can see detailed logs when troubleshooting or minimal output during successful runs.

7. **As any user**, I want the tool to verify the installation was successful, so that I have confidence the cluster is properly configured with STS.

## Functional Requirements

### Core Installation Workflow

1. The tool MUST execute all steps from the "Steps to install an OpenShift Cluster with STS" section:
   - Extract credentials requests from release image
   - Extract openshift-install and ccoctl binaries
   - Create install-config.yaml (delegating to `openshift-install create install-config`)
   - Append `credentialsMode: Manual` to install-config.yaml
   - Create manifests
   - Run ccoctl to create AWS resources (supporting both public and private S3 bucket options)
   - Copy generated manifests to the correct location
   - Copy TLS files to the correct location
   - Run openshift-install to create the cluster
   - Perform post-install verification

2. The tool MUST be implemented as a CLI with subcommands and support for interactive mode
   - Subcommands allow running individual steps (e.g., `extract`, `create-manifests`, `install`)
   - Interactive mode guides users through the entire process with prompts
   - An `install` or `run` command executes the full workflow

3. The tool SHOULD support resuming from a stopped installation by detecting completed steps
   - Detect completion by checking for evidence (files, directories, resources)
   - Use version-aware directory structure (`artifacts/${RHOCP_version}-${Arch}/`) so version changes naturally trigger re-extraction
   - Skip already-completed steps automatically when evidence is found
   - Note: If this adds significant complexity, defer to future version

### Configuration & Input

4. The tool MUST support multiple input methods with the following priority: CLI flags > config file > environment variables > interactive prompts

5. The tool MUST accept the following required inputs:
   - `release-image`: OpenShift release image URL
   - `cluster-name` (or `aws-infra-name`): Name/prefix for AWS resources
   - `aws-region`: AWS region for deployment
   - `pull-secret-path`: Path to the pull-secret file

6. The tool MUST support an optional `--private-bucket` flag to use CloudFront with a private S3 bucket instead of a public S3 bucket

7. The tool MUST support both flag-based and interactive configuration for the private/public S3 bucket option

8. The tool MUST use the native `openshift-install create install-config` command for install-config.yaml generation (no custom prompts needed)

### Pull-Secret Handling

9. The tool MUST detect if a pull-secret file is not provided or doesn't exist

10. When the pull-secret is missing, the tool MUST:
    - Display a message: "Pull-secret is required but not found. Please download it from: https://cloud.redhat.com/openshift/install/pull-secret"
    - Attempt to open the URL in the user's default browser using `xdg-open` (or equivalent)
    - Prompt the user to download the file and specify its path
    - Wait for user confirmation before continuing
    - Validate the pull-secret file format (valid JSON)

### Prerequisite Validation

11. The tool MUST validate that required CLI tools are installed before starting:
    - `oc` (OpenShift CLI) - must be available in PATH
    - `ccoctl` and `openshift-install` will be extracted to `artifacts/${RHOCP_version}-${Arch}/bin/` during the installation process

12. The tool MUST validate that the specified release image is accessible (can be pulled/read)

13. The tool MUST NOT validate AWS permissions before starting - it will rely on AWS errors during execution if permissions are insufficient

### Cleanup Command

14. The tool MUST provide a `cleanup` subcommand to remove partial AWS resources after a failed installation
    - This command should be called manually by the user (not automatic)
    - Should use `ccoctl aws delete` or equivalent commands
    - Should prompt for confirmation before deletion

### Error Handling

15. When an error occurs during any step, the tool MUST:
    - Display the error message clearly
    - Prompt the user: "An error occurred. Continue anyway? (y/n)"
    - Stop execution if the user chooses 'n'
    - Continue to the next step if the user chooses 'y'
    - Track that errors occurred for final summary

16. The tool MUST display a summary at the end showing:
    - Which steps completed successfully
    - Which steps had errors
    - Overall status (success/partial success/failure)

### Logging & Output

17. The tool MUST support configurable verbosity levels via flags:
    - `--quiet`: Only errors and critical steps
    - Default: Progress indicators for each major step
    - `--verbose`: Detailed output from all commands

18. The tool MUST display output to console only (no automatic log file creation)

19. The tool MUST show clear progress indicators for each major step:
    - "⏳ Extracting credentials requests..."
    - "✓ Credentials requests extracted"
    - "✗ Failed to extract credentials requests"

### Post-Install Verification

20. The tool MUST perform post-install verification as documented in tasks/sts.md:
    - Verify that the root credentials secret does not exist (`oc get secrets -n kube-system aws-creds` should fail)
    - Verify that components are using IAM roles (check at least one component's credentials secret)
    - Display verification results to the user

## Non-Goals (Out of Scope)

The following are explicitly **not** included in this tool:

1. **Cluster Management**: Managing or destroying existing clusters (use `openshift-install destroy` directly)
2. **Cluster Upgrades**: Handling cluster version upgrades or migrations
3. **Post-Install Configuration**: Configuring applications, operators, or cluster settings beyond the STS verification
4. **Multi-Cloud Support**: Support for non-AWS cloud providers (GCP, Azure, etc.)
5. **Pull-Secret Automation**: Automated download of pull-secrets via web scraping or unofficial APIs (Red Hat does not provide an official API for this)
6. **AWS Resource Cleanup**: Automated cleanup of AWS resources (use `ccoctl aws delete` directly or follow the documented cleanup process)

## Design Considerations

### CLI Structure

The tool should follow standard CLI conventions:

```bash
# Full installation with interactive prompts
$ openshift-sts-wrapper install

# Full installation with flags
$ openshift-sts-wrapper install \
    --release-image=quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64 \
    --cluster-name=my-cluster \
    --region=us-east-2 \
    --pull-secret=./pull-secret.json \
    --private-bucket

# Run individual steps
$ openshift-sts-wrapper extract --release-image=...
$ openshift-sts-wrapper create-manifests
$ openshift-sts-wrapper create-aws-resources --private-bucket
$ openshift-sts-wrapper deploy

# With config file (looks for openshift-sts-wrapper.yaml by default)
$ openshift-sts-wrapper install
# Or specify custom config file
$ openshift-sts-wrapper install --config=my-config.yaml

# Cleanup after failed installation
$ openshift-sts-wrapper cleanup --cluster-name=my-cluster --region=us-east-2
```

### Config File Format

The tool should look for a config file named `openshift-sts-wrapper.yaml` (not hidden) in the current directory. The format should be:

```yaml
releaseImage: quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64
clusterName: my-cluster
awsRegion: us-east-2
pullSecretPath: ./pull-secret.json
privateBucket: true
outputDir: ./_output
```

Users can also specify a custom config file path using `--config` flag.

### User Experience Flow

1. User runs `openshift-sts-wrapper install`
2. Tool validates prerequisites (oc available, etc.)
3. Tool checks for required inputs (prompts if missing)
4. Tool checks for pull-secret, opens browser if needed, waits for user
5. Tool executes each step with progress indicators
6. On errors, tool prompts user to continue or abort
7. Tool performs post-install verification
8. Tool displays summary of results

## Technical Considerations

### Implementation Language

- **Language**: Go
- **Rationale**: Single binary distribution, no runtime dependencies, excellent CLI library support (cobra/viper), strong error handling

### Dependencies

- `spf13/cobra`: CLI framework with subcommands
- `spf13/viper`: Configuration management (flags, env vars, config files)
- Standard library for process execution

### Directory Structure

The tool should use a working directory structure that organizes artifacts by release version:

```
./
├── artifacts/
│   └── ${RHOCP_version}-${Arch}/  # e.g., 4.12.0-x86_64
│       ├── bin/                    # Extracted binaries (openshift-install, ccoctl)
│       ├── credreqs/               # Extracted credentials requests
│       └── install-config.yaml     # Generated by openshift-install
├── _output/           # ccoctl generated files
│   ├── manifests/
│   └── tls/
├── manifests/         # Installation manifests
├── pull-secret.json   # User-provided pull-secret
└── openshift-sts-wrapper.yaml # Optional config file (not hidden)
```

**Note**:
- The `${RHOCP_version}-${Arch}` directory name is derived from the release image (e.g., `quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64` → `4.12.0-x86_64`)
- The tool will invoke binaries from `artifacts/${RHOCP_version}-${Arch}/bin/` with full paths
- This structure allows multiple release versions to coexist without conflicts

### State Management (for resume capability)

The tool uses a simplified evidence-based approach with version-aware directory structure to detect completed steps. No state file is needed because the directory structure itself encodes version information.

**Simple evidence-based step detection:**

- **Step 1-2 (Extract credentials)**: Skip if `artifacts/${RHOCP_version}-${Arch}/credreqs/` exists and contains files
- **Step 3 (Extract binaries)**: Skip if `artifacts/${RHOCP_version}-${Arch}/bin/openshift-install` and `artifacts/${RHOCP_version}-${Arch}/bin/ccoctl` exist
- **Step 4 (Create install-config.yaml)**: Skip if `artifacts/${RHOCP_version}-${Arch}/install-config.yaml` exists
- **Step 5 (Set credentialsMode)**: Skip if `artifacts/${RHOCP_version}-${Arch}/install-config.yaml` contains `credentialsMode: Manual`
- **Step 6 (Create manifests)**: Skip if `manifests/` directory exists and contains files
- **Step 7 (Create AWS resources)**: Skip if `_output/manifests/` and `_output/tls/` exist with generated files
- **Step 8-9 (Copy files)**: Skip if target files exist in `manifests/` and `tls/` directories
- **Step 10 (Deploy cluster)**: Skip if `.openshift_install.log` exists or cluster is running

**Key benefits:**
- Version changes naturally trigger re-extraction (different directory path)
- No hidden state files
- Users can see exactly what's been done by looking at the directory structure
- Multiple versions can coexist for testing/comparison

### AWS Resource Naming

The tool must use consistent naming for AWS resources:
- S3 bucket: `{cluster-name}-oidc`
- IAM roles: `{cluster-name}-{component}-{credential-name}`

## Success Metrics

The primary success metric is **improved user experience and reduced installation friction**:

- Users can complete STS installation without referring to documentation
- Users report fewer errors compared to manual process
- Installation time is comparable or faster than manual process
- Users successfully complete installations on first attempt

Qualitative measures:
- User feedback indicates the tool is intuitive
- Support requests related to STS installation decrease

## Implementation Decisions

Based on requirements clarification, the following decisions have been made:

1. **Binary Extraction Location**: Extracted binaries (`openshift-install`, `ccoctl`) are placed in `artifacts/${RHOCP_version}-${Arch}/bin/` directory (version-specific) in the current working directory
2. **Validation Depth**: The tool does NOT validate AWS permissions before starting - it relies on AWS errors during execution
3. **Config File Location**: The tool looks for `openshift-sts-wrapper.yaml` (not hidden) in the current directory by default
4. **Cleanup on Failure**: On failure, the tool leaves partial AWS resources for manual inspection. Users must manually run the `cleanup` command to remove them.
5. **Multiple Cluster Support**: Not supported - the tool is designed for single-cluster-at-a-time installation

---

## Appendix: Step Mapping

This section maps the manual steps from `tasks/sts.md` to the tool's implementation:

| Manual Step | Tool Implementation |
|------------|-------------------|
| 1. Set RELEASE_IMAGE | Accept as CLI flag/config/prompt |
| 2. Extract credentials requests | `oc adm release extract --credentials-requests` |
| 3. Extract binaries | `oc adm release extract --command` + `oc image extract` |
| 4. Create install-config.yaml | Execute `openshift-install create install-config` |
| 5. Set credentialsMode | Append to install-config.yaml |
| 6. Create manifests | Execute `openshift-install create manifests` |
| 7A/B. Create AWS resources | Execute `ccoctl aws create-all` with optional `--create-private-s3-bucket` |
| 8. Copy manifests | `cp _output/manifests/* manifests/` |
| 9. Copy TLS | `cp -a _output/tls ./` |
| 10. Run installer | Execute `openshift-install create cluster` |
| Post-install | Execute verification commands |
