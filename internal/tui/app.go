package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morialkar/yvcdb/internal/git"
	"github.com/morialkar/yvcdb/internal/i18n"
	"github.com/morialkar/yvcdb/internal/phases"
	"github.com/morialkar/yvcdb/internal/runner"
)

type appState int

const (
	stateModelSelect appState = iota
	stateGitSetup
	stateStage    // one or more runs in progress / awaiting decision
	stateFeedback // free-form refinement for the selected run
	stateChecklist
	stateFixRun // interactive fix loop after checklist
	stateDone
)

type runStatus int

const (
	runActive runStatus = iota
	runDecision
	runApproved
	runSkipped
)

// stages: phases executed sequentially; phases within a stage run in parallel.
var stages = [][]int{
	{0},       // diagnostic
	{1},       // safety
	{2, 3, 4}, // security + structure + readability en parallèle
	{5},       // devil
}

type startMsg struct{}
type runLineMsg struct {
	slot int
	line string
}
type runDoneMsg struct {
	slot int
	err  error
}
type gitSetupDoneMsg struct{ useGit bool }

type phaseRun struct {
	phaseIdx  int
	iteration int
	lines     []string
	status    runStatus
	workDir   string // worktree dir (parallel) or project dir
	branch    string
	lineCh    chan string
	doneCh    chan error
	errMsg    string
	feedback  string
}

type ChecklistItem struct {
	Label   string
	Checked bool
	Done    bool
}

type Model struct {
	ProjectDir string
	StartPhase int
	NoGit      bool
	Provider   string
	AgentModel string
	MaxTurns   int
	Language   string
	Prompts    map[string]string
	l10n       i18n.Localizer

	state     appState
	stageIdx  int
	runs      []*phaseRun
	activeRun int
	timestamp string
	logDir    string
	useGit    bool
	termW     int
	termH     int

	viewport  viewport.Model
	input     textinput.Model
	nextState appState

	checkItems []ChecklistItem
	checkIdx   int
	checkScore int
	fixRound   int

	statusMsg string
}

