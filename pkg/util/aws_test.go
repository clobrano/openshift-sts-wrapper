package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadAWSCredentials(t *testing.T) {
	// Create a temporary credentials file
	tmpDir := t.TempDir()
	credentialsPath := filepath.Join(tmpDir, "credentials")

	credentialsContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[profile1]
aws_access_key_id = AKIAIOSFODNN7PROFILE1
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYPROFILE1
aws_session_token = FwoGZXIvYXdzEBQaDExampleSessionToken

[profile2]
aws_access_key_id = AKIAIOSFODNN7PROFILE2
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYPROFILE2
`

	if err := os.WriteFile(credentialsPath, []byte(credentialsContent), 0600); err != nil {
		t.Fatalf("Failed to create test credentials file: %v", err)
	}

	// Temporarily override the home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create .aws directory
	awsDir := filepath.Join(tmpDir, ".aws")
	if err := os.Mkdir(awsDir, 0755); err != nil {
		t.Fatalf("Failed to create .aws directory: %v", err)
	}

	// Move credentials file to .aws directory
	finalPath := filepath.Join(awsDir, "credentials")
	if err := os.Rename(credentialsPath, finalPath); err != nil {
		t.Fatalf("Failed to move credentials file: %v", err)
	}

	tests := []struct {
		name          string
		profile       string
		expectError   bool
		expectedKeyID string
		hasToken      bool
	}{
		{
			name:          "default profile",
			profile:       "default",
			expectError:   false,
			expectedKeyID: "AKIAIOSFODNN7EXAMPLE",
			hasToken:      false,
		},
		{
			name:          "empty profile defaults to default",
			profile:       "",
			expectError:   false,
			expectedKeyID: "AKIAIOSFODNN7EXAMPLE",
			hasToken:      false,
		},
		{
			name:          "profile1 with session token",
			profile:       "profile1",
			expectError:   false,
			expectedKeyID: "AKIAIOSFODNN7PROFILE1",
			hasToken:      true,
		},
		{
			name:          "profile2 without session token",
			profile:       "profile2",
			expectError:   false,
			expectedKeyID: "AKIAIOSFODNN7PROFILE2",
			hasToken:      false,
		},
		{
			name:        "nonexistent profile",
			profile:     "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := ReadAWSCredentials(tt.profile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if creds.AccessKeyID != tt.expectedKeyID {
				t.Errorf("Expected AccessKeyID %s, got %s", tt.expectedKeyID, creds.AccessKeyID)
			}

			if creds.SecretAccessKey == "" {
				t.Errorf("SecretAccessKey should not be empty")
			}

			if tt.hasToken && creds.SessionToken == "" {
				t.Errorf("Expected SessionToken to be set")
			}

			if !tt.hasToken && creds.SessionToken != "" {
				t.Errorf("Expected SessionToken to be empty, got %s", creds.SessionToken)
			}
		})
	}
}

func TestGetAWSEnvVars(t *testing.T) {
	// Create a temporary credentials file
	tmpDir := t.TempDir()
	awsDir := filepath.Join(tmpDir, ".aws")
	if err := os.Mkdir(awsDir, 0755); err != nil {
		t.Fatalf("Failed to create .aws directory: %v", err)
	}

	credentialsPath := filepath.Join(awsDir, "credentials")
	credentialsContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
aws_session_token = FwoGZXIvYXdzEBQaDExampleSessionToken
`

	if err := os.WriteFile(credentialsPath, []byte(credentialsContent), 0600); err != nil {
		t.Fatalf("Failed to create test credentials file: %v", err)
	}

	// Temporarily override the home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	envVars, err := GetAWSEnvVars("default")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(envVars) != 3 {
		t.Errorf("Expected 3 environment variables, got %d", len(envVars))
	}

	expectedVars := map[string]bool{
		"AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE":                         false,
		"AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY": false,
		"AWS_SESSION_TOKEN=FwoGZXIvYXdzEBQaDExampleSessionToken":         false,
	}

	for _, envVar := range envVars {
		if _, ok := expectedVars[envVar]; ok {
			expectedVars[envVar] = true
		}
	}

	for varName, found := range expectedVars {
		if !found {
			t.Errorf("Expected environment variable %s not found", varName)
		}
	}
}
