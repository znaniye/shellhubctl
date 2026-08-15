package tui

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/znaniye/shellhubctl/internal/shellhub"
)

var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func layoutModel(t *testing.T, s screen) model {
	t.Helper()

	pctx := newProgramContext(context.Background(), Options{})

	m := model{
		pctx:    pctx,
		screen:  s,
		help:    help.New(),
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
		login:   newLoginModel(pctx),
		namespaces: []shellhub.Namespace{
			{Name: "alpha", TenantID: "t-1", DevicesAcceptedCount: 2},
			{Name: "beta", TenantID: "t-2", DevicesAcceptedCount: 7},
		},
	}

	m.devices = newDevicesModel(pctx, "alpha")
	m.devices.loading = false
	m.hasDevices = true
	m.devices.setDevices([]shellhub.Device{
		{UID: "d-1", Name: "dev1", Status: "accepted", Online: true},
		{UID: "d-2", Name: "dev2", Status: "accepted"},
	}, 2)

	m.layout()

	return m
}

func renderContext(t *testing.T, width, height int) *ProgramContext {
	t.Helper()

	pctx := newProgramContext(context.Background(), Options{})
	pctx.SetSize(width, height)

	return pctx
}

func TestRenderAnchorsTheFooterAtTheBottom(t *testing.T) {
	lines := strings.Split(renderContext(t, 20, 6).render("title", "body", "help"), "\n")

	if len(lines) != 6 {
		t.Fatalf("render produced %d lines, want 6", len(lines))
	}

	if !strings.Contains(lines[0], "title") {
		t.Errorf("first line = %q, want the header", lines[0])
	}

	if !strings.Contains(lines[1], "body") {
		t.Errorf("second line = %q, want the body", lines[1])
	}

	if !strings.Contains(lines[5], "help") {
		t.Errorf("last line = %q, want the footer", lines[5])
	}
}

func TestRenderIndentsEveryLine(t *testing.T) {
	for _, line := range strings.Split(renderContext(t, 20, 4).render("title", "body", "help"), "\n") {
		if !strings.HasPrefix(line, strings.Repeat(" ", contentHorizontalPadding)) {
			t.Errorf("line %q is not indented", line)
		}
	}
}

func TestRenderTruncatesOverflowingContent(t *testing.T) {
	body := strings.Join([]string{"a", "b", "c", "d", "e"}, "\n")

	lines := strings.Split(renderContext(t, 20, 4).render("title", body, "help"), "\n")

	if len(lines) != 4 {
		t.Fatalf("render produced %d lines, want 4", len(lines))
	}

	if !strings.Contains(lines[3], "help") {
		t.Errorf("last line = %q, want the footer to survive the overflow", lines[3])
	}

	if strings.Contains(stripANSI(lines[2]), "c") {
		t.Errorf("third line = %q, want the body cut at the footer", lines[2])
	}
}

func TestRenderClampsEveryLineToTheWidth(t *testing.T) {
	pctx := renderContext(t, 10, 3)

	view := pctx.render("a much longer header", "a much longer body", "a much longer footer")

	for i, line := range strings.Split(view, "\n") {
		if got := utf8.RuneCountInString(stripANSI(line)); got > 10 {
			t.Errorf("line %d spans %d columns, want at most 10", i, got)
		}
	}
}

func TestRenderKeepsStyledTextWithinTheWidth(t *testing.T) {
	styled := lipgloss.NewStyle().Bold(true).Render("hello world")

	view := renderContext(t, 5, 3).render(styled, styled, styled)

	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > 5 {
			t.Errorf("line %q spans %d columns, want at most 5", stripANSI(line), got)
		}
	}
}

