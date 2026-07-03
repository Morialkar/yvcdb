package phases

import "github.com/charmbracelet/lipgloss"

// Phase describes one refactoring phase and its embedded prompt.
type Phase struct {
	ID         string
	Label      string
	Color      lipgloss.Color
	PromptFile string
}

// All lists the refactoring phases in execution order.
var All = []Phase{
	{
		ID:         "diagnostic",
		Label:      "Phase 0",
		Color:      lipgloss.Color("12"),
		PromptFile: "diagnostic.md",
	},
	{
		ID:         "safety",
		Label:      "Phase 1",
		Color:      lipgloss.Color("10"),
		PromptFile: "safety.md",
	},
	{
		ID:         "security",
		Label:      "Phase 2a",
		Color:      lipgloss.Color("9"),
		PromptFile: "security.md",
	},
	{
		ID:         "structure",
		Label:      "Phase 2b",
		Color:      lipgloss.Color("11"),
		PromptFile: "structure.md",
	},
	{
		ID:         "readability",
		Label:      "Phase 2c",
		Color:      lipgloss.Color("13"),
		PromptFile: "readability.md",
	},
	{
		ID:         "devil",
		Label:      "Phase 3",
		Color:      lipgloss.Color("14"),
		PromptFile: "devil.md",
	},
}

// IndexOf returns the index of a phase ID, or -1 when it is unknown.
func IndexOf(id string) int {
	for i, p := range All {
		if p.ID == id {
			return i
		}
	}
	return -1
}
