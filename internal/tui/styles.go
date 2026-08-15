package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/znaniye/shellhubctl/internal/tui/theme"
)

type Styles struct {
	Brand    lipgloss.Style
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Hint     lipgloss.Style
	Info     lipgloss.Style
	Success  lipgloss.Style
	Error    lipgloss.Style

	Rule         lipgloss.Style
	TabActive    lipgloss.Style
	TabInactive  lipgloss.Style
	TabSeparator lipgloss.Style
	FooterBar    lipgloss.Style

	Table table.Styles
}

func InitStyles(t theme.Theme) Styles {
	faint := lipgloss.NewStyle().Foreground(t.FaintText)

	return Styles{
		Brand: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.SecondaryText),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.SecondaryText),
		Subtitle: faint,
		Hint:     faint,
		Info:     faint,
		Success: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.SuccessText),
		Error: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.ErrorText),

		Rule: lipgloss.NewStyle().
			Foreground(t.FaintBorder),
		TabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.SecondaryText),
		TabInactive: faint,
		TabSeparator: lipgloss.NewStyle().
			Foreground(t.FaintBorder),
		FooterBar: lipgloss.NewStyle().
			Foreground(t.FaintText).
			Background(t.SurfaceBackground),

		Table: table.Styles{
			Header: lipgloss.NewStyle().
				Bold(true).
				Foreground(t.SecondaryText).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(t.FaintBorder).
				BorderBottom(true).
				Padding(0, 1),
			Cell: lipgloss.NewStyle().
				Foreground(t.PrimaryText).
				Padding(0, 1),
			Selected: lipgloss.NewStyle().
				Bold(true).
				Foreground(t.InvertedText).
				Background(t.SelectedBackground),
		},
	}
}
