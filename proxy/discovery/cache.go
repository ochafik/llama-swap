package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// GetCacheDirectory returns the cache directory path for llama.cpp models,
// following the same logic as llama.cpp's fs_get_cache_directory function.
//
// Priority order:
// 1. LLAMA_CACHE environment variable (if set, used directly)
// 2. Platform-specific cache directories:
//    - Linux/Unix: $XDG_CACHE_HOME/llama.cpp or $HOME/.cache/llama.cpp
//    - macOS: $HOME/Library/Caches/llama.cpp
//    - Windows: %LOCALAPPDATA%\llama.cpp
func GetCacheDirectory() (string, error) {
	// Priority 1: LLAMA_CACHE environment variable
	if cacheDir := os.Getenv("LLAMA_CACHE"); cacheDir != "" {
		return ensureTrailingSlash(cacheDir), nil
	}

	var baseDir string

	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd", "aix":
		// Priority 2: XDG_CACHE_HOME
		if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
			baseDir = xdgCache
		} else if home := os.Getenv("HOME"); home != "" {
			// Priority 3: $HOME/.cache
			baseDir = filepath.Join(home, ".cache")
		} else {
			// Priority 4: Try to get home directory from user database
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to find HOME directory: %w", err)
			}
			baseDir = filepath.Join(homeDir, ".cache")
		}
		baseDir = filepath.Join(baseDir, "llama.cpp")

	case "darwin":
		home := os.Getenv("HOME")
		if home == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to find HOME directory: %w", err)
			}
			home = homeDir
		}
		baseDir = filepath.Join(home, "Library", "Caches", "llama.cpp")

	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", fmt.Errorf("LOCALAPPDATA environment variable not set")
		}
		baseDir = filepath.Join(localAppData, "llama.cpp")

	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return ensureTrailingSlash(baseDir), nil
}

// ensureTrailingSlash adds a trailing path separator if not present
func ensureTrailingSlash(path string) string {
	if path != "" && !os.IsPathSeparator(path[len(path)-1]) {
		return path + string(filepath.Separator)
	}
	return path
}

// GetCacheFile returns the full path to a file in the cache directory.
// The filename must not contain directory separators.
// Creates the cache directory if it doesn't exist.
func GetCacheFile(filename string) (string, error) {
	// Validate that filename doesn't contain path separators
	if filepath.Base(filename) != filename {
		return "", fmt.Errorf("filename must not contain directory separators: %s", filename)
	}

	cacheDir, err := GetCacheDirectory()
	if err != nil {
		return "", err
	}

	// Remove trailing slash for os.MkdirAll
	cacheDir = filepath.Clean(cacheDir)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	return filepath.Join(cacheDir, filename), nil
}
