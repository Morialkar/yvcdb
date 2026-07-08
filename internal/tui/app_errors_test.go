package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gitops "github.com/Morialkar/yvcdb/internal/git"
	"github.com/Morialkar/yvcdb/internal/phases"
	"github.com/Morialkar/yvcdb/internal/runner"
	tea "github.com/charmbracelet/bubbletea"
)

func newGitRepo(t *testing.T) string {
	t.Helper()
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
	return dir
}

func cancelRuns(m Model) {
	for _, run := range m.runs {
		if run.cancel != nil {
			run.cancel()
		}
	}
}

func TestNewModelDefaultsModelFromProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{name: "claude", provider: "claude", want: "sonnet"},
		{name: "codex", provider: "codex", want: "gpt-5.4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(t.TempDir(), 2, true, tt.provider, "  ", 2, "en", testPrompts(), nil)
			if m.AgentModel != tt.want {
				t.Fatalf("expected suggested %s model, got %q", tt.provider, m.AgentModel)
			}
			if m.stageIdx != 2 {
				t.Fatalf("expected stage containing phase 2, got %d", m.stageIdx)
			}
		})
	}
}

func TestResumeHelpersAcrossWorkflowModes(t *testing.T) {
	baseWorkflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		mode     string
		artifact string
	}{
		{name: "greenfield", mode: phases.ModeGreenfield, artifact: "AFTER_SPEC.md"},
		{name: "feature", mode: phases.ModeFeature, artifact: "AFTER_PLAN.md"},
		{name: "debug", mode: phases.ModeDebug, artifact: "AFTER_BUG.md"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			marker := &runner.ResumeMarker{
				WorkflowMode: tt.mode,
				PhaseIndex:   0,
				PhaseID:      baseWorkflow.Phases[0].ID,
				BranchName:   "refactor/ts/diagnostic",
			}
			m := NewModel(t.TempDir(), 0, true, "claude", "sonnet", 2, "en", testPrompts(), marker, baseWorkflow)
			if got := m.resumeWorkflow(); got.Mode != tt.mode {
				t.Fatalf("resume workflow mode = %s, want %s", got.Mode, tt.mode)
			}
			if got := m.resumeStateArtifactName(); got != tt.artifact {
				t.Fatalf("resume state artifact = %s, want %s", got, tt.artifact)
			}
		})
	}
}

func TestResumeHelpersFallbackToCurrentWorkflow(t *testing.T) {
	workflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		t.Fatal(err)
	}
	marker := &runner.ResumeMarker{
		WorkflowMode: "unknown",
		PhaseIndex:   0,
		PhaseID:      workflow.Phases[0].ID,
		BranchName:   "refactor/ts/diagnostic",
	}
	m := NewModel(t.TempDir(), 0, true, "claude", "sonnet", 2, "en", testPrompts(), marker, workflow)
	if got := m.resumeWorkflow(); got.Mode != workflow.Mode {
		t.Fatalf("resume workflow should fall back to current mode, got %s", got.Mode)
	}
	if got := m.resumeStateArtifactName(); got != "REFACTOR_STATE.md" {
		t.Fatalf("resume state artifact should fall back to refactor state, got %s", got)
	}
}

func TestRenderResumePromptBranches(t *testing.T) {
	workflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(t.TempDir(), 0, true, "claude", "sonnet", 2, "en", testPrompts(), nil, workflow)
	if got := m.renderResumePrompt(); got != "" {
		t.Fatalf("expected empty prompt without candidate, got %q", got)
	}
	marker := &runner.ResumeMarker{
		WorkflowMode: workflow.Mode,
		PhaseIndex:   -1,
		PhaseID:      "custom-phase",
		Iteration:    4,
		BranchName:   "refactor/ts/custom",
	}
	m = NewModel(t.TempDir(), 0, true, "claude", "sonnet", 2, "fr", testPrompts(), marker, workflow)
	got := m.renderResumePrompt()
	for _, want := range []string{"Reprendre la phase interrompue ?", "custom-phase", "Itération : 4", "Branche : refactor/ts/custom"} {
		if !strings.Contains(got, want) {
			t.Fatalf("resume prompt missing %q: %s", want, got)
		}
	}
}

