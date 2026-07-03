package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// streamEvent is a partial decode of claude's stream-json format.
type streamEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`

	// assistant message
	Message *struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
			// tool use
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	} `json:"message"`

	// tool result
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`

	// result (final)
	Result string `json:"result"`
}

func eventToLines(ev streamEvent, language string) []string {
	var lines []string

	switch ev.Type {
	case "assistant":
		if ev.Message == nil {
			return nil
		}
		for _, block := range ev.Message.Content {
			switch block.Type {
			case "text":
				if t := strings.TrimSpace(block.Text); t != "" {
					lines = append(lines, t)
				}
			case "tool_use":
				input := strings.TrimSpace(string(block.Input))
				if len(input) > 120 {
					input = input[:120] + "…"
				}
				lines = append(lines, fmt.Sprintf("  ⚙  %s  %s", block.Name, input))
			}
		}

	case "tool_result":
		snippet := strings.TrimSpace(ev.Content)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "…"
		}
		if snippet != "" {
			lines = append(lines, fmt.Sprintf("  ↳  %s", snippet))
		}

	case "result":
		if t := strings.TrimSpace(ev.Result); t != "" {
			lines = append(lines, "")
			heading := "─── Final result ───"
			if language == "fr" {
				heading = "─── Résultat final ───"
			}
			lines = append(lines, heading)
			lines = append(lines, t)
		}
	}

	return lines
}

type Options struct {
	Provider string
	Model    string
	MaxTurns int
	Feedback string
	Language string
}

type codexEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Message  string `json:"message"`
	Item     *struct {
		Type             string `json:"type"`
		Text             string `json:"text"`
		Command          string `json:"command"`
		AggregatedOutput string `json:"aggregated_output"`
		Status           string `json:"status"`
	} `json:"item"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func codexEventToLines(ev codexEvent) []string {
	if strings.TrimSpace(ev.Message) != "" && (ev.Type == "error" || ev.Type == "turn.failed") {
		return []string{"  [error] " + ev.Message}
	}
	if ev.Error != nil && strings.TrimSpace(ev.Error.Message) != "" {
		return []string{"  [error] " + ev.Error.Message}
	}
	if ev.Item == nil {
		return nil
	}
	item := ev.Item
	switch item.Type {
	case "agent_message":
		if ev.Type == "item.completed" {
			if text := strings.TrimSpace(item.Text); text != "" {
				return []string{text}
			}
		}
	case "reasoning":
		if ev.Type == "item.completed" {
			if text := strings.TrimSpace(item.Text); text != "" {
				return []string{"  ◇  " + text}
			}
		}
	case "command_execution":
		if ev.Type == "item.started" {
			return []string{"  ⚙  " + strings.TrimSpace(item.Command)}
		}
		if ev.Type == "item.completed" {
			output := strings.TrimSpace(item.AggregatedOutput)
			if output == "" {
				return nil
			}
			if len(output) > 300 {
				output = output[:300] + "…"
			}
			return []string{"  ↳  " + output}
		}
	case "file_change":
		if ev.Type == "item.completed" {
			return []string{"  ✎  files changed"}
		}
	case "mcp_tool_call":
		if ev.Type == "item.started" {
			return []string{"  ⚙  MCP tool call"}
		}
	}
	/*
		Keep item handling intentionally limited to stable, human-readable fields.
		Unknown Codex event and item types remain available in the raw log.
	*/
	return nil
}

// RunPhase launches the configured agent CLI and forwards parsed output to lineCh.
func RunPhase(projectDir, logDir, timestamp, phaseID string, iteration int, systemPrompt string, opts Options, lineCh chan<- string, doneCh chan<- error) {
	go func() {
		logFile := filepath.Join(logDir, fmt.Sprintf("%s_%s_iter%d.md", timestamp, phaseID, iteration))
		if err := os.MkdirAll(logDir, 0755); err != nil {
			close(lineCh)
			doneCh <- fmt.Errorf("mkdir logs: %w", err)
			return
		}

		f, err := os.Create(logFile)
		if err != nil {
			close(lineCh)
			doneCh <- fmt.Errorf("create log: %w", err)
			return
		}
		defer f.Close()

		projectLabel := "Project"
		if opts.Language == "fr" {
			projectLabel = "Projet"
		}
		fmt.Fprintf(f, "# Refactoring — %s — iter%d\nDate: %s\n%s: %s\n\n---\n\n",
			phaseID, iteration, time.Now().Format("2006-01-02 15:04:05"), projectLabel, projectDir)

		english := opts.Language != "fr"
		userPrompt := "Analyze the project in the current directory and follow the instructions for your phase. Respond in English."
		if !english {
			userPrompt = "Analyse le projet dans le répertoire courant et applique les instructions de ta phase. Réponds en français."
		}
		if iteration > 1 {
			prevLog := filepath.Join(logDir, fmt.Sprintf("%s_%s_iter%d.md", timestamp, phaseID, iteration-1))
			if prev, err := readFirstNLines(prevLog, 100); err == nil {
				heading := "\n\n## Previous iteration result (to improve)\n"
				if !english {
					heading = "\n\n## Résultat de l'itération précédente (à améliorer)\n"
				}
				userPrompt += heading + prev
			}
		}
		if feedback := strings.TrimSpace(opts.Feedback); feedback != "" {
			heading := "\n\n## Specific user feedback\n"
			instruction := "\n\nAddress this feedback explicitly, preserve what was satisfactory, and modify the project accordingly."
			if !english {
				heading = "\n\n## Retour précis de l'utilisateur\n"
				instruction = "\n\nTraite explicitement ce retour, conserve ce qui était satisfaisant et modifie le projet en conséquence."
			}
			userPrompt += heading + feedback + instruction
		}

		maxTurns := opts.MaxTurns
		if maxTurns <= 0 {
			maxTurns = 20
		}
		provider := opts.Provider
		if provider == "" {
			provider = "claude"
		}
		var cmd *exec.Cmd
		if provider == "codex" {
			prompt := systemPrompt + "\n\n---\n\n" + userPrompt
			args := []string{"-a", "never", "exec", "--json", "--color", "never", "--sandbox", "workspace-write", "--skip-git-repo-check", "--ephemeral", "-C", projectDir}
			if model := strings.TrimSpace(opts.Model); model != "" {
				args = append(args, "--model", model)
			}
			args = append(args, prompt)
			cmd = exec.Command("codex", args...)
		} else {
			args := []string{
				"-p", userPrompt,
				"--append-system-prompt", systemPrompt,
				"--allowedTools", "Read,Write,Edit,Bash,Glob,Grep",
				"--output-format", "stream-json",
				"--verbose",
				"--max-turns", fmt.Sprint(maxTurns),
			}
			if model := strings.TrimSpace(opts.Model); model != "" {
				args = append(args, "--model", model)
			}
			cmd = exec.Command("claude", args...)
		}
		cmd.Dir = projectDir

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			close(lineCh)
			doneCh <- err
			return
		}
		// stderr: forward raw (warnings, auth errors)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			close(lineCh)
			doneCh <- err
			return
		}

		if err := cmd.Start(); err != nil {
			close(lineCh)
			doneCh <- fmt.Errorf("Claude CLI not found or failed to start: %w", err)
			return
		}

		// stderr forwarded as-is in background. Wait before closing lineCh.
		var stderrWG sync.WaitGroup
		stderrWG.Add(1)
		go func() {
			defer stderrWG.Done()
			sc := bufio.NewScanner(stderr)
			for sc.Scan() {
				line := sc.Text()
				if strings.TrimSpace(line) != "" {
					lineCh <- "  [stderr] " + line
				}
			}
		}()

		// stdout: parse stream-json, emit human-readable lines
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024) // 1 MB per line — JSON events can be large
		for sc.Scan() {
			raw := sc.Text()
			fmt.Fprintln(f, raw)

			if provider == "codex" {
				var ev codexEvent
				if err := json.Unmarshal([]byte(raw), &ev); err != nil {
					if t := strings.TrimSpace(raw); t != "" {
						lineCh <- t
					}
					continue
				}
				for _, line := range codexEventToLines(ev) {
					lineCh <- line
				}
				continue
			}

			var ev streamEvent
			if err := json.Unmarshal([]byte(raw), &ev); err != nil {
				// not JSON (startup messages etc.) — forward as-is
				if t := strings.TrimSpace(raw); t != "" {
					lineCh <- t
				}
				continue
			}

			for _, line := range eventToLines(ev, opts.Language) {
				lineCh <- line
			}
		}

		waitErr := cmd.Wait()
		stderrWG.Wait()
		close(lineCh)
		if provider == "codex" && waitErr != nil {
			doneCh <- fmt.Errorf("Codex CLI failed: %w", waitErr)
			return
		}
		doneCh <- nil
	}()
}

func readFirstNLines(path string, n int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() && len(lines) < n {
		lines = append(lines, scanner.Text())
	}
	return strings.Join(lines, "\n"), scanner.Err()
}
