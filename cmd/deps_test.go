package cmd

import (
	"strings"
	"testing"

	"github.com/Morialkar/yvcdb/internal/phases"
)

func TestLoadPromptsForEachLanguage(t *testing.T) {
	for _, language := range []string{"en", "fr"} {
		t.Run(language, func(t *testing.T) {
			prompts, err := loadPrompts(language)
			if err != nil {
				t.Fatal(err)
			}
			if len(prompts) != len(phases.All) {
				t.Fatalf("got %d prompts, want %d", len(prompts), len(phases.All))
			}
			for _, phase := range phases.All {
				if strings.TrimSpace(prompts[phase.ID]) == "" {
					t.Errorf("prompt %s is empty", phase.ID)
				}
			}
		})
	}
}
