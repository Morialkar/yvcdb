package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	gitops "github.com/Morialkar/yvcdb/internal/git"
	"github.com/Morialkar/yvcdb/internal/phases"
	"github.com/Morialkar/yvcdb/internal/runner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestViewsAndPhasePresentation(t *testing.T) {
	m := newTestModel(t)
	if got := m.View(); !strings.Contains(got, "YVCDB") || !strings.Contains(got, "CLAUDE model") {
		t.Fatalf("unexpected model view: %s", got)
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 35})
	m = updated.(Model)
	if m.viewport.Width != 94 || m.viewport.Height != 15 {
		t.Fatalf("unexpected viewport: %dx%d", m.viewport.Width, m.viewport.Height)
	}
	if m.phaseTitle("security") == "security" || m.phaseTitle("unknown") != "unknown" {
		t.Fatal("phase title lookup failed")
	}

	m.state = stateGitSetup
	if got := m.View(); !strings.Contains(got, "Initialize git") {
		t.Fatalf("unexpected git view: %s", got)
	}

	m.state = stateStage
	m.runs = []*phaseRun{{phaseIdx: 0, iteration: 2, status: runActive, workDir: m.ProjectDir, lines: []string{"line"}}}
	m.refreshViewport()
	if got := m.View(); !strings.Contains(got, "Iteration 2") || !strings.Contains(got, "line") {
		t.Fatalf("unexpected stage view: %s", got)
	}
	m.runs[0].status = runDecision
	if got := m.renderStage(); !strings.Contains(got, "Approved") {
		t.Fatalf("decision missing: %s", got)
	}
	m.runs[0].status = runFailed
	m.runs[0].errMsg = "failure"
	if got := m.renderStage(); !strings.Contains(got, "run failed") || !strings.Contains(got, "failure") {
		t.Fatalf("failure missing: %s", got)
	}

	m.runs = []*phaseRun{
		{phaseIdx: 2, status: runActive},
		{phaseIdx: 3, status: runDecision},
		{phaseIdx: 4, status: runApproved},
	}
	if got := m.renderTabs(); !strings.Contains(got, "[1]") || !strings.Contains(got, "[3]") {
		t.Fatalf("tabs missing: %s", got)
	}

	m.state = stateFeedback
	if got := m.View(); !strings.Contains(got, "refinement instructions") {
		t.Fatalf("feedback view missing: %s", got)
	}
	m.state = stateChecklist
	if got := m.View(); !strings.Contains(got, "Final checklist") {
		t.Fatalf("checklist missing: %s", got)
	}
	m.checkItems[0].Done = true
	m.checkItems[0].Checked = false
	m.state = stateDone
	if got := m.View(); !strings.Contains(got, "Failed criteria") {
		t.Fatalf("failed summary missing: %s", got)
	}
	for i := range m.checkItems {
		m.checkItems[i] = ChecklistItem{Label: m.checkItems[i].Label, Done: true, Checked: true}
	}
	m.checkScore = len(m.checkItems)
	if got := m.renderDone(); !strings.Contains(got, "production-ready") {
		t.Fatalf("success summary missing: %s", got)
	}
}

func TestResumePromptViewShowsCandidate(t *testing.T) {
	workflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		t.Fatal(err)
	}
	marker := &runner.ResumeMarker{
		WorkflowMode:   workflow.Mode,
		PhaseIndex:     0,
		PhaseID:        workflow.Phases[0].ID,
		Iteration:      3,
		BranchName:     "refactor/test/diagnostic",
		Provider:       "claude",
		Model:          "sonnet",
		PromptFilePath: "/tmp/.yvcdb_refactor_iter3.md",
	}
	m := NewModel(t.TempDir(), 0, true, "claude", "sonnet", 2, "en", testPrompts(), marker, workflow)
	if m.state != stateResumePrompt {
		t.Fatalf("expected resume state, got %v", m.state)
	}
	view := m.View()
	for _, want := range []string{"Resume interrupted phase", "Workflow mode: refactor", "Iteration: 3", "Branch: refactor/test/diagnostic", workflow.Phases[0].Label, m.phaseTitle(workflow.Phases[0].ID)} {
		if !strings.Contains(view, want) {
			t.Fatalf("resume view missing %q: %s", want, view)
		}
	}
}

