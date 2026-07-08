package cmd

import (
	"os"
	"path/filepath"
	"testing"

	appconfig "github.com/Morialkar/yvcdb/internal/config"
)

func TestRunConfig(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", configDir)
	withStdin(t, "fr\ncodex\ngpt-test\n", func() {
		if err := runConfig(nil, nil); err != nil {
			t.Fatal(err)
		}
	})
	cfg, err := appconfig.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Language != "fr" || cfg.Provider != "codex" || cfg.Model != "gpt-test" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestRunConfigAcceptsOpenCode(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", configDir)
	withStdin(t, "en\nopencode\n\n", func() {
		if err := runConfig(nil, nil); err != nil {
			t.Fatal(err)
		}
	})
	cfg, err := appconfig.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Language != "en" || cfg.Provider != "opencode" || cfg.Model != "" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestRunConfigKeepsExistingDefaultsOnBlankInput(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", configDir)
	want := appconfig.Config{Language: "fr", Provider: "claude", Model: "opus"}
	if err := appconfig.Save(want); err != nil {
		t.Fatal(err)
	}
	withStdin(t, "\n\n\n", func() {
		if err := runConfig(nil, nil); err != nil {
			t.Fatal(err)
		}
	})
	got, err := appconfig.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestRunConfigValidation(t *testing.T) {
	t.Setenv("YVCDB_CONFIG_HOME", t.TempDir())
	withStdin(t, "de\n", func() {
		if err := runConfig(nil, nil); err == nil {
			t.Fatal("expected invalid language error")
		}
	})
	withStdin(t, "en\nother\n", func() {
		if err := runConfig(nil, nil); err == nil {
			t.Fatal("expected invalid provider error")
		}
	})
}

func TestRunRejectsInvalidInputs(t *testing.T) {
	originalLang, originalProvider, originalPhase := flagLang, flagProvider, flagPhase
	originalModel := flagModel
	t.Cleanup(func() {
		flagLang, flagProvider, flagPhase, flagModel = originalLang, originalProvider, originalPhase, originalModel
	})
	t.Setenv("YVCDB_CONFIG_HOME", t.TempDir())

	flagLang = "de"
	if err := run(nil, nil); err == nil {
		t.Fatal("expected invalid language error")
	}
	flagLang = ""
	flagProvider = "other"
	if err := run(nil, nil); err == nil {
		t.Fatal("expected invalid provider error")
	}
	flagProvider = ""
	if err := run(nil, []string{filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("expected missing directory error")
	}

	dir := t.TempDir()
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "claude"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	flagPhase = "unknown"
	if err := run(nil, []string{dir}); err == nil {
		t.Fatal("expected unknown phase error")
	}
}

func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stdin")
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	old := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = old }()
	fn()
}