func NewModel(projectDir string, startPhase int, noGit bool, provider, model string, maxTurns int, language string, prompts map[string]string) Model {
	ts := time.Now().Format("20060102_150405")
	vp := viewport.New(120, 20)
	l10n := i18n.New(language)

	checklistLabels := []string{
		l10n.Pick("The code is understandable without external context", "Le code est compréhensible sans contexte externe"),
		l10n.Pick("No unresolved UNCLEAR: / SECURITY: / DUPLICATE: markers", "Aucun UNCLEAR: / SECURITY: / DUPLICATE: non résolu"),
		l10n.Pick("Tests cover the happy path, edge cases, and errors", "Tests : cas nominal + edge case + erreur couverts"),
		l10n.Pick("No business logic in the UI", "Zéro logique métier dans le UI"),
		l10n.Pick("All catches are explicit (no empty catches)", "Tous les catch sont explicites (pas de catch vide)"),
		l10n.Pick("All external inputs are validated", "Tous les inputs externes sont validés"),
		l10n.Pick("No hardcoded secrets in source code", "Aucun secret hardcodé dans le code source"),
		l10n.Pick("REFACTOR_BACKLOG is documented and prioritized", "REFACTOR_BACKLOG documenté et priorisé"),
	}
	items := make([]ChecklistItem, len(checklistLabels))
	for i, l := range checklistLabels {
		items[i] = ChecklistItem{Label: l}
	}

	useGit := !noGit && git.IsRepo(projectDir)
	state := stateModelSelect
	if strings.TrimSpace(model) == "" {
		model = "sonnet"
	}
	input := textinput.New()
	input.Prompt = l10n.T("model.prompt")
	input.Placeholder = "sonnet"
	input.SetValue(model)
	input.CharLimit = 100
	input.Width = 48
	input.Focus()

	// find the stage containing startPhase
	stageIdx := 0
	for si, stage := range stages {
		for _, pi := range stage {
			if pi == startPhase {
				stageIdx = si
			}
		}
	}

	return Model{
		ProjectDir: projectDir,
		StartPhase: startPhase,
		NoGit:      noGit,
		Provider:   provider,
		AgentModel: model,
		MaxTurns:   maxTurns,
		Language:   l10n.Language,
		Prompts:    prompts,
		l10n:       l10n,
		timestamp:  ts,
		logDir:     filepath.Join(projectDir, "refactor-logs"),
		stageIdx:   stageIdx,
		viewport:   vp,
		input:      input,
		checkItems: items,
		termW:      120,
		termH:      40,
		state:      state,
		useGit:     useGit,
	}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return startMsg{} }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case startMsg:
		return m, nil

	case tea.WindowSizeMsg:
		m.termW = msg.Width
		m.termH = msg.Height
		m.viewport.Width = msg.Width - 6
		m.viewport.Height = m.termH - 20
		if m.viewport.Height < 5 {
			m.viewport.Height = 5
		}
		m.refreshViewport()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case runLineMsg:
		if msg.slot < len(m.runs) {
			r := m.runs[msg.slot]
			r.lines = append(r.lines, msg.line)
			if msg.slot == m.activeRun {
				m.refreshViewport()
			}
			return m, waitForRun(msg.slot, r.lineCh, r.doneCh)
		}
		return m, nil

	case runDoneMsg:
		if msg.slot < len(m.runs) {
			r := m.runs[msg.slot]
			r.status = runDecision
			if msg.err != nil {
				r.errMsg = msg.err.Error()
			}
		}
		return m, nil

	case gitSetupDoneMsg:
		m.useGit = msg.useGit
		m.state = stateStage
		return m.startStage()
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) refreshViewport() {
	if m.activeRun >= len(m.runs) {
		m.viewport.SetContent("")
		return
	}
	content := strings.Join(m.runs[m.activeRun].lines, "\n")
	// wrap long lines to viewport width
	wrapped := lipgloss.NewStyle().Width(m.viewport.Width).Render(content)
	m.viewport.SetContent(wrapped)
	m.viewport.GotoBottom()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.state {
	case stateModelSelect:
		switch key {
		case "enter":
			if model := strings.TrimSpace(m.input.Value()); model != "" {
				m.AgentModel = model
				m.input.Blur()
				if !m.NoGit && !m.useGit {
					m.state = stateGitSetup
					return m, nil
				}
				m.state = stateStage
				return m.startStage()
			}
		case "esc":
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case stateFeedback:
		switch key {
		case "enter":
			feedback := strings.TrimSpace(m.input.Value())
			if feedback == "" {
				return m, nil
			}
			m.state = m.nextState
			m.input.Blur()
			return m.reiterateRunWithFeedback(m.activeRun, feedback)
		case "esc":
			m.state = m.nextState
			m.input.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case stateGitSetup:
		switch key {
		case "o", "O", "y", "Y":
			return m, m.doGitInit()
		case "n", "N":
			return m, func() tea.Msg { return gitSetupDoneMsg{useGit: false} }
		case "q":
			return m, tea.Quit
		}

	case stateStage, stateFixRun:
		// tab switching between parallel runs
		switch key {
		case "tab":
			if len(m.runs) > 1 {
				m.activeRun = (m.activeRun + 1) % len(m.runs)
				m.refreshViewport()
			}
			return m, nil
		case "1", "2", "3":
			idx := int(key[0] - '1')
			if idx < len(m.runs) {
				m.activeRun = idx
				m.refreshViewport()
			}
			return m, nil
		}

		// decision keys apply to the currently viewed run
		if m.activeRun < len(m.runs) && m.runs[m.activeRun].status == runDecision {
			switch key {
			case "o", "O", "y", "Y":
				return m.approveRun(m.activeRun)
			case "r", "R":
				return m.reiterateRun(m.activeRun)
			case "f", "F":
				m.nextState = m.state
				m.state = stateFeedback
				m.input.SetValue("")
				m.input.Placeholder = m.l10n.T("feedback.placeholder")
				return m, m.input.Focus()
			case "s", "S":
				return m.skipRun(m.activeRun)
			case "q", "Q":
				return m, tea.Quit
			}
		}

		// scroll
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case stateChecklist:
		switch key {
		case "o", "O", "y", "Y":
			m.checkItems[m.checkIdx].Checked = true
			m.checkItems[m.checkIdx].Done = true
			m.checkScore++
			return m.nextCheck()
		case "n", "N":
			m.checkItems[m.checkIdx].Done = true
			return m.nextCheck()
		case "q":
			return m, tea.Quit
		}

	case stateDone:
		switch key {
		case "f", "F":
			if m.failedChecks() > 0 {
				return m.startFixRun()
			}
		case "q", "enter":
			return m, tea.Quit
		}
	}

	return m, nil
}

// ─── Views ───────────────────────────────────────────────────────────────────

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderPipeline())
	b.WriteString("\n")

	switch m.state {
	case stateModelSelect:
		b.WriteString(m.renderModelSelect())
	case stateGitSetup:
		b.WriteString(m.renderGitSetup())
	case stateStage, stateFixRun:
		b.WriteString(m.renderStage())
	case stateFeedback:
		b.WriteString(m.renderStage())
		b.WriteString("\n" + m.renderFeedback())
	case stateChecklist:
		b.WriteString(m.renderChecklist())
	case stateDone:
		b.WriteString(m.renderDone())
	}

	if m.statusMsg != "" {
		b.WriteString("\n" + m.statusMsg)
	}

	return b.String()
}