func TestDiscardResumeCandidateRemovesArtifactsAndReturnsToModelSelect(t *testing.T) {
	dir := newGitRepo(t)
	workflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		t.Fatal(err)
	}
	promptFile := filepath.Join(dir, ".yvcdb_refactor_iter2.md")
	if err := os.WriteFile(promptFile, []byte("prompt"), 0o644); err != nil {
		t.Fatal(err)
	}
	marker := &runner.ResumeMarker{
		WorkflowMode:   workflow.Mode,
		PhaseIndex:     0,
		PhaseID:        workflow.Phases[0].ID,
		Iteration:      2,
		BranchName:     "main",
		Provider:       "claude",
		Model:          "sonnet",
		PromptFilePath: promptFile,
		LogFilePath:    filepath.Join(dir, "refactor-logs", "ts_diagnostic_iter2.md"),
	}
	if err := runner.WriteResumeMarker(filepath.Join(dir, ".yvcdb_resume.json"), *marker); err != nil {
		t.Fatal(err)
	}
	m := NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), marker, workflow)
	before, err := gitops.CurrentBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := m.handleKey(key(tea.KeyRunes, 'd'))
	m = updated.(Model)
	after, err := gitops.CurrentBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("discard should not touch git branch: before=%s after=%s", before, after)
	}
	if m.state != stateModelSelect {
		t.Fatalf("expected model select after discard, got %v", m.state)
	}
	if m.ResumeCandidate != nil {
		t.Fatal("resume candidate should be cleared")
	}
	if _, err := os.Stat(filepath.Join(dir, ".yvcdb_resume.json")); !os.IsNotExist(err) {
		t.Fatalf("resume marker should be removed, got err=%v", err)
	}
	if _, err := os.Stat(promptFile); !os.IsNotExist(err) {
		t.Fatalf("prompt file should be removed, got err=%v", err)
	}
}

func TestResumeInterruptedPhaseUsesRecordedProviderModelAndIteration(t *testing.T) {
	dir := newGitRepo(t)
	installFakeClaude(t)
	workflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		t.Fatal(err)
	}
	timestamp := "20260101_010203"
	branch := "refactor/" + timestamp + "/" + workflow.Phases[0].ID
	runTestGit(t, dir, "checkout", "-b", branch)
	marker := &runner.ResumeMarker{
		WorkflowMode:     workflow.Mode,
		PhaseIndex:       0,
		PhaseID:          workflow.Phases[0].ID,
		Iteration:        3,
		BranchName:       branch,
		Provider:         "claude",
		Model:            "sonnet",
		SessionTimestamp: timestamp,
	}
	m := NewModel(dir, 0, false, "opencode", "wrong", 2, "en", testPrompts(), marker, workflow)
	updated, _ := m.handleKey(key(tea.KeyRunes, 'r'))
	m = updated.(Model)
	if m.state != stateStage {
		t.Fatalf("expected stage state after resume, got %v", m.state)
	}
	if m.Provider != marker.Provider || m.AgentModel != marker.Model {
		t.Fatalf("resume should use recorded provider/model, got %s/%s", m.Provider, m.AgentModel)
	}
	if m.StartPhase != marker.PhaseIndex {
		t.Fatalf("resume should use recorded start phase, got %d", m.StartPhase)
	}
	if m.stageIdx != 0 {
		t.Fatalf("unexpected stage index %d", m.stageIdx)
	}
	if len(m.runs) != 1 {
		t.Fatalf("expected one resumed run, got %d", len(m.runs))
	}
	if m.runs[0].phaseIdx != marker.PhaseIndex || m.runs[0].iteration != marker.Iteration {
		t.Fatalf("resumed run mismatch: %+v", m.runs[0])
	}
	if m.runs[0].branch != marker.BranchName {
		t.Fatalf("expected resumed branch %q, got %q", marker.BranchName, m.runs[0].branch)
	}
	if m.timestamp != timestamp {
		t.Fatalf("expected resumed timestamp %q, got %q", timestamp, m.timestamp)
	}
	cancelRuns(m)
	m = drainActiveRuns(t, m)
}

