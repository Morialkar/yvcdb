package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type claudeProvider struct{}

func (claudeProvider) buildCommand(ctx context.Context, projectDir, systemPrompt, userPrompt, model string, maxTurns int) *exec.Cmd {
	args := []string{
		"-p", userPrompt,
		"--append-system-prompt", systemPrompt,
		"--allowedTools", "Read,Write,Edit,Bash,Glob,Grep",
		"--output-format", "stream-json",
		"--verbose",
		"--max-turns", fmt.Sprint(maxTurns),
	}
	if model = strings.TrimSpace(model); model != "" {
		args = append(args, "--model", model)
	}
	return exec.CommandContext(ctx, "claude", args...)
}

func (claudeProvider) parseLine(raw, language string) ([]string, bool) {
	var ev streamEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		if t := strings.TrimSpace(raw); t != "" {
			return []string{t}, false
		}
		return nil, false
	}
	maxTurnsReached := ev.Type == "result" && ev.Subtype == "error_max_turns"
	return eventToLines(ev, language), maxTurnsReached
}

func (claudeProvider) waitSucceeded(waitErr, ctxErr error, maxTurnsReached bool) bool {
	if waitErr == nil {
		return true
	}
	return maxTurnsReached || errors.Is(ctxErr, context.Canceled)
}
