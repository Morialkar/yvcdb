package phases

import "github.com/charmbracelet/lipgloss"

type Phase struct {
	ID         string
	Label      string
	Title      string
	Color      lipgloss.Color
	PromptFile string
}

var All = []Phase{
	{
		ID:         "diagnostic",
		Label:      "Phase 0",
		Title:      "Diagnostic — lecture de reconnaissance",
		Color:      lipgloss.Color("12"),
		PromptFile: "diagnostic.md",
	},
	{
		ID:         "safety",
		Label:      "Phase 1",
		Title:      "Filet de sécurité — tests smoke + snapshot git",
		Color:      lipgloss.Color("10"),
		PromptFile: "safety.md",
	},
	{
		ID:         "security",
		Label:      "Phase 2a",
		Title:      "Sécurité — secrets, validation, auth",
		Color:      lipgloss.Color("9"),
		PromptFile: "security.md",
	},
	{
		ID:         "structure",
		Label:      "Phase 2b",
		Title:      "Structure — logique hors UI + déduplication",
		Color:      lipgloss.Color("11"),
		PromptFile: "structure.md",
	},
	{
		ID:         "readability",
		Label:      "Phase 2c",
		Title:      "Lisibilité — nommage, découpe, documentation",
		Color:      lipgloss.Color("13"),
		PromptFile: "readability.md",
	},
	{
		ID:         "devil",
		Label:      "Phase 3",
		Title:      "Avocat du diable — checklist finale",
		Color:      lipgloss.Color("14"),
		PromptFile: "devil.md",
	},
}

func IndexOf(id string) int {
	for i, p := range All {
		if p.ID == id {
			return i
		}
	}
	return -1
}
