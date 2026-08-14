package tui

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/znaniye/shellhub-tui/internal/shellhub"
)

var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func layoutModel(t *testing.T, s screen) model {
	t.Helper()

	ctx := context.Background()

	m := model{
		ctx:    ctx,
		screen: s,
		width:  defaultWidth,
		height: defaultHeight,
		login:  newLoginModel(ctx, nil, nil),
	}

	m.namespaces = newNamespacesModel(ctx, nil, nil, "alice", m.width, m.height)
	m.namespaces.loading = false
	m.namespaces.setNamespaces([]shellhub.Namespace{
		{Name: "alpha", TenantID: "t-1", DevicesAcceptedCount: 2},
		{Name: "beta", TenantID: "t-2", DevicesAcceptedCount: 7},
	})

	m.devices = newDevicesModel(ctx, nil, nil, "alpha", m.width, m.height)
	m.devices.loading = false
	m.devices.setDevices([]shellhub.Device{
		{UID: "d-1", Name: "dev1", Status: "accepted", Online: true},
		{UID: "d-2", Name: "dev2", Status: "accepted"},
	}, 2)

	return m
}

func TestFrameAnchorsFooterAtTheBottom(t *testing.T) {
	got := frame("title\nbody", "\nhelp", 0, 6)

	want := "title\nbody\n\n\n\nhelp"
	if got != want {
		t.Fatalf("frame = %q, want %q", got, want)
	}
}

func TestFrameKeepsContentAtTheTop(t *testing.T) {
	lines := strings.Split(frame("title", "help", 0, 10), "\n")

	if len(lines) != 10 {
		t.Fatalf("frame produced %d lines, want 10", len(lines))
	}

	if lines[0] != "title" {
		t.Errorf("first line = %q, want title", lines[0])
	}

	if lines[9] != "help" {
		t.Errorf("last line = %q, want help", lines[9])
	}
}

func TestFrameTruncatesOverflowingContent(t *testing.T) {
	content := strings.Join([]string{"a", "b", "c", "d", "e"}, "\n")

	got := frame(content, "help", 0, 3)

	want := "a\nb\nhelp"
	if got != want {
		t.Fatalf("frame = %q, want %q", got, want)
	}
}

func TestFrameKeepsFooterWhenTallerThanScreen(t *testing.T) {
	got := frame("body", "one\ntwo\nthree", 0, 2)

	want := "two\nthree"
	if got != want {
		t.Fatalf("frame = %q, want %q", got, want)
	}
}

func TestFrameClampsEveryLineToTheWidth(t *testing.T) {
	got := frame("short\nmuch longer line", "help bar overflowing", 6, 3)

	want := "short\nmuch l\nhelp b"
	if got != want {
		t.Fatalf("frame = %q, want %q", got, want)
	}
}

func TestFrameKeepsStyledTextWithinTheWidth(t *testing.T) {
	styled := lipgloss.NewStyle().Bold(true).Render("hello world")

	for _, line := range strings.Split(frame(styled, "", 5, 2), "\n") {
		if got := lipgloss.Width(line); got > 5 {
			t.Errorf("line %q spans %d columns, want at most 5", stripANSI(line), got)
		}
	}
}

func TestViewFillsTheScreenAndAnchorsHelp(t *testing.T) {
	screens := []struct {
		name   string
		screen screen
		title  string
	}{
		{name: "login", screen: screenLogin, title: "shellhub-tui"},
		{name: "namespaces", screen: screenNamespaces, title: "namespaces"},
		{name: "devices", screen: screenDevices, title: "devices"},
	}

	sizes := []tea.WindowSizeMsg{
		{Width: 100, Height: 24},
		{Width: 120, Height: 50},
		{Width: 100, Height: 8},
	}

	for _, sc := range screens {
		t.Run(sc.name, func(t *testing.T) {
			m := layoutModel(t, sc.screen)

			for _, size := range sizes {
				m, _ = updateModel(t, m, size)

				lines := strings.Split(m.View(), "\n")
				if len(lines) != size.Height {
					t.Fatalf("height %d: view has %d lines, want %d", size.Height, len(lines), size.Height)
				}

				if first := stripANSI(lines[0]); !strings.Contains(first, sc.title) {
					t.Errorf("height %d: first line = %q, want the %q title", size.Height, first, sc.title)
				}

				if last := stripANSI(lines[len(lines)-1]); !strings.Contains(last, "quit") {
					t.Errorf("height %d: last line = %q, want the help bar", size.Height, last)
				}

				for i, line := range lines {
					if got := lipgloss.Width(line); got > size.Width {
						t.Errorf("%dx%d: line %d spans %d columns", size.Width, size.Height, i, got)
					}
				}
			}
		})
	}
}

func TestWindowSizeReachesEveryScreen(t *testing.T) {
	m := layoutModel(t, screenNamespaces)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 123, Height: 45})

	if m.width != 123 || m.height != 45 {
		t.Fatalf("model size = %dx%d, want 123x45", m.width, m.height)
	}

	if m.login.width != 123 || m.login.height != 45 {
		t.Errorf("login size = %dx%d, want 123x45", m.login.width, m.login.height)
	}

	if m.namespaces.width != 123 || m.namespaces.height != 45 {
		t.Errorf("namespaces size = %dx%d, want 123x45", m.namespaces.width, m.namespaces.height)
	}

	if m.devices.width != 123 || m.devices.height != 45 {
		t.Errorf("devices size = %dx%d, want 123x45", m.devices.width, m.devices.height)
	}

	if got := m.namespaces.list.Height(); got != m.namespaces.listHeight() {
		t.Errorf("list height = %d, want %d", got, m.namespaces.listHeight())
	}
}

func TestTableFollowsTheAvailableWidth(t *testing.T) {
	m := layoutModel(t, screenDevices)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 24})

	header := stripANSI(strings.Split(m.devices.table.View(), "\n")[0])
	if got := utf8.RuneCountInString(header); got != 120 {
		t.Errorf("table header spans %d columns, want 120", got)
	}

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 30, Height: 24})

	if got := m.devices.columns()[0].Width; got != minNameWidth {
		t.Errorf("name column = %d, want the %d floor", got, minNameWidth)
	}
}

func TestTableFitsTheAvailableHeight(t *testing.T) {
	m := layoutModel(t, screenDevices)

	for _, height := range []int{24, 50, 6} {
		m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 100, Height: height})

		rendered := lipgloss.Height(m.devices.table.View())
		if rendered != m.devices.tableHeight() {
			t.Errorf("height %d: table rendered %d lines, want %d", height, rendered, m.devices.tableHeight())
		}
	}
}