func TestViewFillsTheScreenAndAnchorsHelp(t *testing.T) {
	screens := []struct {
		name   string
		screen screen
	}{
		{name: "login", screen: screenLogin},
		{name: "dashboard", screen: screenDashboard},
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

				if first := stripANSI(lines[0]); !strings.Contains(first, "shellhubctl") {
					t.Errorf("height %d: first line = %q, want the brand", size.Height, first)
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

func TestDashboardShowsTheNamespaceTabs(t *testing.T) {
	m := layoutModel(t, screenDashboard)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 24})

	tabs := stripANSI(strings.Split(m.View(), "\n")[1])
	for _, want := range []string{"alpha", "beta"} {
		if !strings.Contains(tabs, want) {
			t.Errorf("tab row = %q, want it to list %q", tabs, want)
		}
	}
}

func TestTabsNeverOverflowTheWidth(t *testing.T) {
	m := layoutModel(t, screenDashboard)

	m.namespaces = nil
	for i := range 40 {
		m.namespaces = append(m.namespaces, shellhub.Namespace{
			Name:                 strings.Repeat("namespace", 2) + string(rune('a'+i%26)),
			DevicesAcceptedCount: i,
		})
	}

	for _, current := range []int{0, 17, 39} {
		m.current = current

		m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 24})

		if got := lipgloss.Width(m.tabsView()); got > m.pctx.MainContentWidth {
			t.Errorf("current %d: tab row spans %d columns, want at most %d",
				current, got, m.pctx.MainContentWidth)
		}

		if !strings.Contains(stripANSI(m.tabsView()), m.namespaces[current].Name) {
			t.Errorf("current %d: tab row dropped the active namespace", current)
		}
	}
}

func TestWindowSizeReachesEveryScreen(t *testing.T) {
	m := layoutModel(t, screenDashboard)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 123, Height: 45})

	if m.pctx.ScreenWidth != 123 || m.pctx.ScreenHeight != 45 {
		t.Fatalf("context size = %dx%d, want 123x45", m.pctx.ScreenWidth, m.pctx.ScreenHeight)
	}

	for name, sub := range map[string]*ProgramContext{
		"login":   m.login.pctx,
		"devices": m.devices.pctx,
	} {
		if sub != m.pctx {
			t.Errorf("%s holds a different context than the root model", name)
		}
	}

	if got := m.pctx.BodyHeight; got != 45-headerHeight-footerHeight {
		t.Errorf("body height = %d, want %d", got, 45-headerHeight-footerHeight)
	}
}

func TestTableFollowsTheAvailableWidth(t *testing.T) {
	m := layoutModel(t, screenDashboard)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 24})

	header := stripANSI(strings.Split(m.devices.table.View(), "\n")[0])
	if got := utf8.RuneCountInString(header); got != m.pctx.MainContentWidth {
		t.Errorf("table header spans %d columns, want %d", got, m.pctx.MainContentWidth)
	}

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 30, Height: 24})

	if got := m.devices.columns()[0].Width; got != minNameWidth {
		t.Errorf("name column = %d, want the %d floor", got, minNameWidth)
	}
}

func TestTableFitsTheAvailableHeight(t *testing.T) {
	m := layoutModel(t, screenDashboard)

	for _, height := range []int{24, 50, 8} {
		m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 100, Height: height})

		rendered := lipgloss.Height(m.devices.table.View())
		if rendered != m.devices.tableHeight() {
			t.Errorf("height %d: table rendered %d lines, want %d", height, rendered, m.devices.tableHeight())
		}
	}
}

func devicesWithOS(t *testing.T) model {
	t.Helper()

	m := layoutModel(t, screenDashboard)
	m.devices.setDevices([]shellhub.Device{
		{UID: "d-1", Name: "dev1", Status: "accepted", Online: true, Info: shellhub.DeviceInfo{
			PrettyName: "Ubuntu 22.04.3 LTS", Arch: "x86_64",
		}},
		{UID: "d-2", Name: "dev2", Status: "accepted", Info: shellhub.DeviceInfo{
			PrettyName: "NixOS 24.05 (Uakari)", Arch: "aarch64",
		}},
	}, 2)

	return m
}

func TestOSColumnGrowsWithTheScreen(t *testing.T) {
	m := devicesWithOS(t)

	want := m.devices.cellWidth(osColumn)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 60, Height: 24})
	narrow := m.devices.columns()[osColumn].Width

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 160, Height: 24})
	wide := m.devices.columns()[osColumn].Width

	if narrow >= wide {
		t.Errorf("OS column stayed at %d columns from 60 to %d, want it to grow", narrow, wide)
	}

	if wide != want {
		t.Errorf("OS column = %d at 160 columns, want %d to fit the longest label", wide, want)
	}

	if strings.Contains(stripANSI(m.devices.table.View()), "…") {
		t.Error("OS label is still truncated with room to spare")
	}
}

