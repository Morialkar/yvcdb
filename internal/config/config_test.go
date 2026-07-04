package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultsWhenConfigIsMissing(t *testing.T) {
	t.Setenv("YVCDB_CONFIG_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Language != "en" || cfg.Provider != "claude" || cfg.Model != "sonnet" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLegacyEnvironmentOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", "")
	t.Setenv("TVCMM_CONFIG_HOME", dir)
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, "config.json") {
		t.Fatalf("unexpected legacy path: %s", path)
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected malformed configuration error")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	want := Config{Language: "fr", Provider: "codex", Model: "gpt-5.4"}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, "config.json") {
		t.Fatalf("unexpected path: %s", path)
	}
}

func TestSuggestedModel(t *testing.T) {
	if got := SuggestedModel("claude"); got != "sonnet" {
		t.Fatalf("claude model: %q", got)
	}
	if got := SuggestedModel("codex"); got != "gpt-5.4" {
		t.Fatalf("codex model: %q", got)
	}
	if got := SuggestedModel("opencode"); got != "" {
		t.Fatalf("opencode model: %q", got)
	}
}

func TestLoadNormalizesInvalidLanguage(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	if err := Save(Config{Language: "de", Provider: "codex"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Language != DefaultLanguage || cfg.Provider != "codex" || cfg.Model != "gpt-5.4" {
		t.Fatalf("language not normalized correctly: %+v", cfg)
	}
}

func TestLoadRejectsInvalidProvider(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	if err := Save(Config{Language: "en", Provider: "other"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid provider error")
	} else if got := err.Error(); got == "" || !containsAll(got, []string{"claude", "codex", "opencode"}) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveAndLoadOpenCode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	want := Config{Language: "en", Provider: "opencode", Model: ""}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func containsAll(s string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
