package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func resetFlags(t *testing.T) {
	t.Helper()
	origLang, origProvider, origPhase, origModel := flagLang, flagProvider, flagPhase, flagModel
	t.Cleanup(func() {
		flagLang, flagProvider, flagPhase, flagModel = origLang, origProvider, origPhase, origModel
	})
	flagLang, flagProvider, flagPhase, flagModel = "", "", "", ""
}

func writeFakeClaude(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "claude"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
}

func TestRunFailsWhenConfigUnreadable(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	if err := os.Mkdir(filepath.Join(dir, "config.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := run(nil, nil); err == nil {
		t.Fatal("expected configuration load error")
	}
}

func TestRunFailsWhenProviderMissing(t *testing.T) {
	resetFlags(t)
	t.Setenv("YVCDB_CONFIG_HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	if err := run(nil, []string{t.TempDir()}); err == nil {
		t.Fatal("expected missing provider CLI error")
	}
}

func TestRunAppliesFlagOverridesAndFailsWithoutTTY(t *testing.T) {
	resetFlags(t)
	t.Setenv("YVCDB_CONFIG_HOME", t.TempDir())
	writeFakeClaude(t)

	// valid overrides reach the TUI launch, which fails in tests (no TTY)
	flagLang = "fr"
	flagProvider = "claude"
	flagModel = "opus"
	flagPhase = "security"
	withStdin(t, "", func() {
		if err := run(nil, []string{t.TempDir()}); err == nil {
			t.Fatal("expected TUI startup failure without a TTY")
		}
	})
}

func TestRunProviderOverrideSuggestsModel(t *testing.T) {
	resetFlags(t)
	t.Setenv("YVCDB_CONFIG_HOME", t.TempDir())
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "codex"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	flagProvider = "codex"
	withStdin(t, "", func() {
		if err := run(nil, nil); err == nil {
			t.Fatal("expected TUI startup failure without a TTY")
		}
	})
}

func TestExecuteWithHelpFlag(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs; rootCmd.SetArgs(nil) })
	os.Args = []string{"tvcmm", "--help"}
	rootCmd.SetArgs([]string{"--help"})
	Execute()
	if rootCmd.Use != "tvcmm [project/path]" {
		t.Fatalf("expected tvcmm alias to rename Use, got %q", rootCmd.Use)
	}
}

func TestRunConfigReadErrors(t *testing.T) {
	t.Setenv("YVCDB_CONFIG_HOME", t.TempDir())
	// EOF at each successive prompt
	withStdin(t, "", func() {
		if err := runConfig(nil, nil); err == nil {
			t.Fatal("expected read language error")
		}
	})
	withStdin(t, "en\n", func() {
		if err := runConfig(nil, nil); err == nil {
			t.Fatal("expected read provider error")
		}
	})
	withStdin(t, "en\nclaude\n", func() {
		if err := runConfig(nil, nil); err == nil {
			t.Fatal("expected read model error")
		}
	})
}

func TestRunConfigFailsWhenConfigUnreadable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	if err := os.Mkdir(filepath.Join(dir, "config.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	withStdin(t, "en\nclaude\nopus\n", func() {
		if err := runConfig(nil, nil); err == nil {
			t.Fatal("expected load error")
		}
	})
}

func TestRunConfigFailsWhenSaveBlocked(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	// read-only config home: Load sees a missing file (fine), Save cannot write
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	withStdin(t, "en\nclaude\nopus\n", func() {
		if err := runConfig(nil, nil); err == nil {
			t.Fatal("expected save error")
		}
	})
}

func TestExecuteExitsOnError(t *testing.T) {
	if os.Getenv("YVCDB_TEST_EXECUTE_FAIL") == "1" {
		rootCmd.SetArgs([]string{"--not-a-flag"})
		rootCmd.SilenceErrors = true
		Execute()
		return // unreachable: Execute must os.Exit(1)
	}
	cmd := exec.Command(os.Args[0], "-test.run", "TestExecuteExitsOnError")
	cmd.Env = append(os.Environ(), "YVCDB_TEST_EXECUTE_FAIL=1")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got: %v", err)
	}
}
