// Package phases defines the managed workflows executed by YVCDB.
package phases

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	// ModeAuto selects a workflow from the contents of the project directory.
	ModeAuto = "auto"
	// ModeRefactor runs the AFTER workflow for an existing codebase.
	ModeRefactor = "refactor"
	// ModeGreenfield runs the complete AFTER workflow for a new project.
	ModeGreenfield = "greenfield"
	// ModeFeature runs the AFTER workflow for adding a feature to an existing codebase.
	ModeFeature = "feature"
	// ModeDebug runs the AFTER workflow for fixing a bug in an existing codebase.
	ModeDebug = "debug"
)

// Phase describes one managed phase and its embedded prompt.
type Phase struct {
	ID         string
	Label      string
	Color      lipgloss.Color
	PromptFile string
}

// Workflow defines the phases and human validation gates for one work mode.
// Phase indexes grouped in one stage may execute in parallel.
type Workflow struct {
	Mode   string
	Phases []Phase
	Stages [][]int
}

var refactorPhases = []Phase{
	{ID: "diagnostic", Label: "Phase 0", Color: lipgloss.Color("12"), PromptFile: "diagnostic.md"},
	{ID: "safety", Label: "Phase 1", Color: lipgloss.Color("10"), PromptFile: "safety.md"},
	{ID: "security", Label: "Phase 2a", Color: lipgloss.Color("9"), PromptFile: "security.md"},
	{ID: "structure", Label: "Phase 2b", Color: lipgloss.Color("11"), PromptFile: "structure.md"},
	{ID: "readability", Label: "Phase 2c", Color: lipgloss.Color("13"), PromptFile: "readability.md"},
	{ID: "devil", Label: "Phase 3", Color: lipgloss.Color("14"), PromptFile: "devil.md"},
}

var greenfieldPhases = []Phase{
	{ID: "specification", Label: "Phase 0", Color: lipgloss.Color("12"), PromptFile: "specification.md"},
	{ID: "architecture", Label: "Phase 1", Color: lipgloss.Color("10"), PromptFile: "architecture.md"},
	{ID: "planning", Label: "Phase 2", Color: lipgloss.Color("11"), PromptFile: "planning.md"},
	{ID: "foundation", Label: "Phase 3", Color: lipgloss.Color("13"), PromptFile: "foundation.md"},
	{ID: "implementation", Label: "Phase 4", Color: lipgloss.Color("9"), PromptFile: "implementation.md"},
	{ID: "verification", Label: "Phase 5", Color: lipgloss.Color("14"), PromptFile: "verification.md"},
	{ID: "devil", Label: "Phase 6", Color: lipgloss.Color("5"), PromptFile: "devil.md"},
}

var featurePhases = []Phase{
	{ID: "scoping", Label: "Phase 0", Color: lipgloss.Color("15"), PromptFile: "scoping.md"},
	{ID: "impact", Label: "Phase 1", Color: lipgloss.Color("2"), PromptFile: "impact.md"},
	{ID: "planning", Label: "Phase 2", Color: lipgloss.Color("3"), PromptFile: "planning.md"},
	{ID: "implementation", Label: "Phase 3", Color: lipgloss.Color("4"), PromptFile: "implementation.md"},
	{ID: "verification", Label: "Phase 4", Color: lipgloss.Color("6"), PromptFile: "verification.md"},
	{ID: "devil", Label: "Phase 5", Color: lipgloss.Color("1"), PromptFile: "devil.md"},
}

var debugPhases = []Phase{
	{ID: "report", Label: "Phase 0", Color: lipgloss.Color("7"), PromptFile: "report.md"},
	{ID: "reproduction", Label: "Phase 1", Color: lipgloss.Color("8"), PromptFile: "reproduction.md"},
	{ID: "diagnosis", Label: "Phase 2", Color: lipgloss.Color("9"), PromptFile: "diagnosis.md"},
	{ID: "fix", Label: "Phase 3", Color: lipgloss.Color("10"), PromptFile: "fix.md"},
	{ID: "verification", Label: "Phase 4", Color: lipgloss.Color("11"), PromptFile: "verification.md"},
	{ID: "devil", Label: "Phase 5", Color: lipgloss.Color("12"), PromptFile: "devil.md"},
}

// All is retained as the public refactor phase list for API compatibility.
var All = refactorPhases

// ForMode returns the managed workflow for mode.
func ForMode(mode string) (Workflow, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ModeRefactor:
		return sequentialWorkflow(ModeRefactor, refactorPhases), nil
	case ModeGreenfield:
		return sequentialWorkflow(ModeGreenfield, greenfieldPhases), nil
	case ModeFeature:
		return sequentialWorkflow(ModeFeature, featurePhases), nil
	case ModeDebug:
		return sequentialWorkflow(ModeDebug, debugPhases), nil
	default:
		return Workflow{}, fmt.Errorf("unsupported workflow mode %q", mode)
	}
}

func sequentialWorkflow(mode string, workflowPhases []Phase) Workflow {
	stages := make([][]int, len(workflowPhases))
	for i := range workflowPhases {
		stages[i] = []int{i}
	}
	return Workflow{Mode: mode, Phases: workflowPhases, Stages: stages}
}

// IndexOf returns the index of a refactor phase ID, or -1 when it is unknown.
func IndexOf(id string) int {
	workflow, _ := ForMode(ModeRefactor)
	return workflow.IndexOf(id)
}

// IndexOf returns the index of a phase ID in the workflow, or -1 when unknown.
func (w Workflow) IndexOf(id string) int {
	for i, phase := range w.Phases {
		if phase.ID == id {
			return i
		}
	}
	return -1
}

// PhaseIDs returns the workflow phase IDs in execution order.
func (w Workflow) PhaseIDs() []string {
	ids := make([]string, len(w.Phases))
	for i, phase := range w.Phases {
		ids[i] = phase.ID
	}
	return ids
}
