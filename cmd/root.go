package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

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
	flagMode     string
	version      = "dev"
)

var rootCmd = &cobra.Command{
	Use:     "yvcdb [project/path]",
	Short:   "Your Vibe Code Deserves Better — managed AFTER workflows powered by Claude Code or Codex",
	Version: version,
	Args:    cobra.MaximumNArgs(1),
	RunE:    run,
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
	rootCmd.Flags().StringVar(&flagPhase, "phase", "", "Resume at a phase in the selected workflow")
	rootCmd.Flags().StringVar(&flagModel, "model", "", "AI model for this run (overrides configuration)")
	rootCmd.Flags().StringVar(&flagProvider, "provider", "", "AI CLI provider: "+strings.Join(appconfig.SupportedProviders(), ", ")+" (overrides configuration)")
	rootCmd.Flags().StringVar(&flagLang, "lang", "", "Interface language: en or fr (overrides configuration)")
	rootCmd.Flags().StringVar(&flagMode, "mode", phases.ModeAuto, "Workflow mode: auto, refactor, greenfield, feature, or debug")
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
			return fmt.Errorf("unsupported provider %q: use %s", flagProvider, strings.Join(appconfig.SupportedProviders(), ", "))
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

	mode := flagMode
	if mode == phases.ModeAuto {
		mode, err = detectMode(absDir)
		if err != nil {
			return fmt.Errorf(l10n.Pick("detect workflow mode: %w", "détection du mode de travail : %w"), err)
		}
	}
	workflow, err := phases.ForMode(mode)
	if err != nil {
		return err
	}

	// Check the selected agent CLI.
	if err := checkProvider(cfg.Provider); err != nil {
		return err
	}

	// Determine start phase
	startPhase := 0
	if flagPhase != "" {
		idx := workflow.IndexOf(flagPhase)
		if idx < 0 {
			return fmt.Errorf(l10n.Pick("unknown phase %q for %s mode. Available phases: %s", "phase %q inconnue pour le mode %s. Phases disponibles : %s"), flagPhase, workflow.Mode, strings.Join(workflow.PhaseIDs(), ", "))
		}
		startPhase = idx
	}

	phaseExplicit := cmd != nil && cmd.Flags().Changed("phase")
	modeExplicit := cmd != nil && cmd.Flags().Changed("mode")
	resumeCandidate, _, err := resolveResumeCandidate(absDir, phaseExplicit, modeExplicit)
	if err != nil {
		return fmt.Errorf(l10n.Pick("resolve resume candidate: %w", "résolution de la reprise : %w"), err)
	}

	// Load embedded prompts
	prompts, err := loadPrompts(cfg.Language, workflow)
	if err != nil {
		return fmt.Errorf(l10n.Pick("load prompts: %w", "chargement des prompts : %w"), err)
	}

	// Launch TUI
	m := tui.NewModel(absDir, startPhase, flagNoGit, cfg.Provider, cfg.Model, flagMaxTurns, cfg.Language, prompts, resumeCandidate, workflow)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI: %w", err)
	}
	return nil
}

func resolveResumeCandidate(projectDir string, phaseExplicit, modeExplicit bool) (*runner.ResumeMarker, bool, error) {
	if phaseExplicit || modeExplicit {
		return nil, false, nil
	}
	markerPath := filepath.Join(projectDir, ".yvcdb_resume.json")
	marker, err := runner.ReadResumeMarker(markerPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		_ = os.Remove(markerPath)
		return nil, true, nil
	}
	if processAlive(marker.PID) {
		return nil, false, nil
	}
	return &marker, false, nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func detectMode(projectDir string) (string, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		switch {
		case entry.Name() == ".git", entry.Name() == ".gitkeep", entry.Name() == ".DS_Store", entry.Name() == "refactor-logs":
			continue
		case strings.HasPrefix(entry.Name(), ".yvcdb_"):
			continue
		default:
			return phases.ModeRefactor, nil
		}
	}
	return phases.ModeGreenfield, nil
}
