package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodexEventToLinesEmptyPayloads(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"agent message empty text", `{"type":"item.completed","item":{"type":"agent_message","text":"  "}}`},
		{"reasoning empty text", `{"type":"item.completed","item":{"type":"reasoning","text":""}}`},
		{"command completed empty output", `{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"  "}}`},
		{"agent message not completed", `{"type":"item.started","item":{"type":"agent_message","text":"early"}}`},
		{"file change not completed", `{"type":"item.started","item":{"type":"file_change"}}`},
		{"mcp not started", `{"type":"item.completed","item":{"type":"mcp_tool_call"}}`},
		{"nested error empty message", `{"type":"turn.failed","error":{"message":"  "}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event codexEvent
			if err := json.Unmarshal([]byte(tt.raw), &event); err != nil {
				t.Fatal(err)
			}
			if got := codexEventToLines(event); got != nil {
				t.Fatalf("expected no lines, got %#v", got)
			}
		})
	}
}

func TestProviderHelpersCoverEdgeCases(t *testing.T) {
	t.Run("claude startup notice", func(t *testing.T) {
		if got := (claudeProvider{}).startupNotice("en"); got != "" {
			t.Fatalf("expected empty notice, got %q", got)
		}
		if got := (claudeProvider{}).startupNotice("fr"); got != "" {
			t.Fatalf("expected empty notice, got %q", got)
		}
	})
	t.Run("codex startup notice", func(t *testing.T) {
		if got := (codexProvider{}).startupNotice("en"); got != "" {
			t.Fatalf("expected empty notice, got %q", got)
		}
		if got := (codexProvider{}).startupNotice("fr"); got != "" {
			t.Fatalf("expected empty notice, got %q", got)
		}
	})
	t.Run("opencode startup notice", func(t *testing.T) {
		if got := (opencodeProvider{}).startupNotice("en"); got != "OpenCode is running with auto-approved permissions." {
			t.Fatalf("unexpected en notice: %q", got)
		}
		if got := (opencodeProvider{}).startupNotice("fr"); got != "OpenCode exécute la phase avec des permissions auto-approuvées." {
			t.Fatalf("unexpected fr notice: %q", got)
		}
	})
	t.Run("opencode bootstrap prompt", func(t *testing.T) {
		if got := opencodeBootstrapPrompt("  "); got != "Read the attached prompt file in full before doing anything else, treat it as your instructions for this phase, then follow the project instructions below." {
			t.Fatalf("unexpected blank bootstrap: %q", got)
		}
		if got := opencodeBootstrapPrompt("  do the thing  "); !strings.Contains(got, "do the thing") || !strings.HasPrefix(got, "Read the attached prompt file in full") {
			t.Fatalf("unexpected bootstrap with prompt: %q", got)
		}
	})
	t.Run("provider waitSucceeded", func(t *testing.T) {
		if !(claudeProvider{}).waitSucceeded(nil, nil, false) {
			t.Fatal("claude should succeed on nil waitErr")
		}
		if !(claudeProvider{}).waitSucceeded(fmt.Errorf("boom"), context.Canceled, false) {
			t.Fatal("claude should succeed on canceled context")
		}
		if !(claudeProvider{}).waitSucceeded(fmt.Errorf("boom"), nil, true) {
			t.Fatal("claude should succeed on max turns")
		}
		if (claudeProvider{}).waitSucceeded(fmt.Errorf("boom"), context.DeadlineExceeded, false) {
			t.Fatal("claude should fail on non-canceled error")
		}
		if !(codexProvider{}).waitSucceeded(nil, nil, false) {
			t.Fatal("codex should succeed on nil waitErr")
		}
		if !(codexProvider{}).waitSucceeded(fmt.Errorf("boom"), context.Canceled, false) {
			t.Fatal("codex should succeed on canceled context")
		}
		if (codexProvider{}).waitSucceeded(fmt.Errorf("boom"), nil, false) {
			t.Fatal("codex should fail on non-canceled error")
		}
		if !(opencodeProvider{}).waitSucceeded(nil, nil, false) {
			t.Fatal("opencode should succeed on nil waitErr")
		}
		if !(opencodeProvider{}).waitSucceeded(fmt.Errorf("boom"), context.Canceled, false) {
			t.Fatal("opencode should succeed on canceled context")
		}
		if (opencodeProvider{}).waitSucceeded(fmt.Errorf("boom"), context.DeadlineExceeded, false) {
			t.Fatal("opencode should fail on non-canceled error")
		}
	})
}

func runPhaseWith(t *testing.T, projectDir, logDir, timestamp string, iteration int, opts Options) ([]string, error) {
	t.Helper()
	lineCh := make(chan string, 64)
	doneCh := make(chan error, 1)
	cancel := RunPhase(projectDir, logDir, timestamp, "phase", iteration, "system prompt", opts, lineCh, doneCh)
	defer cancel()
	var lines []string
	for line := range lineCh {
		lines = append(lines, line)
	}
	return lines, <-doneCh
}

func TestRunPhaseFailsWhenLogDirIsAFile(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocker, []byte("file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runPhaseWith(t, t.TempDir(), filepath.Join(blocker, "logs"), "ts", 1, Options{})
	if err == nil || !strings.Contains(err.Error(), "mkdir logs") {
		t.Fatalf("expected mkdir failure, got: %v", err)
	}
}

func TestRunPhaseFailsWhenLogFileIsADirectory(t *testing.T) {
	logDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(logDir, "ts_phase_iter1.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := runPhaseWith(t, t.TempDir(), logDir, "ts", 1, Options{})
	if err == nil || !strings.Contains(err.Error(), "create log") {
		t.Fatalf("expected create log failure, got: %v", err)
	}
}

func TestRunPhaseOpenCodeParserEdgeCases(t *testing.T) {
	provider := opencodeProvider{}
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty text", raw: `{"type":"text","part":{"text":"   "}}`},
		{name: "tool not completed", raw: `{"type":"tool_use","part":{"tool":"read","state":{"status":"running"}}}`},
		{name: "tool empty name", raw: `{"type":"tool_use","part":{"tool":"   ","state":{"status":"completed"}}}`},
		{name: "unknown type", raw: `{"type":"step_start"}`},
		{name: "malformed", raw: `not json`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, done := provider.parseLine(tt.raw, "en")
			if done {
				t.Fatalf("unexpected completion signal for %s", tt.name)
			}
			if len(got) != 0 {
				t.Fatalf("expected no lines, got %#v", got)
			}
		})
	}
}

func TestRunPhaseFrenchPromptsAndDefaults(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, binDir, "claude", `#!/bin/sh
printf '%s\n' '{"type":"result","result":"Fini"}'
`)
	t.Setenv("PATH", binDir)

	logDir := t.TempDir()
	// no provider, no model, no max-turns: exercises every default branch
	lines, err := runPhaseWith(t, t.TempDir(), logDir, "fr-run", 1, Options{Language: "fr", Feedback: "sois précis"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Résultat final") {
		t.Fatalf("expected French final marker, got: %q", joined)
	}
	data, err := os.ReadFile(filepath.Join(logDir, "fr-run_phase_iter1.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Projet:") {
		t.Fatalf("expected French log header, got: %q", data)
	}
}

func TestRunPhaseFrenchIterationHeading(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, binDir, "claude", `#!/bin/sh
printf '%s\n' 'ok'
`)
	t.Setenv("PATH", binDir)

	logDir := t.TempDir()
	if _, err := runPhaseWith(t, t.TempDir(), logDir, "fr-iter", 1, Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runPhaseWith(t, t.TempDir(), logDir, "fr-iter", 2, Options{Language: "fr", Feedback: "plus de détails"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunPhaseCodexRawLineAndModelFlag(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, binDir, "codex", `#!/bin/sh
printf '%s\n' 'not json output'
`)
	t.Setenv("PATH", binDir)

	lines, err := runPhaseWith(t, t.TempDir(), t.TempDir(), "codex-raw", 1, Options{Provider: "codex", Model: "gpt-5.4"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0] != "not json output" {
		t.Fatalf("expected raw forwarding, got %#v", lines)
	}
}

func TestRunPhaseCancelSuppressesExitError(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, binDir, "claude", `#!/bin/sh
/bin/sleep 5
`)
	t.Setenv("PATH", binDir)

	lineCh := make(chan string, 64)
	doneCh := make(chan error, 1)
	cancel := RunPhase(t.TempDir(), t.TempDir(), "cancelled", "phase", 1, "system prompt", Options{}, lineCh, doneCh)
	time.Sleep(200 * time.Millisecond)
	cancel()
	for range lineCh {
	}
	if err := <-doneCh; err != nil {
		t.Fatalf("cancellation should not surface an error, got: %v", err)
	}
}

func TestReadFirstNLinesMissingFile(t *testing.T) {
	if _, err := readFirstNLines(filepath.Join(t.TempDir(), "missing.md"), 10); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRunPhaseReportsScannerOverflow(t *testing.T) {
	binDir := t.TempDir()
	// stdout line beyond maxJSONEventBytes and stderr line beyond the default
	// scanner buffer both trigger bufio.ErrTooLong
	writeExecutable(t, binDir, "claude", `#!/bin/sh
PATH=/usr/bin:/bin
head -c 2097152 /dev/zero | tr '\0' 'a'
printf '\n'
head -c 131072 /dev/zero | tr '\0' 'b' >&2
printf '\n' >&2
`)
	t.Setenv("PATH", binDir)

	_, err := runPhaseWith(t, t.TempDir(), t.TempDir(), "overflow", 1, Options{})
	if err == nil || !strings.Contains(err.Error(), "stdout") {
		t.Fatalf("expected stdout scan error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "stderr") {
		t.Fatalf("expected stderr scan error, got: %v", err)
	}
}

func TestRunPhaseOpenCodeNonZeroExitSurfacesStderr(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, binDir, "opencode", `#!/bin/sh
set -eu
while [ $# -gt 0 ]; do
	case "$1" in
		run|--auto)
			shift
			;;
		--format)
			shift 2
			;;
		-f|--file|--model)
			shift 2
			;;
		*)
			shift
			;;
	esac
