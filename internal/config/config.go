package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultLanguage is the interface and agent-response language used when no configuration exists.
	DefaultLanguage = "en"
	// DefaultModel is the default model for the Claude provider.
	DefaultModel = "sonnet"
	// DefaultProvider is the agent CLI used when no configuration exists.
	DefaultProvider   = "claude"
	defaultCodexModel = "gpt-5.4"
)

// Config contains the persistent user defaults for YVCDB.
type Config struct {
	Language string `json:"language"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// Default returns a configuration populated with safe defaults.
func Default() Config {
	return Config{Language: DefaultLanguage, Provider: DefaultProvider, Model: DefaultModel}
}

// ValidProvider reports whether provider identifies a supported agent CLI.
func ValidProvider(provider string) bool { return provider == "claude" || provider == "codex" }

// SuggestedModel returns the default model for provider.
func SuggestedModel(provider string) string {
	if provider == "codex" {
		return defaultCodexModel
	}
	return DefaultModel
}

// Path returns the persistent YVCDB configuration path.
func Path() (string, error) {
	if dir := os.Getenv("YVCDB_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "config.json"), nil
	}
	// Legacy override remains supported for automation using the old name.
	if dir := os.Getenv("TVCMM_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "config.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "yvcdb", "config.json"), nil
}

func legacyPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tvcmm", "config.json"), nil
}

// Load reads and normalizes the persistent configuration.
func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		legacy, legacyErr := legacyPath()
		if legacyErr != nil {
			return cfg, fmt.Errorf("resolve legacy configuration path: %w", legacyErr)
		}
		data, err = os.ReadFile(legacy)
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
	}
	if err != nil {
		return cfg, fmt.Errorf("read configuration: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("decode configuration: %w", err)
	}
	if cfg.Language != "en" && cfg.Language != "fr" {
		cfg.Language = DefaultLanguage
	}
	if !ValidProvider(cfg.Provider) {
		cfg.Provider = DefaultProvider
	}
	if cfg.Model == "" {
		cfg.Model = SuggestedModel(cfg.Provider)
	}
	return cfg, nil
}

// Save writes cfg to the persistent configuration file.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return fmt.Errorf("resolve configuration path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write configuration: %w", err)
	}
	return nil
}