func TestResumePromptDeclinesParallelStage(t *testing.T) {
	dir := newGitRepo(t)
	workflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		t.Fatal(err)
	}
	workflow.Stages = [][]int{{0, 1}}
	marker := &runner.ResumeMarker{
		WorkflowMode:   workflow.Mode,
		PhaseIndex:     0,
		PhaseID:        workflow.Phases[0].ID,
		Iteration:      2,
		BranchName:     "main",
		Provider:       "claude",
		Model:          "sonnet",
		PromptFilePath: filepath.Join(dir, ".yvcdb_refactor_iter2.md"),
	}
	if err := os.WriteFile(marker.PromptFilePath, []byte("prompt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runner.WriteResumeMarker(filepath.Join(dir, ".yvcdb_resume.json"), *marker); err != nil {
		t.Fatal(err)
	}
	m := NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), marker, workflow)
	updated, _ := m.handleKey(key(tea.KeyRunes, 'r'))
	m = updated.(Model)
	if m.state != stateModelSelect {
		t.Fatalf("expected fallback to model select, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "parallel stage is not supported") {
		t.Fatalf("expected parallel-stage warning, got %q", m.statusMsg)
	}
	if _, err := os.Stat(filepath.Join(dir, ".yvcdb_resume.json")); !os.IsNotExist(err) {
		t.Fatalf("resume marker should be removed, got err=%v", err)
	}
	if _, err := os.Stat(marker.PromptFilePath); !os.IsNotExist(err) {
		t.Fatalf("prompt file should be removed, got err=%v", err)
	}
}

func TestResumePromptBranchMissingFallsBack(t *testing.T) {
	dir := newGitRepo(t)
	workflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		t.Fatal(err)
	}
	promptFile := filepath.Join(dir, ".yvcdb_refactor_iter2.md")
	if err := os.WriteFile(promptFile, []byte("prompt"), 0o644); err != nil {
		t.Fatal(err)
	}
	marker := &runner.ResumeMarker{
		WorkflowMode:   workflow.Mode,
		PhaseIndex:     0,
		PhaseID:        workflow.Phases[0].ID,
		Iteration:      2,
		BranchName:     "missing-branch",
		Provider:       "claude",
		Model:          "sonnet",
		PromptFilePath: promptFile,
	}
	if err := runner.WriteResumeMarker(filepath.Join(dir, ".yvcdb_resume.json"), *marker); err != nil {
		t.Fatal(err)
	}
	m := NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), marker, workflow)
	updated, _ := m.handleKey(key(tea.KeyRunes, 'r'))
	m = updated.(Model)
	if m.state != stateModelSelect {
		t.Fatalf("expected fallback to model select, got %v", m.state)
	}
	if !strings.Contains(m.statusMsg, "no longer exists") {
		t.Fatalf("expected missing-branch warning, got %q", m.statusMsg)
	}
	if _, err := os.Stat(filepath.Join(dir, ".yvcdb_resume.json")); !os.IsNotExist(err) {
		t.Fatalf("resume marker should be removed, got err=%v", err)
	}
	if _, err := os.Stat(promptFile); !os.IsNotExist(err) {
		t.Fatalf("prompt file should be removed, got err=%v", err)
	}
}

func TestGreenfieldWorkflowUsesManagedChecklistAndStandards(t *testing.T) {
	dir := t.TempDir()
	workflow, err := phases.ForMode(phases.ModeGreenfield)
	if err != nil {
		t.Fatal(err)
	}
	prompts := make(map[string]string, len(workflow.Phases))
	for _, phase := range workflow.Phases {
		prompts[phase.ID] = "prompt for " + phase.ID
	}
	m := NewModel(dir, 0, true, "claude", "sonnet", 2, "en", prompts, nil, workflow)
	if len(m.Workflow.Phases) != 7 || len(m.checkItems) != 9 {
		t.Fatalf("unexpected greenfield model: phases=%d checklist=%d", len(m.Workflow.Phases), len(m.checkItems))
	}
	if err := os.WriteFile(filepath.Join(dir, "AFTER_STANDARDS.md"), []byte("Always test error paths."), 0o644); err != nil {
		t.Fatal(err)
	}
	run := &phaseRun{phaseIdx: 3, iteration: 2, workDir: dir}
	systemPrompt, err := m.phaseSystemPrompt(run, workflow.Phases[3])
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"prompt for foundation", "AFTER operating rules", "Always test error paths.", "iteration 2"} {
		if !strings.Contains(systemPrompt, required) {
			t.Fatalf("system prompt missing %q", required)
		}
	}
}

