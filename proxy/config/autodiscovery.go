package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mostlygeek/llama-swap/proxy/discovery"
)

// generateModelConfig creates a ModelConfig from discovered metadata
func generateModelConfig(meta *discovery.ModelMetadata, serverPath string) (ModelConfig, error) {
	if serverPath == "" {
		return ModelConfig{}, fmt.Errorf("server path cannot be empty")
	}

	// Build the command
	cmdParts := []string{
		serverPath,
		"--model", meta.FilePath,
		"--port", "${PORT}",
	}

	// Add context size if available
	if meta.ContextLength > 0 {
		cmdParts = append(cmdParts, "--ctx-size", fmt.Sprintf("%d", meta.ContextLength))
	}

	cmd := strings.Join(cmdParts, " ")

	// Create the model config
	modelConfig := ModelConfig{
		Cmd:  cmd,
		Name: discovery.GenerateDisplayName(meta),
	}

	// Generate description
	desc := fmt.Sprintf("Auto-discovered %s model", meta.Architecture)
	if meta.SizeLabel != "" {
		desc = fmt.Sprintf("Auto-discovered %s %s model", meta.Architecture, meta.SizeLabel)
	}
	modelConfig.Description = desc

	return modelConfig, nil
}

// generateConfig creates a complete Config from discovered models
func generateConfig(models []*discovery.ModelMetadata, serverPath string) (*Config, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("no models provided")
	}

	if serverPath == "" {
		return nil, fmt.Errorf("server path cannot be empty")
	}

	// Create config with defaults
	cfg := &Config{
		HealthCheckTimeout: 120,
		StartPort:          5800,
		LogLevel:           "info",
		MetricsMaxInMemory: 1000,
		Models:             make(map[string]ModelConfig),
	}

	// Track used IDs to handle duplicates
	usedIDs := make(map[string]int)

	for _, meta := range models {
		baseID := discovery.GenerateModelID(meta)

		// Handle duplicate IDs
		id := baseID
		if count, exists := usedIDs[baseID]; exists {
			// Add numeric suffix for duplicates
			id = fmt.Sprintf("%s-%d", baseID, count+1)
			usedIDs[baseID] = count + 1
		} else {
			usedIDs[baseID] = 0
		}

		// Generate model config
		modelConfig, err := generateModelConfig(meta, serverPath)
		if err != nil {
			return nil, fmt.Errorf("failed to generate config for %s: %w", id, err)
		}

		cfg.Models[id] = modelConfig
	}

	return cfg, nil
}

// LoadConfigOrDiscover attempts to load configuration from a file.
// If the file doesn't exist or has no models defined, it falls back to
// auto-discovering models from the llama.cpp cache directory.
func LoadConfigOrDiscover(path string) (Config, error) {
	// Try to load the config file
	cfg, err := LoadConfig(path)

	// If file doesn't exist, try auto-discovery
	if os.IsNotExist(err) {
		log.Printf("Config file not found at %s, attempting auto-discovery from llama.cpp cache...", path)
		return AutoDiscoverConfig()
	}

	// If file loaded but has other errors, return the error
	if err != nil {
		return Config{}, err
	}

	// If config loaded successfully but has no models, try auto-discovery
	if len(cfg.Models) == 0 {
		log.Printf("Config file has no models defined, attempting auto-discovery from llama.cpp cache...")
		return AutoDiscoverConfig()
	}

	// Config loaded successfully with models
	return cfg, nil
}

// AutoDiscoverConfig discovers models from the llama.cpp cache and generates a config
func AutoDiscoverConfig() (Config, error) {
	// Discover models from cache
	log.Println("Scanning llama.cpp cache directory for GGUF files...")
	models, err := discovery.DiscoverModels()
	if err != nil {
		return Config{}, fmt.Errorf("failed to discover models: %w", err)
	}

	if len(models) == 0 {
		return Config{}, fmt.Errorf("no GGUF models found in llama.cpp cache directory")
	}

	log.Printf("Found %d GGUF file(s) in cache", len(models))

	// Deduplicate models (keep only first of each quantization variant)
	models = discovery.DeduplicateModels(models)
	if len(models) < len(models) {
		log.Printf("After deduplication: %d unique model(s)", len(models))
	}

	// Find llama-server binary
	log.Println("Searching for llama-server binary...")
	serverPath, err := discovery.FindLlamaServer()
	if err != nil {
		return Config{}, fmt.Errorf("failed to find llama-server: %w (set LLAMA_SERVER_PATH environment variable or ensure llama-server is in PATH)", err)
	}

	log.Printf("Found llama-server at: %s", serverPath)

	// Generate config from discovered models
	cfg, err := generateConfig(models, serverPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to generate config: %w", err)
	}

	// Process the generated config through normal validation
	// We need to marshal it to YAML and then unmarshal it to go through LoadConfigFromReader
	// But since we already have a valid Config struct, we can validate it directly

	log.Printf("Auto-discovered %d model(s):", len(cfg.Models))
	for modelID := range cfg.Models {
		log.Printf("  - %s", modelID)
	}

	// The generated config already has defaults set, but we need to process
	// it through the normal config loading pipeline to ensure all validation
	// and transformation steps are applied
	return *cfg, nil
}
