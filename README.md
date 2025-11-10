# OpenShift STS Installation Wrapper

A CLI tool that automates the installation of OpenShift clusters with AWS Security Token Service (STS) authentication.

## Features

- **Automated Workflow**: Consolidates 10+ manual steps into a single command
- **Error Recovery**: Resume installations from where they left off
- **Flexible Configuration**: Support for CLI flags, config files, and environment variables
- **Interactive Guidance**: Clear progress indicators and error handling
- **Version-Aware**: Automatically handles multiple OpenShift versions
- **Multi-Cluster Support**: Deploy multiple clusters with the same OpenShift version without redundant downloads
- **Efficient Resource Usage**: Shared binaries and credential requests across clusters of the same version

## Installation

### From Source

```bash
git clone https://gitlab.cee.redhat.com/clobrano/ccoctl-sso.git
cd ccoctl-sso
make build
sudo make install
```

## Prerequisites

- `oc` (OpenShift CLI) must be installed and in your PATH
- AWS credentials configured in `~/.aws/credentials`
- Pull secret from Red Hat (will be prompted if not provided)

### AWS Credentials

The tool automatically reads AWS credentials from `~/.aws/credentials` based on the specified profile (defaults to `default`). The credentials are used for:
- Creating install-config.yaml (via openshift-install)
- Creating AWS resources (S3, IAM, OIDC via ccoctl)
- Deploying the cluster

You can specify a different profile using:
- CLI flag: `--aws-profile=my-profile`
- Config file: `awsProfile: my-profile`
- Environment variable: `OPENSHIFT_STS_AWS_PROFILE=my-profile`

If credentials cannot be read from the profile, the tool will proceed without setting AWS environment variables, relying on the default AWS credential chain.

### Configuration Notes

**Cluster Name Requirement**: The `--cluster-name` flag is **required** for both `install` and `cleanup` commands. It must be provided via the CLI flag and cannot be loaded from configuration files or environment variables. This ensures clear cluster identification and prevents configuration conflicts.

**Step 4 (Create install-config.yaml)**: Runs interactively using `openshift-install create install-config`, which will prompt you for:
- SSH public key
- Platform (aws)
- Base domain
- Cluster name (must match the `--cluster-name` flag)
- AWS region
- Pull secret

**Step 7 (Create AWS resources)**: Uses the cluster name from the `--cluster-name` flag. AWS region can be specified via config file/env or will be extracted from install-config.yaml.

## Usage

### Full Installation

```bash
openshift-sts-installer install \
  --release-image=quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64 \
  --cluster-name=my-cluster \
  --region=us-east-2 \
  --pull-secret=./pull-secret.json \
  --aws-profile=default
```

### With Private S3 Bucket

```bash
openshift-sts-installer install \
  --release-image=quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64 \
  --cluster-name=my-cluster \
  --region=us-east-2 \
  --pull-secret=./pull-secret.json \
  --aws-profile=default \
  --private-bucket
```

### Using a Configuration File

Create `openshift-sts-installer.yaml`:

```yaml
releaseImage: quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64
awsProfile: default
pullSecretPath: ./pull-secret.json
privateBucket: false
awsRegion: us-east-2
baseDomain: example.com
sshKeyPath: /home/user/.ssh/id_rsa.pub

# Note: clusterName CANNOT be set in config files - must use --cluster-name flag
```

Then run:

```bash
openshift-sts-installer install --cluster-name=my-cluster
```

**Important:** The `--cluster-name` flag is always required, even when using a config file.

### Resume from Specific Step

If installation was interrupted:

```bash
openshift-sts-installer install --cluster-name=my-cluster --start-from-step=6
```

Step numbers:
1. Extract credentials requests
2. Extract openshift-install binary
3. Extract ccoctl binary
4. Create install-config.yaml
5. Set credentialsMode
6. Create manifests
7. Create AWS resources
8. Copy manifests
9. Copy TLS files
10. Deploy cluster
11. Verify installation

### Cleanup After Failed Installation

The cleanup command removes all AWS resources created during installation:
- OpenShift infrastructure (EC2, VPCs, load balancers, DNS records) via `openshift-install destroy`
- IAM roles and S3 bucket created by ccoctl

**Option 1: Using --from-artifacts (reads cluster info from metadata.json):**

