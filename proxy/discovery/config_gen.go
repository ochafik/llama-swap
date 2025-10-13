package discovery

import (
	"regexp"
	"strings"
)

// GenerateModelID creates a sanitized model ID from metadata
func GenerateModelID(meta *ModelMetadata) string {
	parts := []string{}

	if meta.Name != "" {
		parts = append(parts, meta.Name)
	}

	if meta.SizeLabel != "" {
		parts = append(parts, meta.SizeLabel)
	}

	if meta.Finetune != "" {
		parts = append(parts, meta.Finetune)
	}

	// If we have no parts, fall back to filename
	if len(parts) == 0 {
		// Use filename without extension
		name := strings.TrimSuffix(meta.FileName, ".gguf")
		name = strings.TrimSuffix(name, ".GGUF")
		parts = append(parts, name)
	}

	// Join parts and sanitize
	id := strings.Join(parts, "-")

	// Convert to lowercase
	id = strings.ToLower(id)

	// Replace spaces and underscores with hyphens
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")

	// Remove any characters that aren't alphanumeric, hyphen, or period
	id = sanitizeModelID(id)

	// Collapse multiple hyphens
	re := regexp.MustCompile(`-+`)
	id = re.ReplaceAllString(id, "-")

	// Trim leading/trailing hyphens
	id = strings.Trim(id, "-")

	return id
}

// sanitizeModelID removes invalid characters from a model ID
func sanitizeModelID(id string) string {
	var result strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// GenerateDisplayName creates a human-readable display name from metadata
func GenerateDisplayName(meta *ModelMetadata) string {
	parts := []string{}

	if meta.Name != "" {
		parts = append(parts, meta.Name)
	}

	if meta.SizeLabel != "" {
		parts = append(parts, meta.SizeLabel)
	}

	if meta.Finetune != "" {
		parts = append(parts, meta.Finetune)
	}

	if len(parts) == 0 {
		// Fall back to filename
		return inferNameFromFilename(meta.FileName)
	}

	return strings.Join(parts, " ")
}

// Note: GenerateModelConfig and GenerateConfig have been moved to
// proxy/config/autodiscovery.go to avoid import cycles.

// DeduplicateModels removes duplicate models based on file name similarity.
// It keeps the first occurrence of each model.
func DeduplicateModels(models []*ModelMetadata) []*ModelMetadata {
	if len(models) <= 1 {
		return models
	}

	seen := make(map[string]bool)
	var result []*ModelMetadata

	for _, model := range models {
		// Create a base key from the model name (without quantization suffix)
		baseKey := strings.ToLower(inferNameFromFilename(model.FileName))

		if !seen[baseKey] {
			seen[baseKey] = true
			result = append(result, model)
		}
	}

	return result
}