func TestTableSpansTheWidthWhateverTheContent(t *testing.T) {
	m := devicesWithOS(t)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 1, Height: 24})

	floor := 0
	for _, col := range m.devices.columns() {
		floor += col.Width + tableCellPadding
	}

	for _, content := range []int{floor, floor + 20, floor + 60, floor + 100} {
		width := content + contentHorizontalPadding*2

		m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: width, Height: 24})

		total := 0
		for _, col := range m.devices.columns() {
			total += col.Width + tableCellPadding
		}

		if total != content {
			t.Errorf("width %d: columns span %d, want %d", width, total, content)
		}

		header := stripANSI(strings.Split(m.devices.table.View(), "\n")[0])
		if got := utf8.RuneCountInString(header); got != content {
			t.Errorf("width %d: header spans %d columns", width, got)
		}
	}

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: floor - 20, Height: 24})

	for i, line := range strings.Split(m.View(), "\n") {
		if got := utf8.RuneCountInString(stripANSI(line)); got > floor-20 {
			t.Errorf("below the floor, line %d spans %d columns, want at most %d", i, got, floor-20)
		}
	}
}

func devicesWithOSLabel(t *testing.T, pretty, arch string) model {
	t.Helper()

	m := layoutModel(t, screenDashboard)
	m.devices.setDevices([]shellhub.Device{
		{UID: "d-1", Name: "dev1", Status: "accepted", Online: true, Info: shellhub.DeviceInfo{
			PrettyName: pretty, Arch: arch,
		}},
	}, 1)

	return m
}

func TestOSColumnStaysAtTheDefaultWhenTheLabelFits(t *testing.T) {
	m := devicesWithOSLabel(t, "Ubuntu", "x86_64")

	for _, width := range []int{80, 120, 200} {
		m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: width, Height: 24})

		if got := m.devices.columns()[osColumn].Width; got != minOSWidth {
			t.Errorf("width %d: OS column = %d, want it to stay at the %d default", width, got, minOSWidth)
		}
	}

	if strings.Contains(stripANSI(m.devices.table.View()), "…") {
		t.Error("a label that fits was truncated")
	}
}

func TestOSColumnStopsAtTheCap(t *testing.T) {
	m := devicesWithOSLabel(t, strings.Repeat("distro ", 12), "x86_64")

	for _, width := range []int{140, 200, 300} {
		m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: width, Height: 24})

		if got := m.devices.columns()[osColumn].Width; got != maxOSWidth {
			t.Errorf("width %d: OS column = %d, want the %d cap", width, got, maxOSWidth)
		}
	}

	if !strings.Contains(stripANSI(m.devices.table.View()), "…") {
		t.Error("a label past the cap should stay truncated")
	}
}

func TestFixedColumnsDoNotWasteWidth(t *testing.T) {
	m := devicesWithOS(t)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 160, Height: 24})

	for i, col := range m.devices.columns() {
		if i == nameColumn || i == osColumn {
			continue
		}

		want := max(lipgloss.Width(col.Title), m.devices.cellWidth(i))
		if col.Width != want {
			t.Errorf("%s is %d wide, want %d so it holds its content and no more", col.Title, col.Width, want)
		}
	}
}

func TestLastColumnEndsAtTheRightEdge(t *testing.T) {
	m := devicesWithOS(t)

	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 24})

	cols := m.devices.columns()
	last := cols[len(cols)-1]

	widest := max(lipgloss.Width(last.Title), m.devices.cellWidth(len(cols)-1))
	if slack := last.Width - widest; slack != 0 {
		t.Errorf("%s leaves %d unused columns before the right edge", last.Title, slack)
	}

	rows := strings.Split(stripANSI(m.devices.table.View()), "\n")
	for i, row := range rows {
		if got := utf8.RuneCountInString(row); got != m.pctx.MainContentWidth {
			t.Errorf("row %d spans %d columns, want %d", i, got, m.pctx.MainContentWidth)
		}
	}
}