```bash
# Complete cleanup (infrastructure + IAM/S3)
openshift-sts-installer cleanup \
  --from-artifacts=artifacts/clusters/my-cluster \
  --release-image=quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64

# Minimal cleanup (only IAM roles and S3 bucket)
openshift-sts-installer cleanup \
  --from-artifacts=artifacts/clusters/my-cluster
```

**Option 2: Using explicit flags:**

```bash
# Complete cleanup (infrastructure + IAM/S3)
openshift-sts-installer cleanup \
  --cluster-name=my-cluster \
  --region=us-east-2 \
  --release-image=quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64

# Minimal cleanup (only IAM roles and S3 bucket)
openshift-sts-installer cleanup \
  --cluster-name=my-cluster \
  --region=us-east-2
```

**Note:**
- If you provide `--release-image`, the cleanup will:
  1. Run `openshift-install destroy cluster` (if state file exists) to remove all infrastructure and DNS records
  2. Run `ccoctl aws delete` to remove IAM roles and S3 bucket
- Without `--release-image`, only step 2 runs, leaving infrastructure and DNS records orphaned

## Environment Variables

You can also configure via environment variables (except cluster name):

```bash
export OPENSHIFT_STS_RELEASE_IMAGE=quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64
export OPENSHIFT_STS_AWS_REGION=us-east-2
export OPENSHIFT_STS_AWS_PROFILE=default
export OPENSHIFT_STS_PULL_SECRET_PATH=./pull-secret.json
export OPENSHIFT_STS_PRIVATE_BUCKET=true

# Cluster name must still be provided via flag
openshift-sts-installer install --cluster-name=my-cluster
```

**Note:** `OPENSHIFT_STS_CLUSTER_NAME` environment variable is **not supported**. You must use the `--cluster-name` flag.

## Configuration Priority

Configuration sources are merged with the following priority (highest to lowest):

1. CLI flags
2. Configuration file
3. Environment variables
4. Interactive prompts

## Directory Structure

The tool creates the following directory structure:

```
./
├── artifacts/
│   ├── shared/                        # Shared artifacts across clusters
│   │   └── 4.12.0-x86_64/             # Version-specific shared artifacts
│   │       ├── bin/                   # Extracted binaries (openshift-install, ccoctl)
│   │       └── credreqs/              # Credentials requests
│   └── clusters/                      # Cluster-specific artifacts
│       ├── my-cluster/                # Per-cluster directory
│       │   ├── install-config.yaml   # Created by Step 4, consumed by Step 6
│       │   ├── install-config.yaml.backup  # Backup (before Step 6 consumes it)
│       │   ├── ccoctl-output/        # Temporary ccoctl output (deleted after Step 9)
│       │   ├── manifests/            # Installation manifests
│       │   ├── tls/                  # TLS certificates
│       │   └── auth/                 # Kubeconfig and credentials
│       └── another-cluster/          # Another cluster (same version, different name)
│           └── ...
└── pull-secret.json                   # Pull secret
```

**Benefits of this structure:**
- **No redundant downloads**: Binaries are downloaded once per version and shared across all clusters
- **Cluster isolation**: Each cluster has its own configuration, preventing overwrites
- **Concurrent deployments**: Multiple clusters of the same version can be deployed simultaneously
- **Clear organization**: Shared vs cluster-specific artifacts are clearly separated

## Verbosity Control

```bash
# Quiet mode (errors only)
openshift-sts-installer install --quiet

# Verbose mode (detailed output)
openshift-sts-installer install --verbose
```

## Development

### Running Tests

```bash
make test
```

### Test Coverage

```bash
make test-coverage
```

### Code Quality

```bash
make check  # Runs fmt, vet, and test
```

### Building

```bash
make build
```

## Troubleshooting

### Pull Secret Issues

If you don't have a pull secret, the tool will:
1. Display a message with the download URL
2. Attempt to open your browser to the Red Hat portal
3. Wait for you to provide the path to the downloaded file

### Step Detection

The tool automatically detects completed steps by checking for:
- Existence of directories and files
- Content of configuration files
- Presence of artifacts

If detection fails, use `--start-from-step` to manually specify where to resume.

### AWS Permissions

The tool does not validate AWS permissions before starting. If you encounter AWS errors during execution, verify that your AWS credentials have the required permissions for:
- S3 bucket creation
- IAM role/policy creation
- OIDC provider creation

## License

MIT

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