func TestFeatureWorkflowUsesManagedChecklistAndStandards(t *testing.T) {
	dir := t.TempDir()
	workflow, err := phases.ForMode(phases.ModeFeature)
	if err != nil {
		t.Fatal(err)
	}
	prompts := make(map[string]string, len(workflow.Phases))
	for _, phase := range workflow.Phases {
		prompts[phase.ID] = "prompt for " + phase.ID
	}
	m := NewModel(dir, 0, true, "claude", "sonnet", 2, "fr", prompts, nil, workflow)
	if len(m.Workflow.Phases) != 6 || len(m.checkItems) != 9 {
		t.Fatalf("unexpected feature model: phases=%d checklist=%d", len(m.Workflow.Phases), len(m.checkItems))
	}
	if err := os.WriteFile(filepath.Join(dir, "AFTER_STANDARDS.md"), []byte("Always test feature edge cases."), 0o644); err != nil {
		t.Fatal(err)
	}
	run := &phaseRun{phaseIdx: 3, iteration: 1, workDir: dir}
	systemPrompt, err := m.phaseSystemPrompt(run, workflow.Phases[3])
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"prompt for implementation", m.l10n.Pick("AFTER operating rules", "Règles d'opération AFTER"), "Always test feature edge cases."} {
		if !strings.Contains(systemPrompt, required) {
			t.Fatalf("system prompt missing %q", required)
		}
	}
}

func TestDebugWorkflowUsesManagedChecklistAndStandards(t *testing.T) {
	dir := t.TempDir()
	workflow, err := phases.ForMode(phases.ModeDebug)
	if err != nil {
		t.Fatal(err)
	}
	prompts := make(map[string]string, len(workflow.Phases))
	for _, phase := range workflow.Phases {
		prompts[phase.ID] = "prompt for " + phase.ID
	}
	m := NewModel(dir, 0, true, "claude", "sonnet", 2, "en", prompts, nil, workflow)
	if len(m.Workflow.Phases) != 6 || len(m.checkItems) != 9 {
		t.Fatalf("unexpected debug model: phases=%d checklist=%d", len(m.Workflow.Phases), len(m.checkItems))
	}
	if err := os.WriteFile(filepath.Join(dir, "AFTER_STANDARDS.md"), []byte("Always reproduce the bug before fixing it."), 0o644); err != nil {
		t.Fatal(err)
	}
	run := &phaseRun{phaseIdx: 3, iteration: 1, workDir: dir}
	systemPrompt, err := m.phaseSystemPrompt(run, workflow.Phases[3])
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"prompt for fix", "AFTER operating rules", "Always reproduce the bug before fixing it."} {
		if !strings.Contains(systemPrompt, required) {
			t.Fatalf("system prompt missing %q", required)
		}
	}
}

func TestKeyboardStateTransitions(t *testing.T) {
	m := newTestModel(t)
	m.stageIdx = len(m.Workflow.Stages)
	m.input.SetValue("sonnet")
	updated, _ := m.handleKey(key(tea.KeyEnter, 0))
	m = updated.(Model)
	if m.state != stateChecklist {
		t.Fatalf("expected checklist, got %v", m.state)
	}

	updated, _ = m.handleKey(key(tea.KeyRunes, 'y'))
	m = updated.(Model)
	if !m.checkItems[0].Checked || m.checkIdx != 1 {
		t.Fatal("yes did not advance checklist")
	}
	updated, _ = m.handleKey(key(tea.KeyRunes, 'n'))
	m = updated.(Model)
	if m.checkItems[1].Checked || m.checkIdx != 2 {
		t.Fatal("no did not advance checklist")
	}

	m.state = stateStage
	m.runs = []*phaseRun{{phaseIdx: 2}, {phaseIdx: 3}, {phaseIdx: 4}}
	m.activeRun = 0
	updated, _ = m.handleKey(key(tea.KeyTab, 0))
	m = updated.(Model)
	if m.activeRun != 1 {
		t.Fatal("tab did not switch runs")
	}
	updated, _ = m.handleKey(key(tea.KeyRunes, '3'))
	m = updated.(Model)
	if m.activeRun != 2 {
		t.Fatal("numeric key did not select run")
	}

	m.runs[2].status = runFailed
	updated, _ = m.handleKey(key(tea.KeyRunes, 's'))
	m = updated.(Model)
	if m.runs[2].status != runSkipped {
		t.Fatal("failed run was not skipped")
	}

	m.state = stateFeedback
	m.nextState = stateStage
	m.input.SetValue("")
	updated, _ = m.handleKey(key(tea.KeyEsc, 0))
	m = updated.(Model)
	if m.state != stateStage {
		t.Fatal("escape did not leave feedback")
	}
}

