package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AWSCredentials holds AWS credentials from the credentials file
type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// ReadAWSCredentials reads AWS credentials from ~/.aws/credentials for a given profile
func ReadAWSCredentials(profile string) (*AWSCredentials, error) {
	if profile == "" {
		profile = "default"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	credentialsPath := filepath.Join(homeDir, ".aws", "credentials")
	file, err := os.Open(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open credentials file: %w", err)
	}
	defer file.Close()

	creds := &AWSCredentials{}
	inTargetSection := false
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionName := strings.Trim(line, "[]")
			inTargetSection = (sectionName == profile)
			continue
		}

		// If we're in the target section, read the credentials
		if inTargetSection {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "aws_access_key_id":
				creds.AccessKeyID = value
			case "aws_secret_access_key":
				creds.SecretAccessKey = value
			case "aws_session_token":
				creds.SessionToken = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading credentials file: %w", err)
	}

	// Validate that we found at least the required credentials
	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return nil, fmt.Errorf("profile '%s' not found or missing required credentials", profile)
	}

	return creds, nil
}

// GetAWSEnvVars returns environment variables for AWS credentials
func GetAWSEnvVars(profile string) ([]string, error) {
	// TODO: intergrate it with LoadFromEnv. The source of AWS credentials must be transparent to the users, they shall be able to set env variables or rely on aws-credential file created by aws-saml.py as they like
	creds, err := ReadAWSCredentials(profile)
	if err != nil {
		return nil, err
	}

	envVars := []string{
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", creds.AccessKeyID),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", creds.SecretAccessKey),
	}

	if creds.SessionToken != "" {
		envVars = append(envVars, fmt.Sprintf("AWS_SESSION_TOKEN=%s", creds.SessionToken))
	}

	return envVars, nil
}

// ValidateAWSCredentials checks if AWS credentials are valid and not expired
// by making a simple STS GetCallerIdentity API call
func ValidateAWSCredentials(profile string) error {
	// Try to get credentials for the profile
	envVars, err := GetAWSEnvVars(profile)
	if err != nil {
		return fmt.Errorf("failed to read credentials for profile '%s': %w", profile, err)
	}

	// Run aws sts get-caller-identity to validate credentials
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", profile)

	// Set environment with credentials
	cmd.Env = append(os.Environ(), envVars...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if strings.Contains(outputStr, "ExpiredToken") || strings.Contains(outputStr, "expired") {
			return fmt.Errorf("AWS credentials for profile '%s' have expired. Please refresh your credentials", profile)
		}
		if strings.Contains(outputStr, "InvalidClientTokenId") {
			return fmt.Errorf("AWS credentials for profile '%s' are invalid", profile)
		}
		return fmt.Errorf("failed to validate AWS credentials for profile '%s': %s", profile, outputStr)
	}

	return nil
}
