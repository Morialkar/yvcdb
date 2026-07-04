package cmd

import (
	"embed"
	"fmt"
	"os/exec"

	"github.com/Morialkar/yvcdb/internal/phases"
)

//go:embed prompts/*/*/*.md
var promptsFS embed.FS

func checkProvider(provider string) error {
	if _, err := exec.LookPath(provider); err != nil {
		if provider == "codex" {
			return fmt.Errorf("Codex CLI not found: %w", err)
		}
		return fmt.Errorf("Claude CLI not found; install Claude Code with npm install -g @anthropic-ai/claude-code: %w", err)
	}
	return nil
}

func loadPrompts(language string, workflows ...phases.Workflow) (map[string]string, error) {
	workflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		return nil, err
	}
	if len(workflows) > 0 {
		workflow = workflows[0]
	}
	prompts := make(map[string]string, len(workflow.Phases))
	for _, p := range workflow.Phases {
		data, err := promptsFS.ReadFile("prompts/" + language + "/" + workflow.Mode + "/" + p.PromptFile)
		if err != nil {
			return nil, fmt.Errorf("prompt %s: %w", p.PromptFile, err)
		}
		prompts[p.ID] = string(data)
	}
	return prompts, nil
}
