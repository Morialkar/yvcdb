package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	appconfig "github.com/Morialkar/yvcdb/internal/config"
	"github.com/Morialkar/yvcdb/internal/i18n"
	"github.com/Morialkar/yvcdb/internal/phases"
	"github.com/Morialkar/yvcdb/internal/runner"
	"github.com/Morialkar/yvcdb/internal/tui"
)

var (
	flagPhase    string
	flagModel    string
	flagMaxTurns int
	flagNoGit    bool
	flagLang     string
	flagProvider string
)

var rootCmd = &cobra.Command{
	Use:   "yvcdb [project/path]",
	Short: "Your Vibe Code Deserves Better — automated refactoring powered by Claude Code or Codex",
	Args:  cobra.MaximumNArgs(1),
	RunE:  run,
}

// Execute runs the YVCDB command-line application.
func Execute() {
	if filepath.Base(os.Args[0]) == "tvcmm" {
		rootCmd.Use = "tvcmm [project/path]"
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&flagPhase, "phase", "", "Resume at a specific phase (diagnostic, safety, security, structure, readability, devil)")
	rootCmd.Flags().StringVar(&flagModel, "model", "", "AI model for this run (overrides configuration)")
	rootCmd.Flags().StringVar(&flagProvider, "provider", "", "AI CLI provider: claude or codex (overrides configuration)")
	rootCmd.Flags().StringVar(&flagLang, "lang", "", "Interface language: en or fr (overrides configuration)")
	rootCmd.Flags().IntVar(&flagMaxTurns, "max-turns", runner.DefaultMaxTurns, "Maximum Claude turns (Claude provider only)")
	rootCmd.Flags().BoolVar(&flagNoGit, "no-git", false, "Disable automatic git management")
	rootCmd.AddCommand(configCmd)
}

func run(cmd *cobra.Command, args []string) error {
	cfg, err := appconfig.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if flagLang != "" {
		if flagLang != "en" && flagLang != "fr" {
			return fmt.Errorf("unsupported language %q: use en or fr", flagLang)
		}
		cfg.Language = flagLang
	}
	if flagModel != "" {
		cfg.Model = flagModel
	}
	if flagProvider != "" {
		if !appconfig.ValidProvider(flagProvider) {
			return fmt.Errorf("unsupported provider %q: use claude or codex", flagProvider)
		}
		cfg.Provider = flagProvider
		if flagModel == "" {
			cfg.Model = appconfig.SuggestedModel(cfg.Provider)
		}
	}
	l10n := i18n.New(cfg.Language)

	// Resolve project directory
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf(l10n.Pick("resolve project directory: %w", "résolution du répertoire du projet : %w"), err)
	}
	if _, err := os.Stat(absDir); err != nil {
		return fmt.Errorf(l10n.Pick("inspect project directory: %w", "inspection du répertoire du projet : %w"), err)
	}

	// Check the selected agent CLI.
	if err := checkProvider(cfg.Provider); err != nil {
		return err
	}

	// Determine start phase
	startPhase := 0
	if flagPhase != "" {
		idx := phases.IndexOf(flagPhase)
		if idx < 0 {
			return fmt.Errorf(l10n.Pick("unknown phase: %q. Available phases: diagnostic, safety, security, structure, readability, devil", "phase inconnue : %q. Phases disponibles : diagnostic, safety, security, structure, readability, devil"), flagPhase)
		}
		startPhase = idx
	}

	// Load embedded prompts
	prompts, err := loadPrompts(cfg.Language)
	if err != nil {
		return fmt.Errorf(l10n.Pick("load prompts: %w", "chargement des prompts : %w"), err)
	}

	// Launch TUI
	m := tui.NewModel(absDir, startPhase, flagNoGit, cfg.Provider, cfg.Model, flagMaxTurns, cfg.Language, prompts)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI: %w", err)
	}
	return nil
}
