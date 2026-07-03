package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	DefaultLanguage = "en"
	DefaultModel    = "sonnet"
	DefaultProvider = "claude"
)

type Config struct {
	Language string `json:"language"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func Default() Config {
	return Config{Language: DefaultLanguage, Provider: DefaultProvider, Model: DefaultModel}
}

func ValidProvider(provider string) bool { return provider == "claude" || provider == "codex" }

func SuggestedModel(provider string) string {
	if provider == "codex" {
		return "gpt-5.4"
	}
	return DefaultModel
}

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
			return cfg, nil
		}
		data, err = os.ReadFile(legacy)
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), err
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

func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}
