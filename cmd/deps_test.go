package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Morialkar/yvcdb/internal/phases"
)

func TestLoadPromptsForEachLanguage(t *testing.T) {
	for _, mode := range []string{phases.ModeRefactor, phases.ModeGreenfield, phases.ModeFeature, phases.ModeDebug} {
		workflow, err := phases.ForMode(mode)
		if err != nil {
			t.Fatal(err)
		}
		for _, language := range []string{"en", "fr"} {
			t.Run(mode+"/"+language, func(t *testing.T) {
				prompts, err := loadPrompts(language, workflow)
				if err != nil {
					t.Fatal(err)
				}
				if len(prompts) != len(workflow.Phases) {
					t.Fatalf("got %d prompts, want %d", len(prompts), len(workflow.Phases))
				}
				for _, phase := range workflow.Phases {
					if strings.TrimSpace(prompts[phase.ID]) == "" {
						t.Errorf("prompt %s is empty", phase.ID)
					}
				}
			})
		}
	}
}

func TestCheckProvider(t *testing.T) {
	t.Run("present executables pass", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"claude", "codex", "opencode"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		t.Setenv("PATH", dir)
		for _, provider := range []string{"claude", "codex", "opencode"} {
			if err := checkProvider(provider); err != nil {
				t.Fatalf("%s should pass: %v", provider, err)
			}
		}
	})

	t.Run("missing OpenCode is actionable", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "claude"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", dir)
		if err := checkProvider("opencode"); err == nil {
			t.Fatal("expected missing OpenCode error")
		} else if got := err.Error(); !strings.Contains(got, "OpenCode CLI not found") || !strings.Contains(got, "https://opencode.ai") || !strings.Contains(got, "opencode auth login") {
			t.Fatalf("unexpected OpenCode error: %v", err)
		}
	})
}

func TestLoadPromptsRejectsUnknownLanguage(t *testing.T) {
	if _, err := loadPrompts("de"); err == nil {
		t.Fatal("expected unknown language to fail")
	}
}
