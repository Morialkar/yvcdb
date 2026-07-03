package cmd

import (
	"embed"
	"fmt"
	"os/exec"

	"github.com/morialkar/yvcdb/internal/phases"
)

//go:embed prompts
var promptsFS embed.FS

func checkProvider(provider string) error {
	if _, err := exec.LookPath(provider); err != nil {
		if provider == "codex" {
			return fmt.Errorf("Codex CLI not found. Install it before using the codex provider")
		}
		return fmt.Errorf("Claude CLI not found. Install Claude Code: npm install -g @anthropic-ai/claude-code")
	}
	return nil
}

func loadPrompts() (map[string]string, error) {
	prompts := make(map[string]string, len(phases.All))
	for _, p := range phases.All {
		data, err := promptsFS.ReadFile("prompts/" + p.PromptFile)
		if err != nil {
			return nil, fmt.Errorf("prompt %s: %w", p.PromptFile, err)
		}
		prompts[p.ID] = string(data)
	}
	return prompts, nil
}
