package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Morialkar/yvcdb/internal/phases"
)

func TestLoadPromptsForEachLanguage(t *testing.T) {
	for _, mode := range []string{phases.ModeRefactor, phases.ModeGreenfield, phases.ModeFeature} {
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
	dir := t.TempDir()
	path := filepath.Join(dir, "claude")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if err := checkProvider("claude"); err != nil {
		t.Fatal(err)
	}
	if err := checkProvider("codex"); err == nil {
		t.Fatal("expected missing Codex error")
	}
}

func TestLoadPromptsRejectsUnknownLanguage(t *testing.T) {
	if _, err := loadPrompts("de"); err == nil {
		t.Fatal("expected unknown language to fail")
	}
}
