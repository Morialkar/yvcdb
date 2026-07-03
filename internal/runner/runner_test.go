package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestClaudeEventToLines(t *testing.T) {
	event := streamEvent{Type: "assistant"}
	event.Message = &struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}{}
	event.Message.Content = append(event.Message.Content,
		struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}{Type: "text", Text: "Analysis"},
		struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}{Type: "tool_use", Name: "Read", Input: json.RawMessage(`{"path":"file.go"}`)},
	)
	if got := eventToLines(event, "en"); len(got) != 2 || got[0] != "Analysis" || !strings.Contains(got[1], "Read") {
		t.Fatalf("unexpected assistant lines: %#v", got)
	}

	toolResult := eventToLines(streamEvent{Type: "tool_result", Content: strings.Repeat("x", 220)}, "en")
	if len(toolResult) != 1 || !strings.HasSuffix(toolResult[0], "…") {
		t.Fatalf("unexpected tool result: %#v", toolResult)
	}
	final := eventToLines(streamEvent{Type: "result", Result: "Terminé"}, "fr")
	if len(final) != 3 || final[1] != "─── Résultat final ───" {
		t.Fatalf("unexpected final result: %#v", final)
	}
	if got := eventToLines(streamEvent{Type: "assistant"}, "en"); got != nil {
		t.Fatalf("nil message should produce no lines: %#v", got)
	}
}

func TestCodexEventToLines(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"agent message", `{"type":"item.completed","item":{"type":"agent_message","text":"Done."}}`, []string{"Done."}},
		{"command start", `{"type":"item.started","item":{"type":"command_execution","command":"go test ./..."}}`, []string{"  ⚙  go test ./..."}},
		{"command output", `{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"PASS"}}`, []string{"  ↳  PASS"}},
		{"reasoning", `{"type":"item.completed","item":{"type":"reasoning","text":"Thinking"}}`, []string{"  ◇  Thinking"}},
		{"file change", `{"type":"item.completed","item":{"type":"file_change"}}`, []string{"  ✎  files changed"}},
		{"mcp", `{"type":"item.started","item":{"type":"mcp_tool_call"}}`, []string{"  ⚙  MCP tool call"}},
		{"error", `{"type":"error","message":"authentication failed"}`, []string{"  [error] authentication failed"}},
		{"nested error", `{"type":"turn.failed","error":{"message":"failed"}}`, []string{"  [error] failed"}},
		{"unknown", `{"type":"turn.started"}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event codexEvent
			if err := json.Unmarshal([]byte(tt.raw), &event); err != nil {
				t.Fatal(err)
			}
			if got := codexEventToLines(event); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestRunPhaseWithClaudeAndCodex(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, binDir, "claude", `#!/bin/sh
printf '%s\n' '{"type":"assistant","message":{"content":[{"type":"text","text":"Claude analysis"}]}}'
printf '%s\n' '{"type":"result","result":"Claude done"}'
`)
	writeExecutable(t, binDir, "codex", `#!/bin/sh
printf '%s\n' '{"type":"item.started","item":{"type":"command_execution","command":"go test ./..."}}'
printf '%s\n' '{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"PASS"}}'
printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"Codex done"}}'
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := t.TempDir()
	logDir := filepath.Join(t.TempDir(), "logs")
	for _, provider := range []string{"claude", "codex"} {
		t.Run(provider, func(t *testing.T) {
			lines, err := runTestPhase(projectDir, logDir, "session", provider, 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(lines) == 0 {
				t.Fatal("expected streamed lines")
			}
			logPath := filepath.Join(logDir, "session_"+provider+"_iter1.md")
			if _, err := os.Stat(logPath); err != nil {
				t.Fatalf("missing log: %v", err)
			}
		})
	}

	// A retry reads the previous iteration log and injects feedback.
	if _, err := runTestPhase(projectDir, logDir, "retry", "claude", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := runTestPhase(projectDir, logDir, "retry", "claude", 2); err != nil {
		t.Fatal(err)
	}
}

func TestRunPhaseReportsFailuresAndAllowsMaxTurns(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	projectDir := t.TempDir()
	logDir := t.TempDir()

	writeExecutable(t, binDir, "codex", "#!/bin/sh\nexit 7\n")
	if _, err := runTestPhase(projectDir, logDir, "failure", "codex", 1); err == nil {
		t.Fatal("expected non-zero Codex exit to fail")
	}

	writeExecutable(t, binDir, "claude", `#!/bin/sh
printf '%s\n' '{"type":"result","subtype":"error_max_turns","result":"limit reached"}'
exit 1
`)
	if _, err := runTestPhase(projectDir, logDir, "maxturns", "claude", 1); err != nil {
		t.Fatalf("max-turn exit should be accepted: %v", err)
	}

	writeExecutable(t, binDir, "claude", "#!/bin/sh\nexit 9\n")
	if _, err := runTestPhase(projectDir, logDir, "claude-failure", "claude", 1); err == nil {
		t.Fatal("expected non-max-turn Claude exit to fail")
	}
}

func TestRunPhaseForwardsRawOutputAndStderr(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, binDir, "claude", `#!/bin/sh
printf '%s\n' 'plain output'
printf '%s\n' 'warning' >&2
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	lines, err := runTestPhase(t.TempDir(), t.TempDir(), "raw", "claude", 1)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "plain output") || !strings.Contains(joined, "[stderr] warning") {
		t.Fatalf("unexpected lines: %q", joined)
	}
}

func TestRunPhaseReportsStartupAndPreviousLogErrors(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := runTestPhase(t.TempDir(), t.TempDir(), "missing", "claude", 1); err == nil {
		t.Fatal("expected missing executable error")
	}

	binDir := t.TempDir()
	writeExecutable(t, binDir, "claude", "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir)
	if _, err := runTestPhase(t.TempDir(), t.TempDir(), "no-previous-log", "claude", 2); err == nil {
		t.Fatal("expected missing previous log error")
	}
}

func TestReadFirstNLinesAndTruncate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lines.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readFirstNLines(path, 2)
	if err != nil || got != "one\ntwo" {
		t.Fatalf("got %q, err %v", got, err)
	}
	if got := truncateRunes("éclair", 2); got != "éc…" {
		t.Fatalf("unexpected truncation: %q", got)
	}
	if got := truncateRunes("short", 10); got != "short" {
		t.Fatalf("unexpected short value: %q", got)
	}
}

func runTestPhase(projectDir, logDir, timestamp, provider string, iteration int) ([]string, error) {
	lineCh := make(chan string, 64)
	doneCh := make(chan error, 1)
	cancel := RunPhase(projectDir, logDir, timestamp, provider, iteration, "system prompt", Options{
		Provider: provider,
		Model:    "test-model",
		MaxTurns: 2,
		Feedback: "be precise",
		Language: "en",
	}, lineCh, doneCh)
	defer cancel()
	var lines []string
	for line := range lineCh {
		lines = append(lines, line)
	}
	return lines, <-doneCh
}

func writeExecutable(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
