package runner

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
)

type codexProvider struct{}

func (codexProvider) buildCommand(ctx context.Context, projectDir, systemPrompt, userPrompt, model string, maxTurns int) *exec.Cmd {
	prompt := systemPrompt + "\n\n---\n\n" + userPrompt
	args := []string{"-a", "never", "exec", "--json", "--color", "never", "--sandbox", "workspace-write", "--skip-git-repo-check", "--ephemeral", "-C", projectDir}
	if model = strings.TrimSpace(model); model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, prompt)
	return exec.CommandContext(ctx, "codex", args...)
}

func (codexProvider) parseLine(raw, language string) ([]string, bool) {
	var ev codexEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		if t := strings.TrimSpace(raw); t != "" {
			return []string{t}, false
		}
		return nil, false
	}
	return codexEventToLines(ev), false
}

func (codexProvider) waitSucceeded(waitErr, ctxErr error, maxTurnsReached bool) bool {
	return waitErr == nil || errors.Is(ctxErr, context.Canceled)
}
