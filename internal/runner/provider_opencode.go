package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type opencodeProvider struct{}

func (opencodeProvider) buildCommand(ctx context.Context, projectDir, systemPrompt, userPrompt, model, promptFilePath string, maxTurns int) *exec.Cmd {
	message := opencodeBootstrapPrompt(userPrompt)
	args := []string{"run", message, "--format", "json"}
	// REQUIRES_REVIEW: --auto is acceptable because YVCDB gates work at phase boundaries with git isolation and human approval.
	args = append(args, "--auto")
	if promptFilePath != "" {
		args = append(args, "-f", promptFilePath)
	}
	if model = strings.TrimSpace(model); model != "" {
		args = append(args, "--model", model)
	}
	return exec.CommandContext(ctx, "opencode", args...)
}

func (opencodeProvider) parseLine(raw, language string) ([]string, bool) {
	var ev opencodeEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		return nil, false
	}
	switch ev.Type {
	case "text":
		if ev.Part != nil {
			if text := strings.TrimSpace(ev.Part.Text); text != "" {
				return []string{text}, false
			}
		}
	case "tool_use":
		if ev.Part != nil && ev.Part.State != nil && ev.Part.State.Status == "completed" {
			if tool := strings.TrimSpace(ev.Part.Tool); tool != "" {
				return []string{fmt.Sprintf("  ⚙  %s", truncateRunes(tool, toolInputRunes))}, false
			}
		}
	}
	return nil, false
}

func (opencodeProvider) waitSucceeded(waitErr, ctxErr error, maxTurnsReached bool) bool {
	return waitErr == nil || ctxErr != nil && ctxErr == context.Canceled
}

func (opencodeProvider) needsPromptFile() bool { return true }

func (opencodeProvider) startupNotice(language string) string {
	if language == "fr" {
		return "OpenCode exécute la phase avec des permissions auto-approuvées."
	}
	return "OpenCode is running with auto-approved permissions."
}

func opencodeBootstrapPrompt(userPrompt string) string {
	bootstrap := "Read the attached prompt file in full before doing anything else, treat it as your instructions for this phase, then follow the project instructions below."
	if t := strings.TrimSpace(userPrompt); t != "" {
		return bootstrap + "\n\n" + t
	}
	return bootstrap
}

type opencodeEvent struct {
	Type string `json:"type"`
	Part *struct {
		Text  string `json:"text"`
		Tool  string `json:"tool"`
		State *struct {
			Status string `json:"status"`
		} `json:"state"`
	} `json:"part"`
}
