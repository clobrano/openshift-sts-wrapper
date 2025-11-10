package util

import (
	"io"
	"os"
	"path/filepath"
)

// DirExistsWithFiles checks if a directory exists and contains at least one file
func DirExistsWithFiles(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}

	return len(entries) > 0
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FileContains checks if a file exists and contains the specified string
func FileContains(path string, needle string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return contains(string(content), needle)
}

func contains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && haystack != needle &&
		(haystack == needle || len(haystack) > len(needle) &&
			(haystack[0:len(needle)] == needle ||
				haystack[len(haystack)-len(needle):] == needle ||
				containsSubstring(haystack, needle)))
}

func containsSubstring(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// GetSharedBinaryPath returns the full path to a binary in the shared artifacts directory
func GetSharedBinaryPath(versionArch, binaryName string) string {
	return filepath.Join("artifacts", "shared", versionArch, "bin", binaryName)
}

// GetSharedCredReqsPath returns the path to the shared credentials requests directory
func GetSharedCredReqsPath(versionArch string) string {
	return filepath.Join("artifacts", "shared", versionArch, "credreqs")
}

// GetClusterPath returns the path to a cluster-specific subdirectory
func GetClusterPath(clusterName, subpath string) string {
	return filepath.Join("artifacts", "clusters", clusterName, subpath)
}

// GetInstallConfigPath returns the path to the install-config.yaml for a specific cluster
func GetInstallConfigPath(versionArch, clusterName string) string {
	return filepath.Join("artifacts", "clusters", clusterName, "install-config.yaml")
}

// Legacy path helpers for backward compatibility (deprecated)
// GetBinaryPath returns the full path to a binary in the version-specific artifacts directory
// Deprecated: Use GetSharedBinaryPath instead
func GetBinaryPath(versionArch, binaryName string) string {
	return filepath.Join("artifacts", versionArch, "bin", binaryName)
}

// GetCredReqsPath returns the path to the credentials requests directory
// Deprecated: Use GetSharedCredReqsPath instead
func GetCredReqsPath(versionArch string) string {
	return filepath.Join("artifacts", versionArch, "credreqs")
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
