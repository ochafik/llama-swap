package discovery

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCacheDirectory(t *testing.T) {
	tests := []struct {
		name          string
		setupEnv      func() (cleanup func())
		expectedPath  func() string
		expectError   bool
		errorContains string
	}{
		{
			name: "LLAMA_CACHE takes priority",
			setupEnv: func() func() {
				oldCache := os.Getenv("LLAMA_CACHE")
				os.Setenv("LLAMA_CACHE", "/custom/cache")
				return func() {
					if oldCache != "" {
						os.Setenv("LLAMA_CACHE", oldCache)
					} else {
						os.Unsetenv("LLAMA_CACHE")
					}
				}
			},
			expectedPath: func() string {
				return "/custom/cache" + string(filepath.Separator)
			},
		},
		{
			name: "XDG_CACHE_HOME on Linux",
			setupEnv: func() func() {
				if runtime.GOOS != "linux" {
					return func() {}
				}
				oldCache := os.Getenv("LLAMA_CACHE")
				oldXDG := os.Getenv("XDG_CACHE_HOME")
				os.Unsetenv("LLAMA_CACHE")
				os.Setenv("XDG_CACHE_HOME", "/xdg/cache")
				return func() {
					if oldCache != "" {
						os.Setenv("LLAMA_CACHE", oldCache)
					}
					if oldXDG != "" {
						os.Setenv("XDG_CACHE_HOME", oldXDG)
					} else {
						os.Unsetenv("XDG_CACHE_HOME")
					}
				}
			},
			expectedPath: func() string {
				if runtime.GOOS != "linux" {
					return "" // Skip on non-Linux
				}
				return filepath.Join("/xdg/cache", "llama.cpp") + string(filepath.Separator)
			},
		},
		{
			name: "HOME directory fallback",
			setupEnv: func() func() {
				oldCache := os.Getenv("LLAMA_CACHE")
				oldXDG := os.Getenv("XDG_CACHE_HOME")
				os.Unsetenv("LLAMA_CACHE")
				os.Unsetenv("XDG_CACHE_HOME")
				return func() {
					if oldCache != "" {
						os.Setenv("LLAMA_CACHE", oldCache)
					}
					if oldXDG != "" {
						os.Setenv("XDG_CACHE_HOME", oldXDG)
					}
				}
			},
			expectedPath: func() string {
				home := os.Getenv("HOME")
				if home == "" {
					homeDir, _ := os.UserHomeDir()
					home = homeDir
				}

				switch runtime.GOOS {
				case "linux", "freebsd", "openbsd", "aix":
					return filepath.Join(home, ".cache", "llama.cpp") + string(filepath.Separator)
				case "darwin":
					return filepath.Join(home, "Library", "Caches", "llama.cpp") + string(filepath.Separator)
				case "windows":
					// On Windows, should use LOCALAPPDATA, not HOME
					return ""
				default:
					return ""
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupEnv()
			defer cleanup()

			expectedPath := tt.expectedPath()
			if expectedPath == "" && !tt.expectError {
				t.Skip("Test not applicable for this platform")
			}

			result, err := GetCacheDirectory()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expectedPath, result)
			}
		})
	}
}

func TestGetCacheDirectory_PlatformSpecific(t *testing.T) {
	// Clear LLAMA_CACHE to test platform-specific logic
	oldCache := os.Getenv("LLAMA_CACHE")
	os.Unsetenv("LLAMA_CACHE")
	defer func() {
		if oldCache != "" {
			os.Setenv("LLAMA_CACHE", oldCache)
		}
	}()

	result, err := GetCacheDirectory()
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify trailing slash
	assert.True(t, os.IsPathSeparator(result[len(result)-1]), "should have trailing separator")

	// Verify platform-specific path components
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd", "aix":
		assert.Contains(t, result, ".cache"+string(filepath.Separator)+"llama.cpp")
	case "darwin":
		assert.Contains(t, result, "Library"+string(filepath.Separator)+"Caches"+string(filepath.Separator)+"llama.cpp")
	case "windows":
		assert.Contains(t, result, "llama.cpp")
	}
}

func TestGetCacheFile(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		expectError   bool
		errorContains string
	}{
		{
			name:     "valid filename",
			filename: "model.gguf",
		},
		{
			name:          "filename with path separator",
			filename:      "subdir/model.gguf",
			expectError:   true,
			errorContains: "must not contain directory separators",
		},
		{
			name:          "filename with backslash",
			filename:      "subdir\\model.gguf",
			expectError:   true,
			errorContains: "must not contain directory separators",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip backslash test on non-Windows systems
			if tt.name == "filename with backslash" && runtime.GOOS != "windows" {
				t.Skip("Backslash is not a path separator on this platform")
			}

			result, err := GetCacheFile(tt.filename)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tt.filename)

				// Verify the cache directory was created
				dir := filepath.Dir(result)
				info, err := os.Stat(dir)
				if err == nil {
					assert.True(t, info.IsDir())
				}
			}
		})
	}
}

func TestEnsureTrailingSlash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "path without trailing slash",
			input:    "/path/to/dir",
			expected: "/path/to/dir" + string(filepath.Separator),
		},
		{
			name:     "path with trailing slash",
			input:    "/path/to/dir" + string(filepath.Separator),
			expected: "/path/to/dir" + string(filepath.Separator),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureTrailingSlash(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
