package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Morialkar/yvcdb/internal/runner"
)

func resetFlags(t *testing.T) {
	t.Helper()
	origLang, origProvider, origPhase, origModel, origMode := flagLang, flagProvider, flagPhase, flagModel, flagMode
	t.Cleanup(func() {
		flagLang, flagProvider, flagPhase, flagModel, flagMode = origLang, origProvider, origPhase, origModel, origMode
	})
	flagLang, flagProvider, flagPhase, flagModel, flagMode = "", "", "", "", "auto"
}

func TestDetectMode(t *testing.T) {
	empty := t.TempDir()
	if got, err := detectMode(empty); err != nil || got != "greenfield" {
		t.Fatalf("empty directory: mode=%q err=%v", got, err)
	}
	if err := os.Mkdir(filepath.Join(empty, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := detectMode(empty); err != nil || got != "greenfield" {
		t.Fatalf("git-only directory: mode=%q err=%v", got, err)
	}
	if err := os.WriteFile(filepath.Join(empty, "README.md"), []byte("project"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := detectMode(empty); err != nil || got != "refactor" {
		t.Fatalf("non-empty directory: mode=%q err=%v", got, err)
	}
	artifacts := t.TempDir()
	if err := os.Mkdir(filepath.Join(artifacts, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifacts, ".yvcdb_resume.json"), []byte(`{"schemaVersion":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifacts, ".yvcdb_security_iter1_abcd.md"), []byte("prompt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := detectMode(artifacts); err != nil || got != "greenfield" {
		t.Fatalf("yvcdb artifacts should not affect mode: mode=%q err=%v", got, err)
	}
	if _, err := detectMode(filepath.Join(empty, "missing")); err == nil {
		t.Fatal("missing directory should fail")
	}
}

func TestResolveResumeCandidate(t *testing.T) {
	t.Run("valid marker with dead pid", func(t *testing.T) {
		dir := t.TempDir()
		child := exec.Command("sh", "-c", "exit 0")
		if err := child.Start(); err != nil {
			t.Fatal(err)
		}
		pid := child.Process.Pid
		if err := child.Wait(); err != nil {
			t.Fatal(err)
		}
		marker := runner.ResumeMarker{SchemaVersion: 1, PID: pid, PhaseID: "security"}
		if err := runner.WriteResumeMarker(filepath.Join(dir, ".yvcdb_resume.json"), marker); err != nil {
			t.Fatal(err)
		}
		got, cleanup, err := resolveResumeCandidate(dir, false, false)
		if err != nil {
			t.Fatal(err)
		}
		if cleanup {
			t.Fatal("valid marker should not be cleaned up")
		}
		if got == nil || got.PID != pid {
			t.Fatalf("expected resume candidate, got %#v", got)
		}
	})

	t.Run("malformed marker is removed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".yvcdb_resume.json")
		if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		got, cleanup, err := resolveResumeCandidate(dir, false, false)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil || !cleanup {
			t.Fatalf("expected stale marker cleanup, got=%#v cleanup=%v", got, cleanup)
		}
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("stale marker should be removed, stat err=%v", statErr)
		}
	})

	t.Run("wrong schema marker is removed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".yvcdb_resume.json")
		if err := os.WriteFile(path, []byte(`{"schemaVersion":2,"PID":1}`), 0o600); err != nil {
			t.Fatal(err)
		}
		got, cleanup, err := resolveResumeCandidate(dir, false, false)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil || !cleanup {
			t.Fatalf("expected stale marker cleanup, got=%#v cleanup=%v", got, cleanup)
		}
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("stale marker should be removed, stat err=%v", statErr)
		}
	})

	t.Run("live pid leaves marker alone", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".yvcdb_resume.json")
		if err := runner.WriteResumeMarker(path, runner.ResumeMarker{SchemaVersion: 1, PID: os.Getpid(), PhaseID: "security"}); err != nil {
			t.Fatal(err)
		}
		got, cleanup, err := resolveResumeCandidate(dir, false, false)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil || cleanup {
			t.Fatalf("expected live session suppression, got=%#v cleanup=%v", got, cleanup)
		}
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("live marker should remain in place, stat err=%v", statErr)
		}
	})

	t.Run("explicit phase suppresses resume", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".yvcdb_resume.json")
		if err := runner.WriteResumeMarker(path, runner.ResumeMarker{SchemaVersion: 1, PID: os.Getpid() + 100000, PhaseID: "security"}); err != nil {
			t.Fatal(err)
		}
		got, cleanup, err := resolveResumeCandidate(dir, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil || cleanup {
			t.Fatalf("expected explicit phase to suppress resume, got=%#v cleanup=%v", got, cleanup)
		}
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("marker should remain when explicit flags suppress resume, stat err=%v", statErr)
		}
	})

	t.Run("explicit mode suppresses resume", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".yvcdb_resume.json")
		if err := runner.WriteResumeMarker(path, runner.ResumeMarker{SchemaVersion: 1, PID: os.Getpid() + 100000, PhaseID: "security"}); err != nil {
			t.Fatal(err)
		}
		got, cleanup, err := resolveResumeCandidate(dir, false, true)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil || cleanup {
			t.Fatalf("expected explicit mode to suppress resume, got=%#v cleanup=%v", got, cleanup)
		}
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("marker should remain when explicit flags suppress resume, stat err=%v", statErr)
		}
	})

	t.Run("no marker", func(t *testing.T) {
		got, cleanup, err := resolveResumeCandidate(t.TempDir(), false, false)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil || cleanup {
			t.Fatalf("expected no resume candidate, got=%#v cleanup=%v", got, cleanup)
		}
	})
}

func writeFakeClaude(t *testing.T) {
	t.Helper()
	writeFakeExecutable(t, "claude")
}

func writeFakeOpenCode(t *testing.T) {
	t.Helper()
	writeFakeExecutable(t, "opencode")
}

func writeFakeExecutable(t *testing.T, name string) {
	t.Helper()
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
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

func TestRunFailsWhenConfigProviderInvalid(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	t.Setenv("YVCDB_CONFIG_HOME", dir)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"language":"en","provider":"other","model":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(nil, nil); err == nil {
		t.Fatal("expected invalid provider error")
	} else if got := err.Error(); !strings.Contains(got, "claude") || !strings.Contains(got, "codex") || !strings.Contains(got, "opencode") {
		t.Fatalf("unexpected error: %v", err)
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

func TestRunAcceptsOpenCodeProviderFlag(t *testing.T) {
	resetFlags(t)
	t.Setenv("YVCDB_CONFIG_HOME", t.TempDir())
	writeFakeOpenCode(t)

	flagProvider = "opencode"
	withStdin(t, "", func() {
		if err := run(nil, []string{t.TempDir()}); err == nil {
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
