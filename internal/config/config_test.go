package config

import (
	"path/filepath"
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
}

func TestLoadNormalizesInvalidValues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	if err := Save(Config{Language: "de"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg != Default() {
		t.Fatalf("got %+v, want defaults %+v", cfg, Default())
	}
}
