package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultMaxTurns is the Claude turn limit used when no positive limit is supplied.
	DefaultMaxTurns          = 20
	DefaultInactivityTimeout = 10 * time.Minute
	previousLogLines         = 100
	maxJSONEventBytes        = 1024 * 1024
	toolInputRunes           = 120
	toolResultRunes          = 200
	commandOutputRunes       = 300
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
	Content string `json:"content"`

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
				input = truncateRunes(input, toolInputRunes)
				lines = append(lines, fmt.Sprintf("  ⚙  %s  %s", block.Name, input))
			}
		}

	case "tool_result":
		snippet := strings.TrimSpace(ev.Content)
		snippet = truncateRunes(snippet, toolResultRunes)
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

// Options controls an agent phase execution.
type Options struct {
	Provider          string
	Model             string
	MaxTurns          int
	Feedback          string
	Language          string
	InactivityTimeout time.Duration
}

type codexEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Item    *struct {
		Type             string `json:"type"`
		Text             string `json:"text"`
		Command          string `json:"command"`
		AggregatedOutput string `json:"aggregated_output"`
	} `json:"item"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type inactivityWatchdog struct {
	activity chan struct{}
	stopCh   chan struct{}
	done     chan struct{}
	fired    chan struct{}
}

func newInactivityWatchdog(timeout time.Duration, cancel context.CancelFunc) *inactivityWatchdog {
	w := &inactivityWatchdog{
		activity: make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
		done:     make(chan struct{}),
		fired:    make(chan struct{}),
	}
	go func() {
		defer close(w.done)
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		reset := func() {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(timeout)
		}
		for {
			select {
			case <-timer.C:
				select {
				case <-w.fired:
				default:
					close(w.fired)
				}
				cancel()
				return
			case <-w.activity:
				reset()
			case <-w.stopCh:
				return
			}
		}
	}()
	return w
}

func (w *inactivityWatchdog) recordActivity() {
	select {
	case w.activity <- struct{}{}:
	default:
	}
}

func (w *inactivityWatchdog) stop() {
	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}
	<-w.done
}

func (w *inactivityWatchdog) firedNow() bool {
	select {
	case <-w.fired:
		return true
	default:
		return false
	}
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
			output = truncateRunes(output, commandOutputRunes)
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

// RunPhase launches the configured agent CLI, forwards parsed output to lineCh,
// and returns a function that cancels the subprocess.
func RunPhase(projectDir, logDir, timestamp, phaseID string, iteration int, systemPrompt string, opts Options, lineCh chan<- string, doneCh chan<- error) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		logFile := filepath.Join(logDir, fmt.Sprintf("%s_%s_iter%d.md", timestamp, phaseID, iteration))
		if err := os.MkdirAll(logDir, 0o755); err != nil {
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
		var watchdog *inactivityWatchdog
		promptFilePath := ""
		cleanupPromptFile := func() error {
			if promptFilePath == "" {
				return nil
			}
			if err := os.Remove(promptFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove prompt file: %w", err)
			}
			return nil
		}
		finish := func(runErr error) {
			if watchdog != nil {
				watchdog.stop()
				watchdog = nil
			}
			if cleanupErr := cleanupPromptFile(); cleanupErr != nil {
				runErr = errors.Join(runErr, cleanupErr)
			}
			if closeErr := f.Close(); closeErr != nil {
				runErr = errors.Join(runErr, fmt.Errorf("close event log: %w", closeErr))
			}
			close(lineCh)
			doneCh <- runErr
		}
		failBeforeStart := func(runErr error) {
			finish(runErr)
		}
		projectLabel := "Project"
		if opts.Language == "fr" {
			projectLabel = "Projet"
		}
		if _, err := fmt.Fprintf(f, "# Refactoring — %s — iter%d\nDate: %s\n%s: %s\n\n---\n\n",
			phaseID, iteration, time.Now().Format("2006-01-02 15:04:05"), projectLabel, projectDir); err != nil {
			failBeforeStart(fmt.Errorf("write log header: %w", err))
			return
		}

		english := opts.Language != "fr"
		userPrompt := "Analyze the project in the current directory and follow the instructions for your phase. Respond in English."
		if !english {
			userPrompt = "Analyse le projet dans le répertoire courant et applique les instructions de ta phase. Réponds en français."
		}
		if iteration > 1 {
			prevLog := filepath.Join(logDir, fmt.Sprintf("%s_%s_iter%d.md", timestamp, phaseID, iteration-1))
			prev, err := readFirstNLines(prevLog, previousLogLines)
			if err != nil {
				failBeforeStart(fmt.Errorf("read previous iteration log: %w", err))
				return
			}
			heading := "\n\n## Previous iteration result (to improve)\n"
			if !english {
				heading = "\n\n## Résultat de l'itération précédente (à améliorer)\n"
			}
			userPrompt += heading + prev
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
			maxTurns = DefaultMaxTurns
		}
		inactivityTimeout := opts.InactivityTimeout
		if inactivityTimeout <= 0 {
			inactivityTimeout = DefaultInactivityTimeout
		}
		selected := selectProvider(opts.Provider)
		if selected.needsPromptFile() {
			promptFile, err := os.CreateTemp(projectDir, fmt.Sprintf(".yvcdb_%s_iter%d_*.md", phaseID, iteration))
			if err != nil {
				failBeforeStart(fmt.Errorf("create prompt file: %w", err))
				return
			}
			if _, err := promptFile.WriteString(systemPrompt); err != nil {
				_ = promptFile.Close()
				promptFilePath = promptFile.Name()
				failBeforeStart(fmt.Errorf("write prompt file: %w", err))
				return
			}
			if err := promptFile.Close(); err != nil {
				promptFilePath = promptFile.Name()
				failBeforeStart(fmt.Errorf("close prompt file: %w", err))
				return
			}
			promptFilePath = promptFile.Name()
			if notice := strings.TrimSpace(selected.startupNotice(opts.Language)); notice != "" {
				if _, err := fmt.Fprintln(f, notice); err != nil {
					failBeforeStart(fmt.Errorf("write startup notice: %w", err))
					return
				}
				lineCh <- notice
			}
		}
		cmd := selected.buildCommand(ctx, projectDir, systemPrompt, userPrompt, opts.Model, promptFilePath, maxTurns)
		cmd.Dir = projectDir

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			failBeforeStart(fmt.Errorf("open %s stdout: %w", selected.name, err))
			return
		}
		// stderr: forward raw (warnings, auth errors)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			failBeforeStart(fmt.Errorf("open %s stderr: %w", selected.name, err))
			return
		}

		if err := cmd.Start(); err != nil {
			failBeforeStart(fmt.Errorf("start %s CLI: %w", selected.name, err))
			return
		}
		watchdog = newInactivityWatchdog(inactivityTimeout, cancel)

		// stderr forwarded as-is in background. Wait before closing lineCh.
		var stderrWG sync.WaitGroup
		var stderrScanErr error
		stderrWG.Add(1)
		go func() {
			defer stderrWG.Done()
			sc := bufio.NewScanner(stderr)
			for sc.Scan() {
				if watchdog != nil {
					watchdog.recordActivity()
				}
				line := sc.Text()
				if strings.TrimSpace(line) != "" {
					lineCh <- "  [stderr] " + line
				}
			}
			if err := sc.Err(); err != nil {
				stderrScanErr = fmt.Errorf("read %s stderr: %w", selected.name, err)
				// drain so the subprocess is not blocked writing to a full pipe
				_, _ = io.Copy(io.Discard, stderr)
			}
		}()

		// stdout: parse stream-json, emit human-readable lines
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, maxJSONEventBytes), maxJSONEventBytes)
		var logErr error
		maxTurnsReached := false
		for sc.Scan() {
			if watchdog != nil {
				watchdog.recordActivity()
			}
			raw := sc.Text()
			if logErr == nil {
				if _, err := fmt.Fprintln(f, raw); err != nil {
					logErr = fmt.Errorf("write agent event log: %w", err)
				}
			}

			lines, reachedMaxTurns := selected.parseLine(raw, opts.Language)
			if reachedMaxTurns {
				maxTurnsReached = true
			}
			for _, line := range lines {
				lineCh <- line
			}
		}

		stdoutScanErr := sc.Err()
		if stdoutScanErr != nil {
			// drain the pipe so the subprocess is not blocked writing to it,
			// which would deadlock cmd.Wait below
			_, _ = io.Copy(io.Discard, stdout)
		}
		// finish reading both pipes before Wait: Wait closes them
		stderrWG.Wait()
		waitErr := cmd.Wait()

		var runErr error
		if watchdog != nil && watchdog.firedNow() {
			runErr = fmt.Errorf("inactivity timeout after %s", inactivityTimeout)
		} else if waitErr != nil && !selected.waitSucceeded(waitErr, ctx.Err(), maxTurnsReached) {
			runErr = fmt.Errorf("%s CLI failed: %w", selected.name, waitErr)
		}
		if stdoutScanErr != nil {
			runErr = errors.Join(runErr, fmt.Errorf("read %s stdout: %w", selected.name, stdoutScanErr))
		}
		if stderrScanErr != nil {
			runErr = errors.Join(runErr, stderrScanErr)
		}
		if logErr != nil {
			runErr = errors.Join(runErr, logErr)
		}
		finish(runErr)
	}()
	return cancel
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
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
