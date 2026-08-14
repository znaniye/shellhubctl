package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func assertFits(t *testing.T, view string, width, height int) {
	t.Helper()

	lines := strings.Split(view, "\n")
	if len(lines) != height {
		t.Errorf("view has %d lines, want %d", len(lines), height)
	}

	for i, line := range lines {
		if got := utf8.RuneCountInString(stripANSI(line)); got > width {
			t.Errorf("line %d spans %d columns, want at most %d", i, got, width)
		}
	}
}

func TestConnectStatesFillTheScreen(t *testing.T) {
	sizes := []tea.WindowSizeMsg{
		{Width: 100, Height: 30},
		{Width: 80, Height: 24},
		{Width: 40, Height: 10},
		{Width: 24, Height: 6},
	}

	states := map[string]func(*devicesModel){
		"prompting": func(d *devicesModel) {
			d.connDevice = d.devices[0]
			d.beginPrompt()
		},
		"notice": func(d *devicesModel) {
			d.setConnErr("device dev2 is offline")
		},
	}

	for name, apply := range states {
		for _, size := range sizes {
			m := layoutModel(t, screenDevices)
			m, _ = updateModel(t, m, size)
			apply(&m.devices)

			assertFits(t, m.View(), size.Width, size.Height)
		}

		t.Logf("%s fits every size", name)
	}
}

func TestNoticeKeepsTheTableVisible(t *testing.T) {
	m := layoutModel(t, screenDevices)
	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 24})

	before := m.devices.tableHeight()

	m.devices.setConnErr("device dev2 is offline")

	view := stripANSI(m.View())
	if !strings.Contains(view, "dev1") {
		t.Error("notice hid the device table")
	}

	if !strings.Contains(view, "device dev2 is offline") {
		t.Error("notice is not rendered")
	}

	after := m.devices.tableHeight()
	if after >= before {
		t.Errorf("table height stayed %d, want less than %d to make room for the notice", after, before)
	}

	if got := lipgloss.Height(m.devices.table.View()); got != after {
		t.Errorf("table rendered %d lines, want %d", got, after)
	}
}