func (m Model) renderHeader() string {
	title := stylePurple.Render(m.l10n.T("app.title"))
	sub := styleDim.Render(m.l10n.T("app.subtitle"))
	box := styleBox.Width(52).Render(title + "\n" + sub)
	return lipgloss.PlaceHorizontal(m.termW, lipgloss.Center, box)
}

func (m Model) phaseTitle(id string) string {
	titles := map[string][2]string{
		"diagnostic":  {"Diagnostic — codebase inventory and risks", "Diagnostic — inventaire et risques du code"},
		"safety":      {"Safety net — smoke tests and git snapshot", "Filet de sécurité — tests smoke + snapshot git"},
		"security":    {"Security — secrets, validation, authorization", "Sécurité — secrets, validation, auth"},
		"structure":   {"Structure — business logic out of UI and deduplication", "Structure — logique hors UI + déduplication"},
		"readability": {"Readability — naming, decomposition, documentation", "Lisibilité — nommage, découpe, documentation"},
		"devil":       {"Devil's advocate — final adversarial review", "Avocat du diable — revue finale sans ménagement"},
	}
	title, ok := titles[id]
	if !ok {
		return id
	}
	return m.l10n.Pick(title[0], title[1])
}

func (m Model) phaseState(phaseIdx int) (icon string, active bool, iter int) {
	// completed stage?
	for si, stage := range stages {
		for _, pi := range stage {
			if pi != phaseIdx {
				continue
			}
			if si < m.stageIdx {
				return "done", false, 0
			}
			if si > m.stageIdx {
				return "pending", false, 0
			}
			// current stage — look up run
			for _, r := range m.runs {
				if r.phaseIdx == phaseIdx {
					switch r.status {
					case runApproved, runSkipped:
						return "done", false, 0
					default:
						return "active", true, r.iteration
					}
				}
			}
			return "pending", false, 0
		}
	}
	return "pending", false, 0
}