done
printf '%s\n' 'opencode failure' >&2
exit 7
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	lines, err := runPhaseWith(t, t.TempDir(), t.TempDir(), "opencode-error", 1, Options{Provider: "opencode", Language: "en"})
	if err == nil {
		t.Fatal("expected non-zero OpenCode exit to fail")
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "OpenCode is running with auto-approved permissions.") || !strings.Contains(joined, "[stderr] opencode failure") {
		t.Fatalf("unexpected lines: %q", joined)
	}
}

func TestRunPhaseWatchdogKillsInactiveRun(t *testing.T) {
	binDir := t.TempDir()
	projectDir := t.TempDir()
	logDir := t.TempDir()
	writeExecutable(t, binDir, "opencode", `#!/bin/sh
set -eu
prompt_file=""
while [ $# -gt 0 ]; do
	case "$1" in
		run)
			shift
			;;
		--format)
			shift 2
			;;
		--auto)
			shift
			;;
		-f|--file)
			prompt_file=$2
			shift 2
			;;
		--model)
			shift 2
			;;
		*)
			shift
			;;
	esac
done
printf '%s\n' "$prompt_file" > prompt-path.txt
sleep 2
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	lines, err := runPhaseWithOptions(t, projectDir, logDir, "watchdog", 1, Options{Provider: "opencode", Language: "en", InactivityTimeout: 50 * time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "inactivity timeout after") {
		t.Fatalf("expected watchdog timeout, got err=%v lines=%#v", err, lines)
	}
	promptPathBytes, readErr := os.ReadFile(filepath.Join(projectDir, "prompt-path.txt"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	promptPath := strings.TrimSpace(string(promptPathBytes))
	if _, statErr := os.Stat(promptPath); !os.IsNotExist(statErr) {
		t.Fatalf("prompt file should be deleted after watchdog kill, stat err=%v", statErr)
	}
}

func TestRunPhaseCancelDeletesPromptFile(t *testing.T) {
	binDir := t.TempDir()
	projectDir := t.TempDir()
	logDir := t.TempDir()
	writeExecutable(t, binDir, "opencode", `#!/bin/sh
set -eu
prompt_file=""
while [ $# -gt 0 ]; do
	case "$1" in
		run)
			shift
			;;
		--format)
			shift 2
			;;
		--auto)
			shift
			;;
		-f|--file)
			prompt_file=$2
			shift 2
			;;
		--model)
			shift 2
			;;
		*)
			shift
			;;
	esac
