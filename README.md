# OpenShift STS Installation Wrapper

A CLI tool that automates the installation of OpenShift clusters with AWS Security Token Service (STS) authentication.

## Features

- **Automated Workflow**: Consolidates 10+ manual steps into a single command
- **Error Recovery**: Resume installations from where they left off
- **Flexible Configuration**: Support for CLI flags, config files, and environment variables
- **Interactive Guidance**: Clear progress indicators and error handling
- **Version-Aware**: Automatically handles multiple OpenShift versions

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

**Step 4 (Create install-config.yaml)**: Always runs interactively using `openshift-install create install-config`, which will prompt you for:
- SSH public key
- Platform (aws)
- Base domain
- Cluster name
- AWS region
- Pull secret

**Step 7 (Create AWS resources)**: Automatically reads `clusterName` and `awsRegion` from the install-config.yaml created in Step 4. You don't need to specify these in your configuration file unless you want to override the values from install-config.yaml.

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

# Optional: Fields below are automatically saved after Step 4
# If all are present (along with pullSecretPath), you'll be prompted to reuse them on subsequent runs
clusterName: my-cluster
awsRegion: us-east-2
baseDomain: example.com
sshKeyPath: /home/user/.ssh/id_rsa.pub
```

Then run:

```bash
openshift-sts-installer install
```

**Note:** After the first successful run of Step 4, the tool automatically saves `clusterName`, `awsRegion`, `baseDomain`, and `sshKey` to the config file. On subsequent runs, if all these fields plus `pullSecretPath` are present, you'll be prompted to reuse them instead of entering them interactively.

### Resume from Specific Step

If installation was interrupted:

```bash
openshift-sts-installer install --start-from-step=6
```

Step numbers:
1-2. Extract credentials requests
3. Extract binaries
4. Create install-config.yaml
5. Set credentialsMode
6. Create manifests
7. Create AWS resources
8-9. Copy files
10. Deploy cluster

### Cleanup After Failed Installation

The cleanup command removes all AWS resources created during installation:
- OpenShift infrastructure (EC2, VPCs, load balancers, DNS records) via `openshift-install destroy`
- IAM roles and S3 bucket created by ccoctl

```bash
# Complete cleanup (recommended - includes DNS records and infrastructure)
openshift-sts-installer cleanup \
  --cluster-name=my-cluster \
  --region=us-east-2 \
  --release-image=quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64

# Minimal cleanup (only IAM roles and S3 bucket, no infrastructure/DNS cleanup)
openshift-sts-installer cleanup \
  --cluster-name=my-cluster \
  --region=us-east-2
```

**Note:** If you provide `--release-image`, the cleanup will:
1. Run `openshift-install destroy cluster` (if state file exists) to remove all infrastructure and DNS records
2. Run `ccoctl aws delete` to remove IAM roles and S3 bucket

Without `--release-image`, only step 2 runs, leaving infrastructure and DNS records orphaned.

## Environment Variables

You can also configure via environment variables:

```bash
export OPENSHIFT_STS_RELEASE_IMAGE=quay.io/openshift-release-dev/ocp-release:4.12.0-x86_64
export OPENSHIFT_STS_CLUSTER_NAME=my-cluster
export OPENSHIFT_STS_AWS_REGION=us-east-2
export OPENSHIFT_STS_AWS_PROFILE=default
export OPENSHIFT_STS_PULL_SECRET_PATH=./pull-secret.json
export OPENSHIFT_STS_PRIVATE_BUCKET=true

openshift-sts-installer install
```

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
│   └── 4.12.0-x86_64/       # Version-specific artifacts
│       ├── bin/              # Extracted binaries
│       ├── credreqs/         # Credentials requests
│       ├── _output/          # ccoctl generated files
│       │   ├── manifests/
│       │   └── tls/
│       ├── install-config.yaml.backup  # Backup of install-config (before Step 6 consumes it)
│       └── install-config.yaml         # Created by Step 4, consumed by Step 6
├── manifests/                # Installation manifests (copied from _output)
├── tls/                      # TLS certificates (copied from _output)
└── pull-secret.json          # Pull secret
```

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