func TestFullNoGitPipelineAndFixLoop(t *testing.T) {
	installFakeClaude(t)
	m := newTestModel(t)
	m.state = stateStage
	m.stageIdx = 0
	m, _ = m.startStage()

	for m.state == stateStage {
		m = drainActiveRuns(t, m)
		if m.stageIdx == 0 && m.runs[0].iteration == 1 {
			m, _ = m.reiterateRunWithFeedback(0, "cover this edge case")
			continue
		}
		// approve one decision at a time: approving can complete the stage and
		// replace m.runs with the next stage's runs
		for {
			approved := false
			for i := range m.runs {
				if m.runs[i].status == runDecision {
					m, _ = m.approveRun(i)
					approved = true
					break
				}
			}
			if !approved || m.state != stateStage {
				break
			}
			hasDecision := false
			for i := range m.runs {
				if m.runs[i].status == runDecision {
					hasDecision = true
				}
			}
			if !hasDecision {
				break
			}
		}
	}
	if m.state != stateChecklist {
		t.Fatalf("pipeline ended in state %v", m.state)
	}

	// Fail one item and accept the rest.
	m, _ = updateKeyModel(t, m, key(tea.KeyRunes, 'n'))
	for m.state == stateChecklist {
		m, _ = updateKeyModel(t, m, key(tea.KeyRunes, 'y'))
	}
	if m.failedChecks() != 1 || m.state != stateDone {
		t.Fatal("expected one failed final check")
	}
	m, _ = updateKeyModel(t, m, key(tea.KeyRunes, 'f'))
	if m.state != stateFixRun {
		t.Fatal("fix loop did not start")
	}
	m = drainActiveRuns(t, m)
	m, _ = m.approveRun(0)
	if m.state != stateDone {
		t.Fatal("fix approval did not return to done")
	}
}

func TestGitSetupCommandAndCommitChanges(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_AUTHOR_NAME", "YVCDB Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "YVCDB Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.invalid")

	m := NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), nil)
	msg := m.doGitInit()()
	done, ok := msg.(gitSetupDoneMsg)
	if !ok || done.err != nil || !done.useGit {
		t.Fatalf("git setup failed: %#v", msg)
	}
	m.useGit = true
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.commitChanges(dir, "test commit"); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", dir, "log", "-1", "--pretty=%s")
	output, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) != "test commit" {
		t.Fatalf("unexpected commit: %q, %v", output, err)
	}
}

func TestParallelStageIntegration(t *testing.T) {
	installFakeClaude(t)
	dir := t.TempDir()
	runTestGit(t, dir, "init", "-b", "main")
	runTestGit(t, dir, "config", "user.name", "YVCDB Test")
	runTestGit(t, dir, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.CommitAll(dir, "base"); err != nil {
		t.Fatal(err)
	}

	m := NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), nil)
	m.Workflow.Stages = [][]int{{0}, {1}, {2, 3, 4}, {5}}
	m.state = stateStage
	m.stageIdx = 2
	for i, phaseIdx := range []int{2, 3, 4} {
		worktree := filepath.Join(t.TempDir(), phases.All[phaseIdx].ID)
		branch := "test/" + phases.All[phaseIdx].ID
		if err := gitops.WorktreeAdd(dir, worktree, branch); err != nil {
			t.Fatal(err)
		}
		parent := filepath.Dir(worktree)
		t.Cleanup(func() { _ = os.RemoveAll(parent) })
		status := runApproved
		if i == 1 {
			status = runSkipped
		} else {
			name := phases.All[phaseIdx].ID + ".txt"
			if err := os.WriteFile(filepath.Join(worktree, name), []byte(name+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := gitops.CommitAll(worktree, "add "+name); err != nil {
				t.Fatal(err)
			}
		}
		m.runs = append(m.runs, &phaseRun{phaseIdx: phaseIdx, workDir: worktree, branch: branch, status: status})
	}

	m, _ = m.checkStageDone()
	if m.stageIdx != 3 || len(m.runs) != 1 || m.runs[0].phaseIdx != 5 {
		t.Fatalf("next stage did not start: stage=%d runs=%d", m.stageIdx, len(m.runs))
	}
	for _, name := range []string{"security.txt", "readability.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("merged file %s missing: %v", name, err)
		}
	}
	for _, run := range m.runs {
		if run.cancel != nil {
			run.cancel()
		}
	}
	m = drainActiveRuns(t, m)
}

