package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	appconfig "github.com/Morialkar/yvcdb/internal/config"
	"github.com/Morialkar/yvcdb/internal/git"
	"github.com/Morialkar/yvcdb/internal/i18n"
	"github.com/Morialkar/yvcdb/internal/phases"
	"github.com/Morialkar/yvcdb/internal/runner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultViewportWidth   = 120
	defaultViewportHeight  = 20
	defaultTerminalHeight  = 40
	viewportWidthMargin    = 6
	viewportHeightMargin   = 20
	minimumViewportHeight  = 5
	runChannelCapacity     = 512
	modelInputCharLimit    = 100
	logDirectoryName       = "refactor-logs"
	temporaryDirectoryName = "yvcdb"
	sessionTimestampFormat = "20060102_150405"
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
	runFailed
)

type runLineMsg struct {
	slot int
	line string
}
type runDoneMsg struct {
	slot int
	err  error
}
type gitSetupDoneMsg struct {
	useGit bool
	err    error
}

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
	cancel    context.CancelFunc
}

// ChecklistItem tracks a human response to a final quality criterion.
type ChecklistItem struct {
	Label   string
	Checked bool
	Done    bool
}

// Model is the Bubble Tea application model.
type Model struct {
	ProjectDir      string
	StartPhase      int
	NoGit           bool
	Provider        string
	AgentModel      string
	ResumeCandidate *runner.ResumeMarker
	MaxTurns        int
	Language        string
	Prompts         map[string]string
	Workflow        phases.Workflow
	l10n            i18n.Localizer

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

// NewModel constructs the YVCDB application model.
func NewModel(projectDir string, startPhase int, noGit bool, provider, model string, maxTurns int, language string, prompts map[string]string, resumeCandidate *runner.ResumeMarker, workflows ...phases.Workflow) Model {
	ts := time.Now().Format(sessionTimestampFormat)
	vp := viewport.New(defaultViewportWidth, defaultViewportHeight)
	l10n := i18n.New(language)
	workflow, err := phases.ForMode(phases.ModeRefactor)
	if err != nil {
		panic(err)
	}
	if len(workflows) > 0 {
		workflow = workflows[0]
	}

	checklistLabels := []string{
		l10n.Pick("The code is understandable without external context", "Le code est compréhensible sans contexte externe"),
		l10n.Pick("No unresolved UNCLEAR: / REQUIRES_REVIEW: / ASSUMPTION: / DUPLICATE: markers", "Aucun marqueur UNCLEAR: / REQUIRES_REVIEW: / ASSUMPTION: / DUPLICATE: non résolu"),
		l10n.Pick("Tests cover the happy path, edge cases, and errors", "Tests : cas nominal + edge case + erreur couverts"),
		l10n.Pick("No business logic in the UI", "Zéro logique métier dans le UI"),
		l10n.Pick("All catches are explicit (no empty catches)", "Tous les catch sont explicites (pas de catch vide)"),
		l10n.Pick("All external inputs are validated", "Tous les inputs externes sont validés"),
		l10n.Pick("No hardcoded secrets in source code", "Aucun secret hardcodé dans le code source"),
		l10n.Pick("REFACTOR_BACKLOG is documented and prioritized", "REFACTOR_BACKLOG documenté et priorisé"),
	}
	if workflow.Mode == phases.ModeGreenfield {
		checklistLabels = []string{
			l10n.Pick("The approved specification and explicit constraints are satisfied", "La spécification approuvée et les contraintes explicites sont respectées"),
			l10n.Pick("Architecture, schemas, API signatures, and technology decisions match their approved documents", "L'architecture, les schémas, les signatures API et les choix technologiques correspondent aux documents approuvés"),
			l10n.Pick("Every planned task and acceptance criterion is complete", "Chaque tâche planifiée et chaque critère d'acceptation sont complétés"),
			l10n.Pick("Tests cover the nominal case, an edge case, and an error case for each logic unit", "Les tests couvrent le cas nominal, un cas limite et un cas d'erreur pour chaque unité logique"),
			l10n.Pick("The human reviewer can explain every generated line", "La personne responsable peut expliquer chaque ligne générée"),
			l10n.Pick("Every ASSUMPTION marker has been reviewed and resolved or accepted", "Chaque marqueur ASSUMPTION a été révisé et résolu ou accepté"),
			l10n.Pick("Every REQUIRES_REVIEW marker has received explicit human review", "Chaque marqueur REQUIRES_REVIEW a reçu une revue humaine explicite"),
			l10n.Pick("The full test suite, coverage target, and quality checks pass", "La suite de tests, la cible de couverture et les contrôles qualité passent"),
			l10n.Pick("The adversarial review has no unresolved blocker", "La revue contradictoire ne contient aucun blocage non résolu"),
		}
	} else if workflow.Mode == phases.ModeFeature {
		checklistLabels = []string{
			l10n.Pick("The spec delta (goals, non-goals, acceptance criteria) and inherited plus new constraints are satisfied", "Le delta de spec (buts, non-buts, critères d'acceptation) et les contraintes héritées et nouvelles sont respectés"),
			l10n.Pick("AFTER_ARCHITECTURE.md reflects the real impact: touched modules, schema or migration changes, and API changes", "AFTER_ARCHITECTURE.md reflète l'impact réel : modules touchés, changements de schéma ou migrations, et changements d'API"),
			l10n.Pick("Every planned task and acceptance criterion is complete", "Chaque tâche planifiée et chaque critère d'acceptation sont complétés"),
			l10n.Pick("Tests cover the nominal case, an edge case, and an error case for each logic unit", "Les tests couvrent le cas nominal, un cas limite et un cas d'erreur pour chaque unité logique"),
			l10n.Pick("The full existing test suite passes with no regression", "La suite de tests existante passe au complet, sans régression"),
			l10n.Pick("The feature integrates with existing patterns instead of being grafted alongside them", "La feature s'intègre aux patterns existants au lieu d'être greffée à côté"),
			l10n.Pick("Every ASSUMPTION marker has been reviewed and resolved or accepted", "Chaque marqueur ASSUMPTION a été révisé et résolu ou accepté"),
			l10n.Pick("Every REQUIRES_REVIEW marker has received explicit human review", "Chaque marqueur REQUIRES_REVIEW a reçu une revue humaine explicite"),
			l10n.Pick("The adversarial review has no unresolved blocker", "La revue contradictoire ne contient aucun blocage non résolu"),
		}
	} else if workflow.Mode == phases.ModeDebug {
		checklistLabels = []string{
			l10n.Pick("The bug is described with reproduction steps, expected versus actual behavior, and scope", "Le bug est décrit avec étapes de reproduction, comportement attendu versus observé, et portée"),
			l10n.Pick("A test reproduces the bug and fails before the fix", "Un test reproduit le bug et échoue avant le correctif"),
			l10n.Pick("The documented root cause explains the failure rather than a symptom", "La cause racine documentée explique la défaillance plutôt qu'un symptôme"),
			l10n.Pick("The fix is minimal and targets the root cause", "Le correctif est minimal et vise la cause racine"),
			l10n.Pick("The reproduction test passes and regression tests cover the nominal, edge, and error cases", "Le test de reproduction passe et les tests de régression couvrent les cas nominal, limite et erreur"),
			l10n.Pick("The full existing test suite passes with no regression", "La suite de tests existante passe au complet, sans régression"),
			l10n.Pick("Every ASSUMPTION marker has been reviewed and resolved or accepted", "Chaque marqueur ASSUMPTION a été révisé et résolu ou accepté"),
			l10n.Pick("Every REQUIRES_REVIEW marker has received explicit human review", "Chaque marqueur REQUIRES_REVIEW a reçu une revue humaine explicite"),
			l10n.Pick("The adversarial review has no unresolved blocker", "La revue contradictoire ne contient aucun blocage non résolu"),
		}
	}
	items := make([]ChecklistItem, len(checklistLabels))
	for i, l := range checklistLabels {
		items[i] = ChecklistItem{Label: l}
	}

	useGit := !noGit && git.IsRepo(projectDir)
	state := stateModelSelect
	if strings.TrimSpace(model) == "" {
		model = appconfig.SuggestedModel(provider)
	}
	input := textinput.New()
	input.Prompt = l10n.T("model.prompt")
	input.Placeholder = appconfig.SuggestedModel(provider)
	if strings.TrimSpace(input.Placeholder) == "" {
		input.Placeholder = l10n.T("model.default")
	}
	input.SetValue(model)
	input.CharLimit = modelInputCharLimit
	input.Width = 48
	input.Focus()

	// find the stage containing startPhase
	stageIdx := 0
	for si, stage := range workflow.Stages {
		for _, pi := range stage {
			if pi == startPhase {
				stageIdx = si
			}
		}
	}

	return Model{
		ProjectDir:      projectDir,
		StartPhase:      startPhase,
		NoGit:           noGit,
		Provider:        provider,
		AgentModel:      model,
		ResumeCandidate: resumeCandidate,
		MaxTurns:        maxTurns,
		Language:        l10n.Language,
		Prompts:         prompts,
		Workflow:        workflow,
		l10n:            l10n,
		timestamp:       ts,
		logDir:          filepath.Join(projectDir, logDirectoryName),
		stageIdx:        stageIdx,
		viewport:        vp,
		input:           input,
		checkItems:      items,
		termW:           defaultViewportWidth,
		termH:           defaultTerminalHeight,
		state:           state,
		useGit:          useGit,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.termW = msg.Width
		m.termH = msg.Height
		m.viewport.Width = msg.Width - viewportWidthMargin
		m.viewport.Height = m.termH - viewportHeightMargin
		if m.viewport.Height < minimumViewportHeight {
			m.viewport.Height = minimumViewportHeight
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
			if msg.err != nil {
				r.status = runFailed
				r.errMsg = msg.err.Error()
			} else {
				r.status = runDecision
			}
			r.cancel = nil
		}
		return m, nil

	case gitSetupDoneMsg:
		if msg.err != nil {
			m.statusMsg = styleError.Render("⚠ " + msg.err.Error())
			return m, nil
		}
		m.statusMsg = ""
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
		return m.quit()
	}

	switch m.state {
	case stateModelSelect:
		switch key {
		case "enter":
			if model := strings.TrimSpace(m.input.Value()); model != "" || appconfig.SuggestedModel(m.Provider) == "" {
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
			return m.quit()
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
			return m.quit()
		}

	case stateStage, stateFixRun:
		if key == "q" || key == "Q" {
			return m.quit()
		}
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
		if m.activeRun < len(m.runs) && m.runs[m.activeRun].status == runFailed {
			switch key {
			case "s", "S":
				return m.skipRun(m.activeRun)
			case "q", "Q":
				return m.quit()
			}
		}

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
				return m.quit()
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
			return m.quit()
		}

	case stateDone:
		switch key {
		case "f", "F":
			if m.failedChecks() > 0 {
				return m.startFixRun()
			}
		case "q", "enter":
			return m.quit()
		}
	}

	return m, nil
}

func (m Model) quit() (tea.Model, tea.Cmd) {
	for _, run := range m.runs {
		if run.cancel != nil {
			run.cancel()
		}
	}
	return m, tea.Quit
}

// ─── Views ───────────────────────────────────────────────────────────────────

// View implements tea.Model.
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
		"diagnostic":     {"Diagnostic — codebase inventory and risks", "Diagnostic — inventaire et risques du code"},
		"safety":         {"Safety net — smoke tests and git snapshot", "Filet de sécurité — tests smoke + snapshot git"},
		"security":       {"Security — secrets, validation, authorization", "Sécurité — secrets, validation, auth"},
		"structure":      {"Structure — business logic out of UI and deduplication", "Structure — logique hors UI + déduplication"},
		"readability":    {"Readability — naming, decomposition, documentation", "Lisibilité — nommage, découpe, documentation"},
		"devil":          {"Devil's advocate — final adversarial review", "Avocat du diable — revue finale sans ménagement"},
		"specification":  {"Specification — requirements and acceptance criteria", "Spécification — exigences et critères d'acceptation"},
		"architecture":   {"Architecture — decisions, constraints, schemas, and APIs", "Architecture — décisions, contraintes, schémas et API"},
		"planning":       {"Plan — self-contained implementation tasks", "Plan — tâches d'implémentation autonomes"},
		"foundation":     {"Foundation — scaffold, tooling, and test harness", "Fondations — structure, outillage et banc de tests"},
		"implementation": {"Implementation — production code and tests together", "Implémentation — code de production et tests ensemble"},
		"verification":   {"Verification — rigorous quality and security checks", "Vérification — contrôles rigoureux de qualité et sécurité"},
		"scoping":        {"Scoping — spec delta for the feature", "Cadrage — delta de spec de la feature"},
		"impact":         {"Impact analysis — architecture delta and risks", "Analyse d'impact — delta d'architecture et risques"},
		"report":         {"Report — required bug description and impact", "Rapport — description de bug requise et impact"},
		"reproduction":   {"Reproduction — failing test before the fix", "Reproduction — test en échec avant le correctif"},
		"diagnosis":      {"Diagnosis — root cause analysis", "Diagnostic — analyse de la cause racine"},
		"fix":            {"Fix — minimal correction targeting the root cause", "Correctif — correction minimale visant la cause racine"},
	}
	title, ok := titles[id]
	if !ok {
		return id
	}
	return m.l10n.Pick(title[0], title[1])
}

