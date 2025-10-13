package discovery

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FindLlamaServer attempts to find the llama-server executable.
// It searches in the following order:
// 1. LLAMA_SERVER_PATH environment variable
// 2. PATH environment variable (using exec.LookPath)
// 3. Common installation locations
//
// Returns the absolute path to llama-server or an error if not found.
func FindLlamaServer() (string, error) {
	// Priority 1: LLAMA_SERVER_PATH environment variable
	if serverPath := os.Getenv("LLAMA_SERVER_PATH"); serverPath != "" {
		// Verify the file exists and is executable
		if info, err := os.Stat(serverPath); err == nil {
			if info.IsDir() {
				return "", fmt.Errorf("LLAMA_SERVER_PATH points to a directory: %s", serverPath)
			}
			// Make path absolute
			absPath, err := filepath.Abs(serverPath)
			if err != nil {
				return "", fmt.Errorf("failed to get absolute path: %w", err)
			}
			return absPath, nil
		}
		return "", fmt.Errorf("LLAMA_SERVER_PATH file not found: %s", serverPath)
	}

	// Priority 2: Search PATH
	binaryName := "llama-server"
	if runtime.GOOS == "windows" {
		binaryName = "llama-server.exe"
	}

	if path, err := exec.LookPath(binaryName); err == nil {
		// Make path absolute
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		return absPath, nil
	}

	// Priority 3: Common installation locations
	commonLocations := getCommonServerLocations()
	for _, location := range commonLocations {
		fullPath := filepath.Join(location, binaryName)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				return "", fmt.Errorf("failed to get absolute path: %w", err)
			}
			return absPath, nil
		}
	}

	return "", fmt.Errorf("llama-server not found (checked PATH and common locations)")
}

// getCommonServerLocations returns a list of common locations where llama-server might be installed
func getCommonServerLocations() []string {
	homeDir, _ := os.UserHomeDir()

	locations := []string{
		"/usr/local/bin",
		"/usr/bin",
		"/opt/llama.cpp/bin",
	}

	// Add home directory locations
	if homeDir != "" {
		locations = append(locations,
			filepath.Join(homeDir, "llama.cpp", "build", "bin"),
			filepath.Join(homeDir, ".local", "bin"),
			filepath.Join(homeDir, "bin"),
		)
	}

	// Platform-specific additions
	switch runtime.GOOS {
	case "darwin":
		locations = append(locations,
			"/opt/homebrew/bin",
			"/usr/local/opt/llama.cpp/bin",
		)
	case "windows":
		programFiles := os.Getenv("ProgramFiles")
		if programFiles != "" {
			locations = append(locations,
				filepath.Join(programFiles, "llama.cpp", "bin"),
			)
		}
	}

	return locations
}
