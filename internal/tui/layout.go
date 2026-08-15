package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func clampWidth(s string, width int) []string {
	lines := strings.Split(s, "\n")

	if width <= 0 {
		return lines
	}

	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "")
	}

	return lines
}

func (c *ProgramContext) render(header, body, footer string) string {
	room := c.ScreenHeight - lipgloss.Height(header) - lipgloss.Height(footer)

	sections := make([]string, 0, 3)
	sections = append(sections, header)

	if room > 0 {
		sections = append(sections, lipgloss.NewStyle().
			MaxWidth(c.MainContentWidth).
			Height(room).
			MaxHeight(room).
			Render(body))
	}

	sections = append(sections, footer)

	return lipgloss.NewStyle().
		PaddingLeft(contentHorizontalPadding).
		MaxWidth(c.ScreenWidth).
		Height(c.ScreenHeight).
		MaxHeight(c.ScreenHeight).
		Render(strings.Join(sections, "\n"))
}

func (c *ProgramContext) rule() string {
	if c.MainContentWidth <= 0 {
		return ""
	}

	return c.Styles.Rule.Render(strings.Repeat("─", c.MainContentWidth))
}

func placeApart(width int, left, right string) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return ansi.Truncate(left, width, "")
	}

	return left + strings.Repeat(" ", gap) + right
}