func (m Model) renderPipeline() string {
	var lines []string
	lines = append(lines, styleDim.Render(m.l10n.T("pipeline")))

	for i, p := range phases.All {
		c := lipgloss.NewStyle().Foreground(p.Color)
		label := fmt.Sprintf("%s — %s", p.Label, m.phaseTitle(p.ID))

		st, _, iter := m.phaseState(i)
		var icon, line string
		switch st {
		case "done":
			icon = styleSuccess.Render("✓")
			line = styleDim.Render(label)
		case "active":
			icon = c.Bold(true).Render("▶")
			extra := ""
			if iter > 1 {
				extra = styleDim.Render(fmt.Sprintf(" (iter %d)", iter))
			}
			line = c.Bold(true).Render(label) + extra
		default:
			icon = styleDim.Render("○")
			line = styleDim.Render(label)
		}
		lines = append(lines, fmt.Sprintf("  %s %s", icon, line))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m Model) renderGitSetup() string {
	warn := styleWarn.Render(m.l10n.T("git.missing", m.ProjectDir))
	q := styleCyan.Render("?") + " " + styleBold.Render(m.l10n.T("git.init_question"))
	opts := "  " + styleSuccess.Render("[y/o]") + " " + m.l10n.T("git.yes") + "   " +
		styleError.Render("[n]") + " " + m.l10n.T("git.no")
	return warn + "\n" + q + "\n" + opts + "\n"
}

func (m Model) renderModelSelect() string {
	title := styleBold.Render(m.l10n.T("model.title", strings.ToUpper(m.Provider)))
	help := styleDim.Render(m.l10n.T("model.help"))
	warn := styleWarn.Render(m.l10n.T("model.warning"))
	return styleDecisionBox.Width(64).Render(title + "\n" + help + "\n\n" + m.input.View() + "\n\n" + warn + "\n" + styleDim.Render(m.l10n.T("confirm.quit")))
}

func (m Model) renderFeedback() string {
	q := styleBold.Render(m.l10n.T("feedback.title"))
	help := styleDim.Render(m.l10n.T("feedback.help"))
	return styleDecisionBox.Width(m.termW - 8).Render(q + "\n" + help + "\n\n" + m.input.View() + "\n" + styleDim.Render(m.l10n.T("feedback.send")))
}

func (m Model) renderTabs() string {
	if len(m.runs) <= 1 {
		return ""
	}
	var tabs []string
	for i, r := range m.runs {
		p := phases.All[r.phaseIdx]
		var status string
		switch r.status {
		case runActive:
			status = "⏳"
		case runDecision:
			status = styleWarn.Render("?")
		case runApproved:
			status = styleSuccess.Render("✓")
		case runSkipped:
			status = styleDim.Render("s")
		}
		label := fmt.Sprintf("[%d] %s %s", i+1, p.Label, status)
		if i == m.activeRun {
			tabs = append(tabs, lipgloss.NewStyle().Foreground(p.Color).Bold(true).Underline(true).Render(label))
		} else {
			tabs = append(tabs, styleDim.Render(label))
		}
	}
	return strings.Join(tabs, "   ") + "   " + styleDim.Render(m.l10n.T("tabs.help")) + "\n"
}

func (m Model) renderStage() string {
	if m.activeRun >= len(m.runs) {
		return ""
	}
	r := m.runs[m.activeRun]

	var header, info string
	if m.state == stateFixRun {
		c := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
		header = c.Render(m.l10n.T("fix.round", m.fixRound))
		info = styleDim.Render("  " + m.ProjectDir)
	} else {
		p := phases.All[r.phaseIdx]
		c := lipgloss.NewStyle().Foreground(p.Color).Bold(true)
		header = c.Render(fmt.Sprintf("▶ %s — %s", p.Label, m.phaseTitle(p.ID)))
		info = styleDim.Render("  " + m.l10n.T("iteration", r.iteration, r.workDir))
	}

	borderColor := lipgloss.Color("8")
	if m.state != stateFixRun && r.phaseIdx < len(phases.All) {
		borderColor = phases.All[r.phaseIdx].Color
	}
	vpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(m.termW - 4)

	out := m.renderTabs() + header + "\n" + info + "\n" + vpStyle.Render(m.viewport.View()) + "\n"

	if r.status == runDecision {
		out += m.renderDecision(r)
	}
	if r.errMsg != "" {
		out += "\n" + styleError.Render("⚠ "+r.errMsg)
	}
	return out
}

func (m Model) renderDecision(r *phaseRun) string {
	var name string
	if m.state == stateFixRun {
		name = m.l10n.T("fix.name")
	} else {
		name = phases.All[r.phaseIdx].Label
	}
	q := styleCyan.Render("?") + " " + styleBold.Render(m.l10n.T("decision.question", name))
	opts := strings.Join([]string{
		"  " + styleSuccess.Render("[y/o]") + " " + m.l10n.T("decision.approve"),
		"  " + styleWarn.Render("[r]") + " " + m.l10n.T("decision.retry"),
		"  " + styleCyan.Render("[f]") + " " + m.l10n.T("decision.refine"),
		"  " + styleCyan.Render("[s]") + " " + m.l10n.T("decision.skip"),
		"  " + styleError.Render("[q]") + " " + m.l10n.T("decision.quit"),
	}, "   ")
	return styleDecisionBox.Width(m.termW - 8).Render(q + "\n" + opts)
}

func (m Model) renderChecklist() string {
	var b strings.Builder
	b.WriteString(styleBold.Render(m.l10n.T("checklist.title")) + "\n\n")

	for i, item := range m.checkItems {
		var prefix string
		switch {
		case item.Done && item.Checked:
			prefix = styleSuccess.Render("[✓]")
		case item.Done && !item.Checked:
			prefix = styleError.Render("[✗]")
		case i == m.checkIdx:
			prefix = styleCyan.Render("[ ]") + " ← " + styleCyan.Render(m.l10n.T("check.yesno"))
		default:
			prefix = styleDim.Render("[ ]")
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", prefix, item.Label))
	}
	return b.String()
}

func (m Model) failedChecks() int {
	n := 0
	for _, item := range m.checkItems {
		if item.Done && !item.Checked {
			n++
		}
	}
	return n
}

func (m Model) renderDone() string {
	total := len(m.checkItems)
	failed := m.failedChecks()

	var b strings.Builder
	if failed == 0 {
		b.WriteString(styleSuccess.Render(m.l10n.T("done.ready", m.checkScore, total)))
	} else {
		b.WriteString(styleWarn.Render(m.l10n.T("done.score", m.checkScore, total)))
		b.WriteString("\n\n" + m.l10n.T("done.failed") + "\n")
		for _, item := range m.checkItems {
			if item.Done && !item.Checked {
				b.WriteString("  " + styleError.Render("✗") + " " + item.Label + "\n")
			}
		}
		b.WriteString("\n" + styleCyan.Render("[f]") + " " + m.l10n.T("done.fix") + "\n")
	}
	b.WriteString("\n" + styleDim.Render(m.l10n.T("done.logs", m.logDir, m.timestamp)))
	b.WriteString("\n" + styleDim.Render(m.l10n.T("done.quit")))
	return b.String()
}

// ─── Stage orchestration ─────────────────────────────────────────────────────

func (m Model) startStage() (Model, tea.Cmd) {
	if m.stageIdx >= len(stages) {
		return m.enterChecklist()
	}

	stage := stages[m.stageIdx]
	m.runs = nil
	m.activeRun = 0

	parallel := len(stage) > 1
	var cmds []tea.Cmd

	for slot, phaseIdx := range stage {
		// respect --phase start point within the stage
		if phaseIdx < m.StartPhase {
			continue
		}
		p := phases.All[phaseIdx]
		r := &phaseRun{
			phaseIdx:  phaseIdx,
			iteration: 1,
			workDir:   m.ProjectDir,
			branch:    fmt.Sprintf("refactor/%s/%s", m.timestamp, p.ID),
		}

		if m.useGit {
			if parallel {
				wtDir := filepath.Join(os.TempDir(), "yvcdb", m.timestamp, p.ID)
				if err := git.WorktreeAdd(m.ProjectDir, wtDir, r.branch); err != nil {
					r.errMsg = err.Error()
				} else {
					r.workDir = wtDir
				}
			} else {
				if !git.BranchExists(m.ProjectDir, r.branch) {
					_ = git.CreateBranch(m.ProjectDir, r.branch)
				}
			}
		}

		m.runs = append(m.runs, r)
		cmds = append(cmds, m.launchRun(len(m.runs)-1))
		_ = slot
	}

	if len(m.runs) == 0 {
		m.stageIdx++
		return m.startStage()
	}

	m.state = stateStage
	m.refreshViewport()
	return m, tea.Batch(cmds...)
}

func (m *Model) launchRun(slot int) tea.Cmd {
	r := m.runs[slot]
	p := phases.All[r.phaseIdx]

	systemPrompt := m.Prompts[p.ID]
	if r.iteration > 1 {
		iterationPrompt := m.l10n.Pick(
			"\n\nIMPORTANT: This is iteration %d of this phase. The previous result was not satisfactory. Be more exhaustive and critical, and cover what was missed.",
			"\n\nIMPORTANT: C'est l'itération %d de cette phase. Le résultat précédent n'était pas satisfaisant. Sois plus exhaustif et critique. Couvre ce qui a été manqué.",
		)
		systemPrompt += fmt.Sprintf(iterationPrompt, r.iteration)
	}
	systemPrompt += m.l10n.Pick("\n\nAlways communicate your analysis and final result in English.", "\n\nCommunique toujours ton analyse et ton résultat final en français.")

	r.lineCh = make(chan string, 512)
	r.doneCh = make(chan error, 1)
	r.lines = nil
	r.status = runActive

	runner.RunPhase(r.workDir, m.logDir, m.timestamp, p.ID, r.iteration, systemPrompt, runner.Options{
		Provider: m.Provider, Model: m.AgentModel, MaxTurns: m.MaxTurns, Feedback: r.feedback, Language: m.Language,
	}, r.lineCh, r.doneCh)
	r.feedback = ""
	return waitForRun(slot, r.lineCh, r.doneCh)
}

func waitForRun(slot int, lineCh chan string, doneCh chan error) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-lineCh
		if ok {
			return runLineMsg{slot: slot, line: line}
		}
		return runDoneMsg{slot: slot, err: <-doneCh}
	}
}

