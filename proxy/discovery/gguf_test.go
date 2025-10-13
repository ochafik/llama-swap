package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInferNameFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "basic gguf file",
			filename: "llama-3.1-8b.gguf",
			expected: "llama-3.1-8b",
		},
		{
			name:     "with quantization Q4_K_M",
			filename: "Llama-3.1-8B-Instruct-Q4_K_M.gguf",
			expected: "Llama-3.1-8B-Instruct",
		},
		{
			name:     "with quantization Q5_K_S",
			filename: "model-Q5_K_S.gguf",
			expected: "model",
		},
		{
			name:     "with underscore separator",
			filename: "qwen2_7b_instruct_Q8_0.gguf",
			expected: "qwen2_7b_instruct",
		},
		{
			name:     "uppercase extension",
			filename: "model.GGUF",
			expected: "model",
		},
		{
			name:     "F16 quantization",
			filename: "phi-2-F16.gguf",
			expected: "phi-2",
		},
		{
			name:     "multiple quantization patterns",
			filename: "model-Q4_K_M-Q5_K_S.gguf",
			expected: "model-Q4_K_M", // Only removes from end
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferNameFromFilename(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScanCacheForGGUF(t *testing.T) {
	t.Run("empty cache directory", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "cache-test-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Set LLAMA_CACHE to our temp directory
		oldCache := os.Getenv("LLAMA_CACHE")
		os.Setenv("LLAMA_CACHE", tempDir)
		defer func() {
			if oldCache != "" {
				os.Setenv("LLAMA_CACHE", oldCache)
			} else {
				os.Unsetenv("LLAMA_CACHE")
			}
		}()

		files, err := ScanCacheForGGUF()
		assert.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("cache with gguf files", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "cache-test-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create some test files
		testFiles := []string{
			"model1.gguf",
			"model2.GGUF",
			"model3.txt",      // Should be ignored
			"notamodel",       // Should be ignored
			"model4.gguf.bak", // Should be ignored
		}

		for _, name := range testFiles {
			f, err := os.Create(filepath.Join(tempDir, name))
			assert.NoError(t, err)
			f.Close()
		}

		// Set LLAMA_CACHE to our temp directory
		oldCache := os.Getenv("LLAMA_CACHE")
		os.Setenv("LLAMA_CACHE", tempDir)
		defer func() {
			if oldCache != "" {
				os.Setenv("LLAMA_CACHE", oldCache)
			} else {
				os.Unsetenv("LLAMA_CACHE")
			}
		}()

		files, err := ScanCacheForGGUF()
		assert.NoError(t, err)
		assert.Len(t, files, 2) // Only model1.gguf and model2.GGUF

		// Verify file paths
		found := make(map[string]bool)
		for _, f := range files {
			basename := filepath.Base(f)
			found[basename] = true
		}

		assert.True(t, found["model1.gguf"])
		assert.True(t, found["model2.GGUF"])
		assert.False(t, found["model3.txt"])
	})

	t.Run("nonexistent cache directory", func(t *testing.T) {
		// Set LLAMA_CACHE to a non-existent directory
		oldCache := os.Getenv("LLAMA_CACHE")
		os.Setenv("LLAMA_CACHE", "/nonexistent/path/that/does/not/exist")
		defer func() {
			if oldCache != "" {
				os.Setenv("LLAMA_CACHE", oldCache)
			} else {
				os.Unsetenv("LLAMA_CACHE")
			}
		}()

		_, err := ScanCacheForGGUF()
		assert.NoError(t, err) // Non-existent cache is not an error
	})

	t.Run("cache is a file not directory", func(t *testing.T) {
		// Create a temporary file (not directory)
		tempFile, err := os.CreateTemp("", "cache-file-*")
		assert.NoError(t, err)
		tempFile.Close()
		defer os.Remove(tempFile.Name())

		// Set LLAMA_CACHE to the file
		oldCache := os.Getenv("LLAMA_CACHE")
		os.Setenv("LLAMA_CACHE", tempFile.Name())
		defer func() {
			if oldCache != "" {
				os.Setenv("LLAMA_CACHE", oldCache)
			} else {
				os.Unsetenv("LLAMA_CACHE")
			}
		}()

		_, err = ScanCacheForGGUF()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})
}

func TestExtractMetadata_ErrorCases(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		meta, err := ExtractMetadata("/nonexistent/file.gguf")
		assert.Error(t, err)
		assert.Nil(t, meta)
	})

	t.Run("not a gguf file", func(t *testing.T) {
		// Create a temporary text file
		tempFile, err := os.CreateTemp("", "not-gguf-*.txt")
		assert.NoError(t, err)
		defer os.Remove(tempFile.Name())

		tempFile.WriteString("This is not a GGUF file")
		tempFile.Close()

		meta, err := ExtractMetadata(tempFile.Name())
		assert.Error(t, err)
		assert.Nil(t, meta)
	})
}

func TestDiscoverModels(t *testing.T) {
	t.Run("empty cache", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "discover-test-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Set LLAMA_CACHE to our temp directory
		oldCache := os.Getenv("LLAMA_CACHE")
		os.Setenv("LLAMA_CACHE", tempDir)
		defer func() {
			if oldCache != "" {
				os.Setenv("LLAMA_CACHE", oldCache)
			} else {
				os.Unsetenv("LLAMA_CACHE")
			}
		}()

		models, err := DiscoverModels()
		assert.NoError(t, err)
		assert.Empty(t, models)
	})

	t.Run("nonexistent cache", func(t *testing.T) {
		// Set LLAMA_CACHE to a non-existent directory
		oldCache := os.Getenv("LLAMA_CACHE")
		os.Setenv("LLAMA_CACHE", "/nonexistent/discover/cache")
		defer func() {
			if oldCache != "" {
				os.Setenv("LLAMA_CACHE", oldCache)
			} else {
				os.Unsetenv("LLAMA_CACHE")
			}
		}()

		models, err := DiscoverModels()
		assert.NoError(t, err) // Should not error on non-existent cache
		assert.Empty(t, models)
	})
}

// Note: Full integration tests with actual GGUF files would require
// sample GGUF files to be checked into the repository or generated
// during test setup. These tests focus on the error handling and
// path manipulation logic that can be tested without real GGUF files.