func TestResumeInterruptedPhaseFallbackBranches(t *testing.T) {
	workflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("nil candidate", func(t *testing.T) {
		m := NewModel(t.TempDir(), 0, true, "claude", "sonnet", 2, "en", testPrompts(), nil, workflow)
		updated, _ := m.resumeInterruptedPhase()
		if updated.state != stateModelSelect {
			t.Fatalf("expected model select, got %v", updated.state)
		}
	})

	t.Run("phase not in workflow stages", func(t *testing.T) {
		dir := newGitRepo(t)
		marker := &runner.ResumeMarker{
			WorkflowMode:   workflow.Mode,
			PhaseIndex:     2,
			PhaseID:        workflow.Phases[2].ID,
			BranchName:     "refactor/ts/security",
			PromptFilePath: filepath.Join(dir, ".yvcdb_security_iter2.md"),
		}
		_ = os.WriteFile(marker.PromptFilePath, []byte("prompt"), 0o644)
		_ = runner.WriteResumeMarker(filepath.Join(dir, ".yvcdb_resume.json"), *marker)
		m := NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), marker, workflow)
		m.Workflow.Stages = [][]int{{0}, {1}, {3}, {4}, {5}}
		updated, _ := m.resumeInterruptedPhase()
		if updated.state != stateModelSelect || !strings.Contains(updated.statusMsg, "no longer exists") {
			t.Fatalf("expected missing-branch fallback, got state=%v msg=%q", updated.state, updated.statusMsg)
		}
	})

	t.Run("parallel stage declined", func(t *testing.T) {
		dir := newGitRepo(t)
		marker := &runner.ResumeMarker{
			WorkflowMode:   workflow.Mode,
			PhaseIndex:     0,
			PhaseID:        workflow.Phases[0].ID,
			BranchName:     "refactor/ts/diagnostic",
			PromptFilePath: filepath.Join(dir, ".yvcdb_diagnostic_iter2.md"),
		}
		_ = os.WriteFile(marker.PromptFilePath, []byte("prompt"), 0o644)
		_ = runner.WriteResumeMarker(filepath.Join(dir, ".yvcdb_resume.json"), *marker)
		m := NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), marker, workflow)
		m.Workflow.Stages = [][]int{{0, 1}, {2}, {3, 4}, {5}}
		updated, _ := m.resumeInterruptedPhase()
		if updated.state != stateModelSelect || !strings.Contains(updated.statusMsg, "parallel stage is not supported") {
			t.Fatalf("expected parallel fallback, got state=%v msg=%q", updated.state, updated.statusMsg)
		}
	})

	t.Run("git checkout error", func(t *testing.T) {
		marker := &runner.ResumeMarker{
			WorkflowMode:   workflow.Mode,
			PhaseIndex:     0,
			PhaseID:        workflow.Phases[0].ID,
			BranchName:     "refactor/ts/diagnostic",
			PromptFilePath: filepath.Join(t.TempDir(), ".yvcdb_diagnostic_iter2.md"),
		}
		m := NewModel(t.TempDir(), 0, false, "claude", "sonnet", 2, "en", testPrompts(), marker, workflow)
		updated, _ := m.resumeInterruptedPhase()
		if updated.state != stateModelSelect || updated.statusMsg == "" {
			t.Fatalf("expected checkout/branch error fallback, got state=%v msg=%q", updated.state, updated.statusMsg)
		}
	})
}

func TestPhaseStateCoversDoneActiveAndPending(t *testing.T) {
	m := newTestModel(t)
	m.Workflow.Stages = [][]int{{0}, {1}, {2, 3}, {4}, {5}}
	m.stageIdx = 1
	if state, iter := m.phaseState(0); state != "done" || iter != 0 {
		t.Fatalf("phase 0 should be done, got %s iter %d", state, iter)
	}
	m.runs = []*phaseRun{{phaseIdx: 1, status: runActive, iteration: 1}}
	if state, iter := m.phaseState(1); state != "active" || iter != 1 {
		t.Fatalf("phase 1 should be active, got %s iter %d", state, iter)
	}
	m.runs = []*phaseRun{{phaseIdx: 1, status: runApproved, iteration: 2}}
	if state, iter := m.phaseState(1); state != "done" || iter != 0 {
		t.Fatalf("approved phase should be done, got %s iter %d", state, iter)
	}
	m.stageIdx = 3
	if state, iter := m.phaseState(4); state != "pending" || iter != 0 {
		t.Fatalf("phase 4 should be pending, got %s iter %d", state, iter)
	}
}