func (m Model) approveRun(slot int) (Model, tea.Cmd) {
	r := m.runs[slot]
	if m.state == stateFixRun {
		if m.useGit && git.HasChanges(r.workDir) {
			_ = git.CommitAll(r.workDir, fmt.Sprintf("refactor(fix): interactive fix round %d — YVCDB", m.fixRound))
		}
		m.state = stateDone
		return m, nil
	}

	p := phases.All[r.phaseIdx]
	if m.useGit && git.HasChanges(r.workDir) {
		_ = git.CommitAll(r.workDir, fmt.Sprintf("refactor(%s): changes applied by YVCDB", p.Label))
	}
	r.status = runApproved
	return m.checkStageDone()
}

func (m Model) skipRun(slot int) (Model, tea.Cmd) {
	if m.state == stateFixRun {
		m.state = stateDone
		return m, nil
	}
	m.runs[slot].status = runSkipped
	return m.checkStageDone()
}

func (m Model) reiterateRun(slot int) (Model, tea.Cmd) {
	return m.reiterateRunWithFeedback(slot, "")
}

func (m Model) reiterateRunWithFeedback(slot int, feedback string) (Model, tea.Cmd) {
	r := m.runs[slot]
	if m.useGit && git.HasChanges(r.workDir) {
		label := "fix"
		if m.state != stateFixRun {
			label = phases.All[r.phaseIdx].Label
		}
		_ = git.CommitAll(r.workDir, fmt.Sprintf("refactor(%s): iter%d — YVCDB", label, r.iteration))
	}
	r.iteration++
	r.feedback = feedback
	if m.state == stateFixRun {
		m.fixRound++
		return m.restartFixRun(feedback)
	}
	cmd := m.launchRun(slot)
	m.refreshViewport()
	return m, cmd
}