func (m Model) phaseState(phaseIdx int) (state string, iter int) {
	// completed stage?
	for si, stage := range m.Workflow.Stages {
		for _, pi := range stage {
			if pi != phaseIdx {
				continue
			}
			if si < m.stageIdx {
				return "done", 0
			}
			if si > m.stageIdx {
				return "pending", 0
			}
			// current stage — look up run
			for _, r := range m.runs {
				if r.phaseIdx == phaseIdx {
					switch r.status {
					case runApproved, runSkipped:
						return "done", 0
					default:
						return "active", r.iteration
					}
				}
			}
			return "pending", 0
		}
	}
	return "pending", 0
}

func (m Model) renderPipeline() string {
	var lines []string
	lines = append(lines, styleDim.Render(m.l10n.T("pipeline")))

	for i, p := range m.Workflow.Phases {
		c := lipgloss.NewStyle().Foreground(p.Color)
		label := fmt.Sprintf("%s — %s", p.Label, m.phaseTitle(p.ID))

		st, iter := m.phaseState(i)
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
	extra := ""
	if strings.TrimSpace(m.AgentModel) == "" && appconfig.SuggestedModel(m.Provider) == "" {
		extra = "\n" + styleDim.Render(m.l10n.T("model.default"))
	}
	warn := styleWarn.Render(m.l10n.T("model.warning"))
	return styleDecisionBox.Width(64).Render(title + "\n" + help + "\n\n" + m.input.View() + extra + "\n\n" + warn + "\n" + styleDim.Render(m.l10n.T("confirm.quit")))
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
		p := m.Workflow.Phases[r.phaseIdx]
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
		case runFailed:
			status = styleError.Render("!")
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
		p := m.Workflow.Phases[r.phaseIdx]
		c := lipgloss.NewStyle().Foreground(p.Color).Bold(true)
		header = c.Render(fmt.Sprintf("▶ %s — %s", p.Label, m.phaseTitle(p.ID)))
		info = styleDim.Render("  " + m.l10n.T("iteration", r.iteration, r.workDir))
	}

	borderColor := lipgloss.Color("8")
	if m.state != stateFixRun && r.phaseIdx < len(m.Workflow.Phases) {
		borderColor = m.Workflow.Phases[r.phaseIdx].Color
	}
	vpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(m.termW - 4)

	out := m.renderTabs() + header + "\n" + info + "\n" + vpStyle.Render(m.viewport.View()) + "\n"

	if r.status == runDecision || r.status == runFailed {
		out += m.renderDecision(r)
	}
	if r.errMsg != "" {
		out += "\n" + styleError.Render("⚠ "+r.errMsg)
	}
	return out
}

