package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTooSmallScreenReplacesTheView(t *testing.T) {
	m := layoutModel(t, screenDashboard)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: minScreenWidth() - 1, Height: 20})

	view := stripANSI(m.View())
	if !strings.Contains(view, "terminal too small") {
		t.Fatalf("view = %q, want the too small screen", view)
	}

	if strings.Contains(view, "dev1") {
		t.Error("the device table leaked into the too small screen")
	}

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: minScreenWidth(), Height: 20})

	if strings.Contains(stripANSI(m.View()), "terminal too small") {
		t.Error("the too small screen survived a terminal wide enough")
	}
}

func TestTooSmallScreenReportsBothDimensions(t *testing.T) {
	m := layoutModel(t, screenDashboard)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 24, Height: 40})

	view := stripANSI(m.View())
	for _, want := range []string{"width", "24", "height", "40"} {
		if !strings.Contains(view, want) {
			t.Errorf("view = %q, want it to mention %q", view, want)
		}
	}
}

func TestTooSmallScreenFillsTheScreen(t *testing.T) {
	m := layoutModel(t, screenDashboard)

	for _, size := range []tea.WindowSizeMsg{
		{Width: 24, Height: 6},
		{Width: 39, Height: 20},
		{Width: 10, Height: 3},
		{Width: 80, Height: 1},
	} {
		m, _ = updateModel(t, m, size)

		assertFits(t, m.View(), size.Width, size.Height)
	}
}