func (m Model) checkStageDone() (Model, tea.Cmd) {
	for _, r := range m.runs {
		if r.status != runApproved && r.status != runSkipped {
			// switch view to a run still needing attention
			for i, other := range m.runs {
				if other.status == runDecision {
					m.activeRun = i
					m.refreshViewport()
					break
				}
			}
			return m, nil
		}
	}

	// stage complete — integrate parallel branches one by one. Each approved
	// branch is rebased on the updated base before a fast-forward merge.
	if len(m.runs) > 1 && m.useGit {
		var problems []string
		baseBranch, branchErr := git.CurrentBranch(m.ProjectDir)
		if branchErr != nil {
			problems = append(problems, branchErr.Error())
		}
		for _, r := range m.runs {
			if r.workDir != m.ProjectDir {
				if r.status == runApproved {
					if branchErr == nil {
						if err := git.Rebase(r.workDir, baseBranch); err != nil {
							problems = append(problems, err.Error())
							continue
						}
					}
					if err := git.WorktreeRemove(m.ProjectDir, r.workDir); err != nil {
						problems = append(problems, err.Error())
						continue
					}
					if err := git.MergeFastForward(m.ProjectDir, r.branch); err != nil {
						problems = append(problems, err.Error())
					}
				} else if err := git.WorktreeRemove(m.ProjectDir, r.workDir); err != nil {
					problems = append(problems, err.Error())
				}
			}
		}
		if len(problems) > 0 {
			m.statusMsg = styleError.Render(m.l10n.T("merge.failed", strings.Join(problems, "\n")))
		}
	}

	m.stageIdx++
	return m.startStage()
}