func (m Model) renderDecision(r *phaseRun) string {
	if r.status == runFailed {
		return styleDecisionBox.Width(m.termW - 8).Render(
			styleError.Render(m.l10n.T("run.failed")) + "\n  " + styleCyan.Render("[s]") + " " + m.l10n.T("decision.skip") + "   " + styleError.Render("[q]") + " " + m.l10n.T("decision.quit"),
		)
	}
	var name string
	if m.state == stateFixRun {
		name = m.l10n.T("fix.name")
	} else {
		name = m.Workflow.Phases[r.phaseIdx].Label
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
	if m.stageIdx >= len(m.Workflow.Stages) {
		return m.enterChecklist()
	}

	stage := m.Workflow.Stages[m.stageIdx]
	m.runs = nil
	m.activeRun = 0

	parallel := len(stage) > 1
	var cmds []tea.Cmd

	for _, phaseIdx := range stage {
		// respect --phase start point within the stage
		if phaseIdx < m.StartPhase {
			continue
		}
		p := m.Workflow.Phases[phaseIdx]
		r := &phaseRun{
			phaseIdx:  phaseIdx,
			iteration: 1,
			workDir:   m.ProjectDir,
			branch:    fmt.Sprintf("%s/%s/%s", m.Workflow.Mode, m.timestamp, p.ID),
		}

		if m.useGit {
			if parallel {
				wtDir := filepath.Join(os.TempDir(), temporaryDirectoryName, m.timestamp, p.ID)
				if err := git.WorktreeAdd(m.ProjectDir, wtDir, r.branch); err != nil {
					r.workDir = ""
					r.status = runFailed
					r.errMsg = err.Error()
				} else {
					r.workDir = wtDir
				}
			} else {
				branchExists, err := git.BranchExists(m.ProjectDir, r.branch)
				if err != nil {
					r.workDir = ""
					r.status = runFailed
					r.errMsg = err.Error()
				} else if !branchExists {
					if err := git.CreateBranch(m.ProjectDir, r.branch); err != nil {
						r.workDir = ""
						r.status = runFailed
						r.errMsg = err.Error()
					}
				}
			}
		}

		m.runs = append(m.runs, r)
		if r.status != runFailed {
			cmds = append(cmds, m.launchRun(len(m.runs)-1))
		}
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
	p := m.Workflow.Phases[r.phaseIdx]

	systemPrompt, err := m.phaseSystemPrompt(r, p)
	if err != nil {
		r.status = runFailed
		r.errMsg = err.Error()
		return nil
	}

	r.lineCh = make(chan string, runChannelCapacity)
	r.doneCh = make(chan error, 1)
	r.lines = nil
	r.status = runActive

	r.cancel = runner.RunPhase(r.workDir, m.logDir, m.timestamp, p.ID, r.iteration, systemPrompt, runner.Options{
		Provider: m.Provider, Model: m.AgentModel, MaxTurns: m.MaxTurns, Feedback: r.feedback, Language: m.Language,
	}, r.lineCh, r.doneCh)
	r.feedback = ""
	return waitForRun(slot, r.lineCh, r.doneCh)
}

func (m Model) phaseSystemPrompt(r *phaseRun, phase phases.Phase) (string, error) {
	systemPrompt := m.Prompts[phase.ID]
	systemPrompt += m.l10n.Pick(
		"\n\n# AFTER operating rules\nThe human makes every consequential decision. Preserve approved constraints and stop with DECISION_REQUIRED when one is missing. Mark every inference not grounded in project evidence as ASSUMPTION. Mark security-sensitive code involving authentication, payments, permissions, secrets, or personal data as REQUIRES_REVIEW. Generate tests with every behavior change, covering the nominal case, an edge case, and an error case. Treat generated output as unverified until commands prove it. Your response must be self-contained for a future session with no conversational memory.",
		"\n\n# Règles d'opération AFTER\nLa personne responsable prend chaque décision conséquente. Respecte les contraintes approuvées et arrête-toi avec DECISION_REQUIRED lorsqu'une décision manque. Marque toute inférence non fondée sur le projet par ASSUMPTION. Marque le code sensible touchant l'authentification, les paiements, les permissions, les secrets ou les données personnelles par REQUIRES_REVIEW. Génère les tests avec chaque changement de comportement, couvrant le cas nominal, un cas limite et un cas d'erreur. Toute sortie générée demeure non vérifiée jusqu'à preuve par commandes. Ta réponse doit être autonome pour une session future sans mémoire conversationnelle.",
	)
	if standards, err := os.ReadFile(filepath.Join(r.workDir, "AFTER_STANDARDS.md")); err == nil {
		systemPrompt += "\n\n# Project quality standards\n\n" + string(standards)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read AFTER_STANDARDS.md: %w", err)
	}
	if r.iteration > 1 {
		iterationPrompt := m.l10n.Pick(
			"\n\nIMPORTANT: This is iteration %d of this phase. The previous result was not satisfactory. Be more exhaustive and critical, and cover what was missed.",
			"\n\nIMPORTANT: C'est l'itération %d de cette phase. Le résultat précédent n'était pas satisfaisant. Sois plus exhaustif et critique. Couvre ce qui a été manqué.",
		)
		systemPrompt += fmt.Sprintf(iterationPrompt, r.iteration)
	}
	systemPrompt += m.l10n.Pick("\n\nAlways communicate your analysis and final result in English.", "\n\nCommunique toujours ton analyse et ton résultat final en français.")
	return systemPrompt, nil
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
		if err := m.commitChanges(r.workDir, fmt.Sprintf("refactor(fix): interactive fix round %d — YVCDB", m.fixRound)); err != nil {
			r.errMsg = err.Error()
			return m, nil
		}
		m.state = stateDone
		return m, nil
	}

	p := m.Workflow.Phases[r.phaseIdx]
	if err := m.commitChanges(r.workDir, fmt.Sprintf("%s(%s): changes applied by YVCDB", m.Workflow.Mode, p.Label)); err != nil {
		r.errMsg = err.Error()
		return m, nil
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
	label := "fix"
	if m.state != stateFixRun {
		label = m.Workflow.Phases[r.phaseIdx].Label
	}
	if err := m.commitChanges(r.workDir, fmt.Sprintf("%s(%s): iter%d — YVCDB", m.Workflow.Mode, label, r.iteration)); err != nil {
		r.errMsg = err.Error()
		return m, nil
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

func (m Model) commitChanges(dir, message string) error {
	if !m.useGit {
		return nil
	}
	hasChanges, err := git.HasChanges(dir)
	if err != nil {
		return err
	}
	if !hasChanges {
		return nil
	}
	return git.CommitAll(dir, message)
}

func (m Model) checkStageDone() (Model, tea.Cmd) {
	for _, r := range m.runs {
		if r.status != runApproved && r.status != runSkipped {
			// switch view to a run still needing attention
			for i, other := range m.runs {
				if other.status == runDecision || other.status == runFailed {
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
			m.statusMsg = styleError.Render(m.l10n.T("merge.failed", branchErr.Error()))
			return m, nil
		}
		for _, r := range m.runs {
			if r.workDir != "" && r.workDir != m.ProjectDir {
				if r.status == runApproved {
					if err := git.Rebase(r.workDir, baseBranch); err != nil {
						problems = append(problems, err.Error())
						continue
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
			return m, nil
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
		"You are performing final fixes after a managed AFTER workflow. A human reviewer marked these quality criteria as NOT satisfied:\n\n"+strings.Join(failed, "\n")+"\n\nAnalyze the project, identify precisely why each criterion fails, and fix it. Be exhaustive and concrete: modify the code directly. Respond in English.",
		"Tu effectues les corrections finales d'un workflow AFTER géré. Les critères de qualité suivants ont été jugés NON satisfaits par une revue humaine :\n\n"+strings.Join(failed, "\n")+"\n\nAnalyse le projet, identifie précisément pourquoi chaque critère échoue, et corrige. Sois exhaustif et concret : modifie le code directement. Réponds en français.",
	)
	if standards, err := os.ReadFile(filepath.Join(m.ProjectDir, "AFTER_STANDARDS.md")); err == nil {
		systemPrompt += "\n\n# Project quality standards\n\n" + string(standards)
	}

	r := &phaseRun{
		phaseIdx:  len(m.Workflow.Phases) - 1, // use the devil phase color
		iteration: m.fixRound,
		workDir:   m.ProjectDir,
		lineCh:    make(chan string, runChannelCapacity),
		doneCh:    make(chan error, 1),
		status:    runActive,
		feedback:  feedback,
	}
	m.runs = []*phaseRun{r}
	m.activeRun = 0
	m.state = stateFixRun
	m.refreshViewport()

	r.cancel = runner.RunPhase(r.workDir, m.logDir, m.timestamp, "fix", m.fixRound, systemPrompt, runner.Options{
		Provider: m.Provider, Model: m.AgentModel, MaxTurns: m.MaxTurns, Feedback: feedback, Language: m.Language,
	}, r.lineCh, r.doneCh)
	return m, waitForRun(0, r.lineCh, r.doneCh)
}

func (m Model) doGitInit() tea.Cmd {
	return func() tea.Msg {
		if err := git.Init(m.ProjectDir); err != nil {
			return gitSetupDoneMsg{useGit: false, err: fmt.Errorf("initialize git: %w", err)}
		}
		return gitSetupDoneMsg{useGit: true}
	}
}
