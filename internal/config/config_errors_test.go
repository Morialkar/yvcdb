package config

import (
	"os"
	"path/filepath"
	"testing"
)

// clearConfigEnv blanks every variable that Path/legacyPath consult so that
// os.UserConfigDir fails on both darwin (HOME) and linux (XDG_CONFIG_HOME, HOME).
func clearConfigEnv(t *testing.T) {
	t.Helper()
	t.Setenv("YVCDB_CONFIG_HOME", "")
	t.Setenv("TVCMM_CONFIG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
}

func TestPathFailsWithoutUserConfigDir(t *testing.T) {
	clearConfigEnv(t)
	if _, err := Path(); err == nil {
		t.Fatal("expected error when the user config dir cannot be resolved")
	}
}

func TestLoadFailsWithoutUserConfigDir(t *testing.T) {
	clearConfigEnv(t)
	if _, err := Load(); err == nil {
		t.Fatal("expected error when the config path cannot be resolved")
	}
}

func TestLoadFailsWhenLegacyPathUnresolvable(t *testing.T) {
	// Path succeeds via the env override, but the file is absent so Load falls
	// back to legacyPath, which needs the user config dir and fails.
	clearConfigEnv(t)
	t.Setenv("YVCDB_CONFIG_HOME", t.TempDir())
	if _, err := Load(); err == nil {
		t.Fatal("expected legacy path resolution error")
	}
}

func TestLoadFailsWhenConfigIsUnreadable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	// config.json as a directory makes os.ReadFile fail with a non-NotExist error
	if err := os.Mkdir(filepath.Join(dir, "config.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected read error")
	}
}

func TestSaveFailsWithoutUserConfigDir(t *testing.T) {
	clearConfigEnv(t)
	if err := Save(Default()); err == nil {
		t.Fatal("expected error when the config path cannot be resolved")
	}
}

func TestSaveFailsWhenConfigDirIsAFile(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocker, []byte("file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// config home nested under a regular file → MkdirAll fails
	t.Setenv("YVCDB_CONFIG_HOME", filepath.Join(blocker, "nested"))
	if err := Save(Default()); err == nil {
		t.Fatal("expected mkdir failure")
	}
}

func TestSaveFailsWhenConfigFileIsADirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	if err := os.Mkdir(filepath.Join(dir, "config.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Save(Default()); err == nil {
		t.Fatal("expected write failure when the target is a directory")
	}
}

func TestLoadFallsBackToLegacyFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", "")
	t.Setenv("TVCMM_CONFIG_HOME", "")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	legacyDir, err := legacyPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(legacyDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyDir, []byte(`{"language":"fr","provider":"claude","model":"opus"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Language != "fr" || cfg.Model != "opus" {
		t.Fatalf("legacy config not loaded: %+v", cfg)
	}
}