// ─── Checklist & fix loop ────────────────────────────────────────────────────

func (m Model) enterChecklist() (Model, tea.Cmd) {
	m.state = stateChecklist
	m.checkIdx = 0
	m.checkScore = 0
	for i := range m.checkItems {
		m.checkItems[i].Checked = false
		m.checkItems[i].Done = false
	}
	return m, nil
}

func (m Model) nextCheck() (Model, tea.Cmd) {
	m.checkIdx++
	if m.checkIdx >= len(m.checkItems) {
		m.state = stateDone
	}
	return m, nil
}

func (m Model) startFixRun() (Model, tea.Cmd) {
	return m.restartFixRun("")
}

func (m Model) restartFixRun(feedback string) (Model, tea.Cmd) {
	if m.fixRound == 0 {
		m.fixRound = 1
	}

	var failed []string
	for _, item := range m.checkItems {
		if item.Done && !item.Checked {
			failed = append(failed, "- "+item.Label)
		}
	}

	systemPrompt := m.l10n.Pick(
		"You are performing the final fixes after a refactoring. A human reviewer marked these quality criteria as NOT satisfied:\n\n"+strings.Join(failed, "\n")+"\n\nAnalyze the project, identify precisely why each criterion fails, and fix it. Be exhaustive and concrete: modify the code directly. Respond in English.",
		"Tu es en mode correction finale d'un refactoring. Les critères de qualité suivants ont été jugés NON satisfaits par une revue humaine :\n\n"+strings.Join(failed, "\n")+"\n\nAnalyse le projet, identifie précisément pourquoi chaque critère échoue, et corrige. Sois exhaustif et concret : modifie le code directement. Réponds en français.",
	)

	r := &phaseRun{
		phaseIdx:  len(phases.All) - 1, // devil, pour la couleur
		iteration: m.fixRound,
		workDir:   m.ProjectDir,
		lineCh:    make(chan string, 512),
		doneCh:    make(chan error, 1),
		status:    runActive,
		feedback:  feedback,
	}
	m.runs = []*phaseRun{r}
	m.activeRun = 0
	m.state = stateFixRun
	m.refreshViewport()

	runner.RunPhase(r.workDir, m.logDir, m.timestamp, "fix", m.fixRound, systemPrompt, runner.Options{
		Provider: m.Provider, Model: m.AgentModel, MaxTurns: m.MaxTurns, Feedback: feedback, Language: m.Language,
	}, r.lineCh, r.doneCh)
	return m, waitForRun(0, r.lineCh, r.doneCh)
}

func (m Model) doGitInit() tea.Cmd {
	return func() tea.Msg {
		if err := git.Init(m.ProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "git init: %v\n", err)
			return gitSetupDoneMsg{useGit: false}
		}
		return gitSetupDoneMsg{useGit: true}
	}
}
