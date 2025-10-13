package discovery

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindLlamaServer(t *testing.T) {
	t.Run("LLAMA_SERVER_PATH environment variable", func(t *testing.T) {
		// Create a temporary file to act as llama-server
		tempFile, err := os.CreateTemp("", "llama-server-*")
		assert.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		// Set LLAMA_SERVER_PATH
		oldPath := os.Getenv("LLAMA_SERVER_PATH")
		os.Setenv("LLAMA_SERVER_PATH", tempFile.Name())
		defer func() {
			if oldPath != "" {
				os.Setenv("LLAMA_SERVER_PATH", oldPath)
			} else {
				os.Unsetenv("LLAMA_SERVER_PATH")
			}
		}()

		path, err := FindLlamaServer()
		assert.NoError(t, err)

		// Should return absolute path
		assert.True(t, filepath.IsAbs(path))

		// Should be the same file
		absTemp, _ := filepath.Abs(tempFile.Name())
		assert.Equal(t, absTemp, path)
	})

	t.Run("LLAMA_SERVER_PATH points to directory", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "llama-server-dir-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Set LLAMA_SERVER_PATH to directory
		oldPath := os.Getenv("LLAMA_SERVER_PATH")
		os.Setenv("LLAMA_SERVER_PATH", tempDir)
		defer func() {
			if oldPath != "" {
				os.Setenv("LLAMA_SERVER_PATH", oldPath)
			} else {
				os.Unsetenv("LLAMA_SERVER_PATH")
			}
		}()

		path, err := FindLlamaServer()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "points to a directory")
		assert.Empty(t, path)
	})

	t.Run("LLAMA_SERVER_PATH file not found", func(t *testing.T) {
		// Set LLAMA_SERVER_PATH to non-existent file
		oldPath := os.Getenv("LLAMA_SERVER_PATH")
		os.Setenv("LLAMA_SERVER_PATH", "/nonexistent/llama-server")
		defer func() {
			if oldPath != "" {
				os.Setenv("LLAMA_SERVER_PATH", oldPath)
			} else {
				os.Unsetenv("LLAMA_SERVER_PATH")
			}
		}()

		path, err := FindLlamaServer()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
		assert.Empty(t, path)
	})

	t.Run("fallback when not found", func(t *testing.T) {
		// Clear LLAMA_SERVER_PATH
		oldPath := os.Getenv("LLAMA_SERVER_PATH")
		os.Unsetenv("LLAMA_SERVER_PATH")
		defer func() {
			if oldPath != "" {
				os.Setenv("LLAMA_SERVER_PATH", oldPath)
			}
		}()

		// This test will succeed if llama-server is in PATH,
		// otherwise it will fail with "not found" error
		path, err := FindLlamaServer()

		if err != nil {
			// Expected if llama-server is not installed
			assert.Contains(t, err.Error(), "not found")
		} else {
			// If found, should be absolute path
			assert.True(t, filepath.IsAbs(path))
		}
	})
}

func TestGetCommonServerLocations(t *testing.T) {
	locations := getCommonServerLocations()

	// Should return non-empty list
	assert.NotEmpty(t, locations)

	// Should include /usr/local/bin on Unix-like systems
	if runtime.GOOS != "windows" {
		assert.Contains(t, locations, "/usr/local/bin")
	}

	// All paths should be absolute
	for _, loc := range locations {
		assert.True(t, filepath.IsAbs(loc), "location should be absolute: %s", loc)
	}

	// Should include home directory paths if HOME is set
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		found := false
		expectedPath := filepath.Join(homeDir, "llama.cpp", "build", "bin")
		for _, loc := range locations {
			if loc == expectedPath {
				found = true
				break
			}
		}
		assert.True(t, found, "should include home directory llama.cpp path")
	}
}

func TestFindLlamaServer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("create mock server in temp directory", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "llama-bin-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a mock llama-server file
		binaryName := "llama-server"
		if runtime.GOOS == "windows" {
			binaryName = "llama-server.exe"
		}

		serverPath := filepath.Join(tempDir, binaryName)
		f, err := os.Create(serverPath)
		assert.NoError(t, err)
		f.Close()

		// Make it executable on Unix
		if runtime.GOOS != "windows" {
			os.Chmod(serverPath, 0755)
		}

		// Set LLAMA_SERVER_PATH
		oldPath := os.Getenv("LLAMA_SERVER_PATH")
		os.Setenv("LLAMA_SERVER_PATH", serverPath)
		defer func() {
			if oldPath != "" {
				os.Setenv("LLAMA_SERVER_PATH", oldPath)
			} else {
				os.Unsetenv("LLAMA_SERVER_PATH")
			}
		}()

		// Should find the mock server
		path, err := FindLlamaServer()
		assert.NoError(t, err)
		assert.True(t, filepath.IsAbs(path))

		absServer, _ := filepath.Abs(serverPath)
		assert.Equal(t, absServer, path)
	})
}
