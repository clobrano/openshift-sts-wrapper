package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindSSHKeyPath searches for a file in ~/.ssh directory that contains the given SSH key content.
// If multiple files match, returns the first one found.
// Returns error if ~/.ssh doesn't exist or no matching file is found.
func FindSSHKeyPath(sshKeyContent string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if !FileExists(sshDir) {
		return "", fmt.Errorf("~/.ssh directory does not exist")
	}

	// Trim whitespace from the target content for comparison
	targetContent := strings.TrimSpace(sshKeyContent)

	// Read all files in ~/.ssh
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return "", fmt.Errorf("could not read ~/.ssh directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(sshDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			// Skip files we can't read
			continue
		}

		// Compare trimmed content
		if strings.TrimSpace(string(content)) == targetContent {
			return filePath, nil
		}
	}

	return "", fmt.Errorf("no matching SSH key file found in ~/.ssh")
}