func TestResumeMarkerForRunSkipsFixState(t *testing.T) {
	m := newTestModel(t)
	m.state = stateFixRun
	if got := m.resumeMarkerForRun(&phaseRun{phaseIdx: 0, workDir: m.ProjectDir, branch: "refactor/ts/diagnostic"}); got != nil {
		t.Fatalf("expected no marker during fix state, got %#v", got)
	}
}

func TestNewModelKeepsOpenCodeModelBlankAndUsesOpenCodeMessage(t *testing.T) {
	m := NewModel(t.TempDir(), 2, true, "opencode", "  ", 2, "en", testPrompts(), nil)
	if m.AgentModel != "" {
		t.Fatalf("expected blank OpenCode model, got %q", m.AgentModel)
	}
	view := m.View()
	if !strings.Contains(view, "OpenCode default (configured in your OpenCode settings)") {
		t.Fatalf("expected OpenCode default message, got %q", view)
	}
	if !strings.Contains(view, "Cost and plan usage depend on the selected model.") {
		t.Fatalf("expected cost warning, got %q", view)
	}

	fr := NewModel(t.TempDir(), 2, true, "opencode", "  ", 2, "fr", testPrompts(), nil)
	frView := fr.View()
	if !strings.Contains(frView, "Modèle par défaut d'OpenCode") || !strings.Contains(frView, "configuré dans vos paramètres") {
		t.Fatalf("expected French OpenCode default message, got %q", frView)
	}
}

func TestOpenCodeModelSelectAllowsBlankAndShowsExplicitModel(t *testing.T) {
	installFakeOpenCode(t)
	blank := NewModel(t.TempDir(), 0, true, "opencode", "  ", 2, "en", testPrompts(), nil)
	if blank.AgentModel != "" {
		t.Fatalf("expected blank OpenCode model, got %q", blank.AgentModel)
	}
	updated, _ := blank.handleKey(key(tea.KeyEnter, 0))
	blank = updated.(Model)
	if blank.state != stateStage {
		t.Fatalf("blank OpenCode model should advance, got state %v", blank.state)
	}
	blank = drainActiveRuns(t, blank)
	cancelRuns(blank)

	explicit := NewModel(t.TempDir(), 0, true, "opencode", "custom/model", 2, "en", testPrompts(), nil)
	if explicit.AgentModel != "custom/model" {
		t.Fatalf("expected explicit OpenCode model, got %q", explicit.AgentModel)
	}
	view := explicit.View()
	if !strings.Contains(view, "custom/model") {
		t.Fatalf("expected explicit model in view, got %q", view)
	}
	if strings.Contains(view, "OpenCode default (configured in your OpenCode settings)") {
		t.Fatalf("default message should not show for explicit model, got %q", view)
	}
}

