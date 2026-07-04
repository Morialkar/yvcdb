package runner

import (
	"context"
	"os/exec"
)

type providerBackend interface {
	buildCommand(ctx context.Context, projectDir, systemPrompt, userPrompt, model string, maxTurns int) *exec.Cmd
	parseLine(raw, language string) ([]string, bool)
	waitSucceeded(waitErr, ctxErr error, maxTurnsReached bool) bool
}

type providerRuntime struct {
	name    string
	backend providerBackend
}

func selectProvider(name string) providerRuntime {
	selected := providerRuntime{name: name, backend: claudeProvider{}}
	if selected.name == "" {
		selected.name = "claude"
	}
	if name == "codex" {
		selected.backend = codexProvider{}
	}
	return selected
}

func (p providerRuntime) buildCommand(ctx context.Context, projectDir, systemPrompt, userPrompt, model string, maxTurns int) *exec.Cmd {
	return p.backend.buildCommand(ctx, projectDir, systemPrompt, userPrompt, model, maxTurns)
}

func (p providerRuntime) parseLine(raw, language string) ([]string, bool) {
	return p.backend.parseLine(raw, language)
}

func (p providerRuntime) waitSucceeded(waitErr, ctxErr error, maxTurnsReached bool) bool {
	return p.backend.waitSucceeded(waitErr, ctxErr, maxTurnsReached)
}
