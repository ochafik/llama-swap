package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abrander/gguf"
)

// ModelMetadata contains extracted information from a GGUF file
type ModelMetadata struct {
	FilePath        string // Absolute path to the GGUF file
	FileName        string // Base filename (e.g., "model.gguf")
	Architecture    string // Model architecture (e.g., "llama", "qwen2")
	Name            string // Human-readable model name
	SizeLabel       string // Size label (e.g., "8B", "70B")
	ContextLength   int    // Maximum context window size
	EmbeddingLength int    // Embedding dimension size
	Finetune        string // Finetune type (e.g., "Instruct", "Chat")
}

// ScanCacheForGGUF scans the llama.cpp cache directory for GGUF files
// and returns a list of discovered file paths.
func ScanCacheForGGUF() ([]string, error) {
	cacheDir, err := GetCacheDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	// Remove trailing slash for directory operations
	cacheDir = filepath.Clean(cacheDir)

	// Check if cache directory exists
	info, err := os.Stat(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // Empty cache is not an error
		}
		return nil, fmt.Errorf("failed to stat cache directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("cache path is not a directory: %s", cacheDir)
	}

	// Read directory contents
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	var ggufFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check for .gguf extension (case-insensitive)
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".gguf") {
			fullPath := filepath.Join(cacheDir, entry.Name())
			ggufFiles = append(ggufFiles, fullPath)
		}
	}

	return ggufFiles, nil
}

// ExtractMetadata parses a GGUF file and extracts relevant metadata
func ExtractMetadata(path string) (*ModelMetadata, error) {
	// Open the GGUF file
	g, err := gguf.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open GGUF file: %w", err)
	}

	meta := &ModelMetadata{
		FilePath: path,
		FileName: filepath.Base(path),
	}

	// Extract architecture (required)
	arch, err := g.Metadata.String("general.architecture")
	if err != nil {
		return nil, fmt.Errorf("missing general.architecture: %w", err)
	}
	meta.Architecture = arch

	// Extract optional fields with fallback values
	if name, err := g.Metadata.String("general.name"); err == nil {
		meta.Name = name
	}

	if sizeLabel, err := g.Metadata.String("general.size_label"); err == nil {
		meta.SizeLabel = sizeLabel
	}

	if finetune, err := g.Metadata.String("general.finetune"); err == nil {
		meta.Finetune = finetune
	}

	// Extract context length (architecture-specific key)
	contextKey := fmt.Sprintf("%s.context_length", arch)
	if ctx, err := g.Metadata.Int(contextKey); err == nil {
		meta.ContextLength = int(ctx)
	}

	// Extract embedding length (architecture-specific key)
	embeddingKey := fmt.Sprintf("%s.embedding_length", arch)
	if emb, err := g.Metadata.Int(embeddingKey); err == nil {
		meta.EmbeddingLength = int(emb)
	}

	// If name is still empty, try to infer from filename
	if meta.Name == "" {
		meta.Name = inferNameFromFilename(meta.FileName)
	}

	return meta, nil
}

// inferNameFromFilename extracts a reasonable model name from the filename
func inferNameFromFilename(filename string) string {
	// Remove .gguf extension
	name := strings.TrimSuffix(filename, ".gguf")
	name = strings.TrimSuffix(name, ".GGUF")

	// Remove common quantization suffixes (e.g., Q4_K_M, Q8_0, etc.)
	quantPatterns := []string{
		"-Q4_K_M", "-Q4_K_S", "-Q4_0", "-Q4_1",
		"-Q5_K_M", "-Q5_K_S", "-Q5_0", "-Q5_1",
		"-Q6_K", "-Q8_0", "-F16", "-F32",
		"_Q4_K_M", "_Q4_K_S", "_Q4_0", "_Q4_1",
		"_Q5_K_M", "_Q5_K_S", "_Q5_0", "_Q5_1",
		"_Q6_K", "_Q8_0", "_F16", "_F32",
	}

	for _, pattern := range quantPatterns {
		name = strings.TrimSuffix(name, pattern)
	}

	return name
}

// DiscoverModels scans the cache and extracts metadata from all GGUF files
func DiscoverModels() ([]*ModelMetadata, error) {
	ggufFiles, err := ScanCacheForGGUF()
	if err != nil {
		return nil, err
	}

	if len(ggufFiles) == 0 {
		return []*ModelMetadata{}, nil
	}

	var models []*ModelMetadata
	var failedFiles []string

	for _, filePath := range ggufFiles {
		meta, err := ExtractMetadata(filePath)
		if err != nil {
			// Log the error but continue processing other files
			failedFiles = append(failedFiles, fmt.Sprintf("%s: %v", filepath.Base(filePath), err))
			continue
		}
		models = append(models, meta)
	}

	// If some files failed but we have at least one successful model, that's okay
	if len(models) > 0 && len(failedFiles) > 0 {
		// Return models with a note about failures (could be logged by caller)
		return models, nil
	}

	// If all files failed, return an error
	if len(failedFiles) > 0 {
		return nil, fmt.Errorf("failed to parse any GGUF files: %s", strings.Join(failedFiles, "; "))
	}

	return models, nil
}
