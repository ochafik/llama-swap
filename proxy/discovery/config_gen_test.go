package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateModelID(t *testing.T) {
	tests := []struct {
		name     string
		meta     *ModelMetadata
		expected string
	}{
		{
			name: "full metadata",
			meta: &ModelMetadata{
				Name:      "LLaMA 3.1",
				SizeLabel: "8B",
				Finetune:  "Instruct",
				FileName:  "model.gguf",
			},
			expected: "llama-3.1-8b-instruct",
		},
		{
			name: "no finetune",
			meta: &ModelMetadata{
				Name:      "Qwen2",
				SizeLabel: "7B",
				FileName:  "model.gguf",
			},
			expected: "qwen2-7b",
		},
		{
			name: "name with spaces",
			meta: &ModelMetadata{
				Name:      "Phi Two",
				SizeLabel: "2B",
				FileName:  "model.gguf",
			},
			expected: "phi-two-2b",
		},
		{
			name: "name with underscores",
			meta: &ModelMetadata{
				Name:      "model_name_test",
				SizeLabel: "1B",
				FileName:  "model.gguf",
			},
			expected: "model-name-test-1b",
		},
		{
			name: "fallback to filename",
			meta: &ModelMetadata{
				FileName: "custom-model-Q4_K_M.gguf",
			},
			expected: "custom-model-q4-k-m",
		},
		{
			name: "special characters removed",
			meta: &ModelMetadata{
				Name:     "Model@Name#Test!",
				FileName: "model.gguf",
			},
			expected: "modelnametest",
		},
		{
			name: "multiple hyphens collapsed",
			meta: &ModelMetadata{
				Name:     "Model--Name---Test",
				FileName: "model.gguf",
			},
			expected: "model-name-test",
		},
		{
			name: "version numbers preserved",
			meta: &ModelMetadata{
				Name:      "LLaMA 3.1.5",
				SizeLabel: "70B",
				FileName:  "model.gguf",
			},
			expected: "llama-3.1.5-70b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateModelID(tt.meta)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		meta     *ModelMetadata
		expected string
	}{
		{
			name: "full metadata",
			meta: &ModelMetadata{
				Name:      "LLaMA 3.1",
				SizeLabel: "8B",
				Finetune:  "Instruct",
				FileName:  "model.gguf",
			},
			expected: "LLaMA 3.1 8B Instruct",
		},
		{
			name: "no finetune",
			meta: &ModelMetadata{
				Name:      "Qwen2",
				SizeLabel: "7B",
				FileName:  "model.gguf",
			},
			expected: "Qwen2 7B",
		},
		{
			name: "fallback to filename",
			meta: &ModelMetadata{
				FileName: "custom-model-Q4_K_M.gguf",
			},
			expected: "custom-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateDisplayName(tt.meta)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Note: GenerateModelConfig and GenerateConfig tests have been moved to
// proxy/config/autodiscovery_test.go since those functions are now in the config package

func TestDeduplicateModels(t *testing.T) {
	tests := []struct {
		name     string
		models   []*ModelMetadata
		expected int
	}{
		{
			name:     "empty list",
			models:   []*ModelMetadata{},
			expected: 0,
		},
		{
			name: "single model",
			models: []*ModelMetadata{
				{FileName: "model.gguf", Name: "Model"},
			},
			expected: 1,
		},
		{
			name: "different models",
			models: []*ModelMetadata{
				{FileName: "model1.gguf", Name: "Model One"},
				{FileName: "model2.gguf", Name: "Model Two"},
			},
			expected: 2,
		},
		{
			name: "duplicate quantizations",
			models: []*ModelMetadata{
				{FileName: "llama-3-8b-Q4_K_M.gguf", Name: "LLaMA 3"},
				{FileName: "llama-3-8b-Q5_K_S.gguf", Name: "LLaMA 3"},
				{FileName: "llama-3-8b-Q8_0.gguf", Name: "LLaMA 3"},
			},
			expected: 1, // Should keep only first
		},
		{
			name: "similar names different sizes",
			models: []*ModelMetadata{
				{FileName: "model-7b-Q4_K_M.gguf", Name: "Model"},
				{FileName: "model-13b-Q4_K_M.gguf", Name: "Model"},
			},
			expected: 2, // Different base names (model-7b vs model-13b)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeduplicateModels(tt.models)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestSanitizeModelID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "alphanumeric and hyphens",
			input:    "model-123-test",
			expected: "model-123-test",
		},
		{
			name:     "special characters removed",
			input:    "model@#$test!",
			expected: "modeltest",
		},
		{
			name:     "uppercase converted elsewhere",
			input:    "abc123-xyz",
			expected: "abc123-xyz",
		},
		{
			name:     "periods preserved",
			input:    "model.3.1.5",
			expected: "model.3.1.5",
		},
		{
			name:     "underscores removed",
			input:    "model_test",
			expected: "modeltest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeModelID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
