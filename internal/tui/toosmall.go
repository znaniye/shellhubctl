package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func minScreenWidth() int {
	return minMainContentWidth + contentHorizontalPadding*2
}

func minScreenHeight() int {
	return minMainContentHeight + footerHeight
}

func (c *ProgramContext) dimensionView(label string, got, want int) string {
	value := c.Styles.Success
	if got < want {
		value = c.Styles.Error
	}

	return c.Styles.Hint.Render(fmt.Sprintf("%-7s", label)) +
		value.Render(fmt.Sprintf("%3d", got)) +
		c.Styles.Hint.Render(fmt.Sprintf(" / %d needed", want))
}

func (c *ProgramContext) tooSmallView() string {
	content := strings.Join([]string{
		c.Styles.Error.Render("terminal too small"),
		"",
		c.dimensionView("width", c.ScreenWidth, minScreenWidth()),
		c.dimensionView("height", c.ScreenHeight, minScreenHeight()),
		"",
		c.Styles.Hint.Render("resize the window to continue"),
	}, "\n")

	return center(content, c.ScreenWidth, c.ScreenHeight)
}

func center(content string, width, height int) string {
	lines := clampWidth(content, width)

	if width > 0 {
		for i, line := range lines {
			lines[i] = lipgloss.PlaceHorizontal(width, lipgloss.Center, line)
		}
	}

	if height <= 0 {
		return strings.Join(lines, "\n")
	}

	if len(lines) > height {
		lines = lines[:height]
	}

	centered := make([]string, height)
	copy(centered[(height-len(lines))/2:], lines)

	return strings.Join(centered, "\n")
}