func TestAdditionalKeyPathsAndCancellation(t *testing.T) {
	installFakeClaude(t)
	m := newTestModel(t)
	cancelled := false
	m.runs = []*phaseRun{{phaseIdx: 0, status: runActive, cancel: func() { cancelled = true }}}
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)
	if !cancelled {
		t.Fatal("quit did not cancel active run")
	}
	if m.Init() != nil {
		t.Fatal("Init should not schedule work")
	}

	m = newTestModel(t)
	m.input.SetValue("")
	updated, _ = m.handleKey(key(tea.KeyEnter, 0))
	m = updated.(Model)
	if m.state != stateModelSelect {
		t.Fatal("empty model should not advance")
	}
	updated, _ = m.handleKey(key(tea.KeyEsc, 0))
	m = updated.(Model)

	m = newTestModel(t)
	m.state = stateGitSetup
	updated, cmd := m.handleKey(key(tea.KeyRunes, 'n'))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("git setup decline should return a command")
	}
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)
	for _, run := range m.runs {
		if run.cancel != nil {
			run.cancel()
		}
	}

	m = newTestModel(t)
	m.state = stateStage
	m.runs = []*phaseRun{{phaseIdx: 0, iteration: 1, status: runDecision, workDir: m.ProjectDir}}
	m.activeRun = 0
	updated, _ = m.handleKey(key(tea.KeyRunes, 'f'))
	m = updated.(Model)
	if m.state != stateFeedback {
		t.Fatal("feedback state did not open")
	}
	m.input.SetValue("refine this")
	updated, _ = m.handleKey(key(tea.KeyEnter, 0))
	m = updated.(Model)
	if m.state != stateStage || m.runs[0].iteration != 2 {
		t.Fatal("feedback was not submitted")
	}
	m = drainActiveRuns(t, m)
	m, _ = m.reiterateRun(0)
	if m.runs[0].iteration != 3 {
		t.Fatal("retry did not increment iteration")
	}
	for _, run := range m.runs {
		if run.cancel != nil {
			run.cancel()
		}
	}
	m = drainActiveRuns(t, m)

	m = newTestModel(t)
	m.state = stateFixRun
	m.runs = []*phaseRun{{phaseIdx: 5, status: runDecision}}
	m, _ = m.skipRun(0)
	if m.state != stateDone {
		t.Fatal("skipping fix did not finish")
	}
}

func newTestModel(t *testing.T) Model {
	t.Helper()
	return NewModel(t.TempDir(), 0, true, "claude", "sonnet", 2, "en", testPrompts(), nil)
}

func testPrompts() map[string]string {
	prompts := make(map[string]string, len(phases.All))
	for _, phase := range phases.All {
		prompts[phase.ID] = "test prompt for " + phase.ID
	}
	return prompts
}

func installFakeClaude(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "claude")
	script := `#!/bin/sh
printf '%s\n' '{"type":"assistant","message":{"content":[{"type":"text","text":"analysis"}]}}'
printf '%s\n' '{"type":"result","result":"done"}'
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func drainActiveRuns(t *testing.T, m Model) Model {
	t.Helper()
	for slot, run := range m.runs {
		if run.status != runActive {
			continue
		}
		cmd := waitForRun(slot, run.lineCh, run.doneCh)
		for m.runs[slot].status == runActive {
			msg := cmd()
			updated, next := m.Update(msg)
			m = updated.(Model)
			cmd = next
		}
	}
	return m
}

func updateKeyModel(t *testing.T, m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

func key(keyType tea.KeyType, r rune) tea.KeyMsg {
	msg := tea.KeyMsg{Type: keyType}
	if keyType == tea.KeyRunes {
		msg.Runes = []rune{r}
	}
	return msg
}

func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"-C", dir}, args...)
	if output, err := exec.Command("git", cmdArgs...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
