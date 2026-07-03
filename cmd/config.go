package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	appconfig "github.com/morialkar/yvcdb/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure the language, AI CLI provider, and default model",
	Args:  cobra.NoArgs,
	RunE:  runConfig,
}

func runConfig(_ *cobra.Command, _ []string) error {
	cfg, err := appconfig.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Language [en/fr] (%s): ", cfg.Language)
	language, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	language = strings.ToLower(strings.TrimSpace(language))
	if language != "" {
		if language != "en" && language != "fr" {
			return fmt.Errorf("unsupported language %q: use en or fr", language)
		}
		cfg.Language = language
	}

	fmt.Printf("AI CLI provider [claude/codex] (%s): ", cfg.Provider)
	provider, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != "" {
		if !appconfig.ValidProvider(provider) {
			return fmt.Errorf("unsupported provider %q: use claude or codex", provider)
		}
		if provider != cfg.Provider {
			cfg.Provider = provider
			cfg.Model = appconfig.SuggestedModel(provider)
		}
	}

	fmt.Printf("Default %s model (%s): ", cfg.Provider, cfg.Model)
	model, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if model = strings.TrimSpace(model); model != "" {
		cfg.Model = model
	}

	if err := appconfig.Save(cfg); err != nil {
		return fmt.Errorf("save configuration: %w", err)
	}
	path, _ := appconfig.Path()
	fmt.Printf("Configuration saved to %s\n", path)
	fmt.Printf("Language: %s\nProvider: %s\nModel: %s\n", cfg.Language, cfg.Provider, cfg.Model)
	return nil
}