done
printf '%s\n' "$prompt_file" > prompt-path.txt
sleep 2
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	lineCh := make(chan string, 64)
	doneCh := make(chan error, 1)
	cancel := RunPhase(projectDir, logDir, "cancel", "phase", 1, "system prompt", Options{
		Provider:          "opencode",
		Language:          "en",
		InactivityTimeout: 5 * time.Second,
	}, lineCh, doneCh)
	waitForFile(t, filepath.Join(projectDir, "prompt-path.txt"))
	cancel()
	for range lineCh {
	}
	if err := <-doneCh; err != nil {
		t.Fatalf("cancellation should succeed, got: %v", err)
	}
	promptPathBytes, err := os.ReadFile(filepath.Join(projectDir, "prompt-path.txt"))
	if err != nil {
		t.Fatal(err)
	}
	promptPath := strings.TrimSpace(string(promptPathBytes))
	if _, statErr := os.Stat(promptPath); !os.IsNotExist(statErr) {
		t.Fatalf("prompt file should be deleted after cancellation, stat err=%v", statErr)
	}
}

func TestRunPhaseWatchdogResetsOnActivity(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, binDir, "opencode", `#!/bin/sh
set -eu
prompt_file=""
while [ $# -gt 0 ]; do
	case "$1" in
		run)
			shift
			;;
		--format)
			shift 2
			;;
		--auto)
			shift
			;;
		-f|--file)
			prompt_file=$2
			shift 2
			;;
		--model)
			shift 2
			;;
		*)
			shift
			;;
	esac
done
printf '%s\n' "$prompt_file" > prompt-path.txt
printf '%s\n' '{"type":"text","part":{"text":"tick 1"}}'
sleep 0.05
printf '%s\n' '{"type":"text","part":{"text":"tick 2"}}'
sleep 0.05
printf '%s\n' '{"type":"text","part":{"text":"tick 3"}}'
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	lines, err := runPhaseWithOptions(t, t.TempDir(), t.TempDir(), "watchdog-activity", 1, Options{Provider: "opencode", Language: "en", InactivityTimeout: 200 * time.Millisecond})
	if err != nil {
		t.Fatalf("expected active run to succeed, got: %v", err)
	}
	if joined := strings.Join(lines, "\n"); !strings.Contains(joined, "tick 1") || !strings.Contains(joined, "tick 3") {
		t.Fatalf("unexpected lines: %q", joined)
	}
}

func runPhaseWithOptions(t *testing.T, projectDir, logDir, timestamp string, iteration int, opts Options) ([]string, error) {
	t.Helper()
	lineCh := make(chan string, 64)
	doneCh := make(chan error, 1)
	cancel := RunPhase(projectDir, logDir, timestamp, "phase", iteration, "system prompt", opts, lineCh, doneCh)
	defer cancel()
	var lines []string
	for line := range lineCh {
		lines = append(lines, line)
	}
	return lines, <-doneCh
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("file did not appear: %s", path)
}
