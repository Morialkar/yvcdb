package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleBold    = lipgloss.NewStyle().Bold(true)
	styleDim     = lipgloss.NewStyle().Faint(true)
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleCyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	stylePurple  = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("13")).
			Padding(0, 1)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("13"))

	styleDecisionBox = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("14")).
				Padding(0, 2).
				MarginTop(1)

	styleCheckItem = lipgloss.NewStyle().Padding(0, 1)
)