func installFakeOpenCode(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode")
	script := `#!/bin/sh
printf '%s\n' '{"type":"text","part":{"text":"opencode ready"}}'
printf '%s\n' '{"type":"result","result":"done"}'
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestUpdateMessageEdgeCases(t *testing.T) {
	m := newTestModel(t)

	// out-of-range slots are ignored
	updated, _ := m.Update(runLineMsg{slot: 5, line: "orphan"})
	m = updated.(Model)
	updated, _ = m.Update(runDoneMsg{slot: 5})
	m = updated.(Model)

	// failed run records the error
	m.runs = []*phaseRun{{phaseIdx: 0, status: runActive}}
	updated, _ = m.Update(runDoneMsg{slot: 0, err: errors.New("agent exploded")})
	m = updated.(Model)
	if m.runs[0].status != runFailed || m.runs[0].errMsg != "agent exploded" {
		t.Fatalf("failed run not recorded: %+v", m.runs[0])
	}

	// git setup failure surfaces in statusMsg without changing state
	m.state = stateGitSetup
	updated, _ = m.Update(gitSetupDoneMsg{err: errors.New("init blew up")})
	m = updated.(Model)
	if m.state != stateGitSetup || !strings.Contains(m.statusMsg, "init blew up") {
		t.Fatalf("git failure not surfaced: state=%v msg=%q", m.state, m.statusMsg)
	}

	// non-key messages fall through to the viewport
	updated, _ = m.Update(tea.MouseMsg{})
	_ = updated.(Model)
}

func TestModelSelectKeys(t *testing.T) {
	// esc quits
	m := newTestModel(t)
	_, cmd := m.handleKey(key(tea.KeyEsc, 0))
	if cmd == nil {
		t.Fatal("esc should quit")
	}

	// typing updates the input
	m = newTestModel(t)
	m.input.SetValue("")
	updated, _ := m.handleKey(key(tea.KeyRunes, 'x'))
	m = updated.(Model)
	if m.input.Value() != "x" {
		t.Fatalf("input not updated: %q", m.input.Value())
	}

	// enter without git repo and NoGit=false goes to git setup
	m = NewModel(t.TempDir(), 0, false, "claude", "sonnet", 2, "en", testPrompts(), nil)
	m.input.SetValue("sonnet")
	updated, _ = m.handleKey(key(tea.KeyEnter, 0))
	m = updated.(Model)
	if m.state != stateGitSetup {
		t.Fatalf("expected git setup, got %v", m.state)
	}
}

func TestGitSetupAndDoneKeys(t *testing.T) {
	m := newTestModel(t)
	m.state = stateGitSetup
	if _, cmd := m.handleKey(key(tea.KeyRunes, 'o')); cmd == nil {
		t.Fatal("git accept should return the init command")
	}
	if _, cmd := m.handleKey(key(tea.KeyRunes, 'q')); cmd == nil {
		t.Fatal("q should quit git setup")
	}

	m.state = stateChecklist
	if _, cmd := m.handleKey(key(tea.KeyRunes, 'q')); cmd == nil {
		t.Fatal("q should quit checklist")
	}

	m.state = stateDone
	// no failed checks: f is a no-op
	updated, cmd := m.handleKey(key(tea.KeyRunes, 'f'))
	m = updated.(Model)
	if m.state != stateDone || cmd != nil {
		t.Fatal("f without failures should do nothing")
	}
	if _, cmd := m.handleKey(key(tea.KeyEnter, 0)); cmd == nil {
		t.Fatal("enter should quit done state")
	}
}

func TestStageKeysEdgeCases(t *testing.T) {
	m := newTestModel(t)
	m.state = stateStage
	m.runs = []*phaseRun{{phaseIdx: 0, status: runActive}}

	// q quits even mid-run
	if _, cmd := m.handleKey(key(tea.KeyRunes, 'q')); cmd == nil {
		t.Fatal("q should quit the stage")
	}
	// tab with a single run is a no-op
	updated, _ := m.handleKey(key(tea.KeyTab, 0))
	m = updated.(Model)
	if m.activeRun != 0 {
		t.Fatal("tab with one run should not move")
	}
	// numeric selection out of range is ignored
	updated, _ = m.handleKey(key(tea.KeyRunes, '3'))
	m = updated.(Model)
	if m.activeRun != 0 {
		t.Fatal("out-of-range selection should not move")
	}
	// scroll fall-through on active run
	updated, _ = m.handleKey(key(tea.KeyDown, 0))
	m = updated.(Model)

	// failed run: q quits
	m.runs[0].status = runFailed
	if _, cmd := m.handleKey(key(tea.KeyRunes, 'q')); cmd == nil {
		t.Fatal("q should quit a failed run")
	}

	// decision: q quits
	m.runs[0].status = runDecision
	if _, cmd := m.handleKey(key(tea.KeyRunes, 'q')); cmd == nil {
		t.Fatal("q should quit at decision")
	}
}

func TestFeedbackTyping(t *testing.T) {
	m := newTestModel(t)
	m.state = stateFeedback
	m.nextState = stateStage
	m.input.SetValue("")
	m.input.Focus()
	updated, _ := m.handleKey(key(tea.KeyRunes, 'a'))
	m = updated.(Model)
	if m.input.Value() != "a" {
		t.Fatalf("feedback input not updated: %q", m.input.Value())
	}
	// enter with empty trimmed feedback stays in feedback state
	m.input.SetValue("   ")
	updated, _ = m.handleKey(key(tea.KeyEnter, 0))
	m = updated.(Model)
	if m.state != stateFeedback {
		t.Fatal("blank feedback should not submit")
	}
}

func TestRenderVariants(t *testing.T) {
	m := newTestModel(t)

	// pipeline with an active iter > 1 and a pending later phase
	m.state = stateStage
	m.stageIdx = 0
	m.runs = []*phaseRun{{phaseIdx: 0, iteration: 3, status: runActive}}
	if got := m.renderPipeline(); !strings.Contains(got, "(iter 3)") {
		t.Fatalf("iteration marker missing: %s", got)
	}
	// phase in the current stage without a matching run reports pending
	if st, _ := m.phaseState(1); st != "pending" {
		m.stageIdx = 1
		if st, _ = m.phaseState(1); st != "pending" {
			t.Fatalf("expected pending, got %s", st)
		}
	}

	// fix-run rendering: header, decision name, and error line
	m.state = stateFixRun
	m.fixRound = 2
	m.runs = []*phaseRun{{phaseIdx: 5, status: runDecision, errMsg: "leftover"}}
	m.activeRun = 0
	got := m.renderStage()
	if !strings.Contains(got, "leftover") {
		t.Fatalf("errMsg missing from stage: %s", got)
	}
	if got := m.renderDecision(m.runs[0]); got == "" {
		t.Fatal("fix decision should render")
	}

	// tabs including skipped status
	m.state = stateStage
	m.runs = []*phaseRun{
		{phaseIdx: 2, status: runSkipped},
		{phaseIdx: 3, status: runFailed},
	}
	if got := m.renderTabs(); !strings.Contains(got, "[2]") {
		t.Fatalf("tabs missing: %s", got)
	}

	// checklist current-item and untouched-item prefixes
	m.checkIdx = 1
	m.checkItems[0] = ChecklistItem{Label: "done", Done: true, Checked: true}
	if got := m.renderChecklist(); !strings.Contains(got, "←") {
		t.Fatalf("current item marker missing: %s", got)
	}

	// stage render with no runs
	m.runs = nil
	m.activeRun = 0
	if got := m.renderStage(); got != "" {
		t.Fatalf("empty stage should render nothing: %q", got)
	}
}

func TestStartStageGitBranching(t *testing.T) {
	installFakeClaude(t)
	dir := newGitRepo(t)

	m := NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), nil)
	m.useGit = true
	m.state = stateStage
	m.stageIdx = 0
	m, _ = m.startStage()
	if m.runs[0].status == runFailed {
		t.Fatalf("branch creation failed: %s", m.runs[0].errMsg)
	}
	m = drainActiveRuns(t, m)
	cancelRuns(m)

	// second start with the same timestamp: branch already exists
	m2 := NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), nil)
	m2.useGit = true
	m2.state = stateStage
	m2.stageIdx = 0
	m2.timestamp = m.timestamp
	runTestGit(t, dir, "checkout", "refactor/"+m.timestamp+"/diagnostic")
	m2, _ = m2.startStage()
	if m2.runs[0].status == runFailed {
		t.Fatalf("existing branch should be reused: %s", m2.runs[0].errMsg)
	}
	m2 = drainActiveRuns(t, m2)
	cancelRuns(m2)
}

func TestStartStageGitFailures(t *testing.T) {
	// non-repo with useGit forced: BranchExists errors → runFailed
	m := newTestModel(t)
	m.useGit = true
	m.state = stateStage
	m.stageIdx = 0
	m, _ = m.startStage()
	if m.runs[0].status != runFailed {
		t.Fatal("expected failed run on broken git sequential setup")
	}

	// parallel stage: WorktreeAdd fails for every phase
	m = newTestModel(t)
	m.Workflow.Stages = [][]int{{0}, {1}, {2, 3, 4}, {5}}
	m.useGit = true
	m.state = stateStage
	m.stageIdx = 2
	m, _ = m.startStage()
	for _, r := range m.runs {
		if r.status != runFailed {
			t.Fatal("expected failed run on broken git parallel setup")
		}
	}
}

func TestStartStageSkipsPhasesBeforeStartPhase(t *testing.T) {
	installFakeClaude(t)
	m := NewModel(t.TempDir(), 5, true, "claude", "sonnet", 2, "en", testPrompts(), nil)
	m.state = stateStage
	m.stageIdx = 0 // stages 0..2 contain phases < 5 only: all skipped
	m, _ = m.startStage()
	if m.stageIdx != 5 || len(m.runs) != 1 || m.runs[0].phaseIdx != 5 {
		t.Fatalf("expected jump to devil stage, got stage=%d runs=%d", m.stageIdx, len(m.runs))
	}
	m = drainActiveRuns(t, m)
	cancelRuns(m)
}

func TestApproveAndReiterateCommitErrors(t *testing.T) {
	// useGit with a non-repo workdir → HasChanges errors
	m := newTestModel(t)
	m.useGit = true
	m.state = stateStage
	m.runs = []*phaseRun{{phaseIdx: 0, status: runDecision, workDir: t.TempDir()}}
	m, _ = m.approveRun(0)
	if m.runs[0].errMsg == "" {
		t.Fatal("approve should surface the git error")
	}

	m.runs[0].errMsg = ""
	m, _ = m.reiterateRunWithFeedback(0, "again")
	if m.runs[0].errMsg == "" {
		t.Fatal("reiterate should surface the git error")
	}

	// fix-run approval with git error
	m.state = stateFixRun
	m.runs[0].errMsg = ""
	m, _ = m.approveRun(0)
	if m.runs[0].errMsg == "" || m.state != stateFixRun {
		t.Fatal("fix approval should surface the git error and stay put")
	}
}

func TestFixRunReiterationRestartsWithFeedback(t *testing.T) {
	installFakeClaude(t)
	m := newTestModel(t)
	m.checkItems[0].Done = true // one failed criterion
	m, _ = m.startFixRun()
	if m.state != stateFixRun || m.fixRound != 1 {
		t.Fatalf("fix run not started: state=%v round=%d", m.state, m.fixRound)
	}
	m = drainActiveRuns(t, m)
	m, _ = m.reiterateRunWithFeedback(0, "push harder")
	if m.fixRound != 2 {
		t.Fatalf("fix round not incremented: %d", m.fixRound)
	}
	m = drainActiveRuns(t, m)
	cancelRuns(m)
}

func TestCheckStageDoneGitFailures(t *testing.T) {
	// CurrentBranch fails on a non-repo project
	m := newTestModel(t)
	m.useGit = true
	m.runs = []*phaseRun{
		{phaseIdx: 2, status: runApproved, workDir: ""},
		{phaseIdx: 3, status: runApproved, workDir: ""},
	}
	m, _ = m.checkStageDone()
	if m.statusMsg == "" {
		t.Fatal("expected merge failure status")
	}

	// worktree removal fails for a skipped run pointing at a bogus dir
	dir := newGitRepo(t)
	m = NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), nil)
	m.useGit = true
	m.stageIdx = 2
	m.runs = []*phaseRun{
		{phaseIdx: 2, status: runSkipped, workDir: filepath.Join(t.TempDir(), "ghost")},
		{phaseIdx: 3, status: runSkipped, workDir: dir},
	}
	m, _ = m.checkStageDone()
	if m.statusMsg == "" {
		t.Fatal("expected worktree removal failure status")
	}
}

func TestCheckStageDoneRebaseConflict(t *testing.T) {
	dir := newGitRepo(t)
	m := NewModel(dir, 0, false, "claude", "sonnet", 2, "en", testPrompts(), nil)
	m.Workflow.Stages = [][]int{{0}, {1}, {2, 3, 4}, {5}}
	m.useGit = true
	m.stageIdx = 2

	// two approved worktrees editing the same file → second rebase conflicts
	for i, phaseIdx := range []int{2, 3} {
		worktree := filepath.Join(t.TempDir(), phases.All[phaseIdx].ID)
		branch := "conflict/" + phases.All[phaseIdx].ID
		if err := gitops.WorktreeAdd(dir, worktree, branch); err != nil {
			t.Fatal(err)
		}
		content := []byte(phases.All[phaseIdx].ID + " version\n")
		if err := os.WriteFile(filepath.Join(worktree, "base.txt"), content, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := gitops.CommitAll(worktree, "edit base "+phases.All[phaseIdx].ID); err != nil {
			t.Fatal(err)
		}
		m.runs = append(m.runs, &phaseRun{phaseIdx: phaseIdx, workDir: worktree, branch: branch, status: runApproved})
		_ = i
	}

	m, _ = m.checkStageDone()
	if m.statusMsg == "" {
		t.Fatal("expected rebase conflict to be reported")
	}
	if m.stageIdx != 2 {
		t.Fatal("stage should not advance on merge failure")
	}
}

func TestDoGitInitSupportsEmptyGreenfieldProject(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "YVCDB Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "YVCDB Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.invalid")
	// Empty greenfield projects need an initial commit before phase branches.
	m := NewModel(t.TempDir(), 0, false, "claude", "sonnet", 2, "en", testPrompts(), nil)
	msg := m.doGitInit()()
	done, ok := msg.(gitSetupDoneMsg)
	if !ok || done.err != nil || !done.useGit {
		t.Fatalf("expected successful empty repository initialization, got %#v", msg)
	}
}
