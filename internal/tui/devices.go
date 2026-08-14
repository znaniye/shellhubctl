package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/znaniye/shellhub-tui/internal/auth"
	"github.com/znaniye/shellhub-tui/internal/shellhub"
)

const (
	minTableHeight   = 3
	minNameWidth     = 12
	minOSWidth       = 20
	maxOSWidth       = 50
	tableCellPadding = 2

	nameColumn = 0
	osColumn   = 3
)

type devicesModel struct {
	ctx    context.Context
	client *shellhub.Client
	store  *auth.Store
	nsName string

	table   table.Model
	spinner spinner.Model
	help    help.Model

	devices []shellhub.Device

	hasTable bool
	loading  bool
	empty    bool
	err      string
	total    int

	width  int
	height int
}

func newDevicesModel(ctx context.Context, c *shellhub.Client, store *auth.Store, nsName string, width, height int) devicesModel {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	h := help.New()
	h.Width = width

	return devicesModel{
		ctx:     ctx,
		client:  c,
		store:   store,
		nsName:  nsName,
		spinner: sp,
		help:    h,
		loading: true,
		width:   width,
		height:  height,
	}
}

func (m devicesModel) initCmd() tea.Cmd {
	return tea.Batch(listDevicesCmd(m.ctx, m.client), m.spinner.Tick)
}

func (m *devicesModel) setDevices(devices []shellhub.Device, total int) {
	m.total = total
	m.devices = devices
	m.empty = len(devices) == 0
	m.hasTable = false

	if m.empty {
		return
	}

	rows := make([]table.Row, 0, len(devices))
	for _, d := range devices {
		rows = append(rows, deviceRow(d))
	}

	t := table.New(
		table.WithColumns(m.columns()),
		table.WithRows(rows),
		table.WithFocused(true),
	)
	t.SetStyles(tableStyles)

	m.table = t
	m.hasTable = true

	m.resize()
}

func (m devicesModel) cellWidth(column int) int {
	width := 0

	for _, d := range m.devices {
		if got := lipgloss.Width(deviceRow(d)[column]); got > width {
			width = got
		}
	}

	return width
}

func (m devicesModel) columns() []table.Column {
	cols := []table.Column{
		{Title: "NAME", Width: minNameWidth},
		{Title: "STATUS"},
		{Title: "ONLINE"},
		{Title: "OS", Width: minOSWidth},
		{Title: "LAST SEEN"},
	}

	for i, col := range cols {
		if i == nameColumn || i == osColumn {
			continue
		}

		cols[i].Width = max(lipgloss.Width(col.Title), m.cellWidth(i))
	}

	fixed := tableCellPadding
	for _, col := range cols[1:] {
		fixed += col.Width + tableCellPadding
	}

	surplus := m.width - fixed - minNameWidth
	if surplus <= 0 {
		return cols
	}

	nameWant := max(0, m.cellWidth(nameColumn)-minNameWidth)
	osWant := max(0, min(m.cellWidth(osColumn), maxOSWidth)-minOSWidth)

	osGrow := osWant
	if total := nameWant + osWant; total > surplus {
		osGrow = surplus * osWant / total
	}

	cols[osColumn].Width += osGrow
	cols[nameColumn].Width = m.width - fixed - osGrow

	return cols
}

func (m devicesModel) headerView() string {
	return strings.Join([]string{
		titleStyle.Render("devices"),
		subtitleStyle.Render("namespace " + m.nsName + " · esc to go back"),
		"",
	}, "\n")
}

func (m devicesModel) countView() string {
	return hintStyle.Render(deviceCountFooter(len(m.table.Rows()), m.total))
}

func (m devicesModel) footerView() string {
	return strings.Join([]string{
		"",
		m.help.View(navKeys),
	}, "\n")
}

func (m devicesModel) tableHeight() int {
	height := m.height -
		lipgloss.Height(m.headerView()) -
		lipgloss.Height(m.countView()) -
		lipgloss.Height(m.footerView())

	if height < minTableHeight {
		return minTableHeight
	}

	return height
}

func (m *devicesModel) resize() {
	if !m.hasTable {
		return
	}

	m.table.SetColumns(m.columns())
	m.table.SetWidth(m.width)
	m.table.SetHeight(m.tableHeight())
}

func (m *devicesModel) reload() tea.Cmd {
	m.loading = true
	m.err = ""

	return tea.Batch(listDevicesCmd(m.ctx, m.client), m.spinner.Tick)
}

func (m *devicesModel) Update(msg tea.Msg) (devicesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.Width = msg.Width

		m.resize()

		return *m, nil
	case tea.KeyMsg:
		if m.loading {
			return *m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return *m, tea.Quit
		case "esc":
			return *m, back
		case "r":
			return *m, m.reload()
		case "?":
			m.help.ShowAll = !m.help.ShowAll

			m.resize()

			return *m, nil
		}
	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spinner, cmd = m.spinner.Update(msg)
		if !m.loading {
			return *m, nil
		}

		return *m, cmd
	case errMsg:
		m.loading = false
		m.err = msg.err.Error()

		return *m, nil
	case devicesLoadedMsg:
		m.loading = false
		m.err = ""
		m.setDevices(msg.devices, msg.total)

		return *m, nil
	}

	if m.hasTable {
		var cmd tea.Cmd

		m.table, cmd = m.table.Update(msg)

		return *m, cmd
	}

	return *m, nil
}

func (m devicesModel) View() string {
	var body string

	switch {
	case m.loading:
		body = m.spinner.View() + " loading devices…"
	case m.err != "":
		body = errorStyle.Render(m.err) + "\n\n" +
			hintStyle.Render("press r to retry, esc to go back or q to quit")
	case m.empty:
		body = infoStyle.Render("no accepted device in this namespace.") + "\n\n" +
			hintStyle.Render("press esc to go back or q to quit")
	case m.hasTable:
		body = m.table.View() + "\n" + m.countView()
	}

	content := strings.Join([]string{m.headerView(), body}, "\n")

	return frame(content, m.footerView(), m.width, m.height)
}

func deviceRow(d shellhub.Device) table.Row {
	return table.Row{
		d.Name,
		d.Status,
		onlineLabel(d.Online),
		osLabel(d.Info),
		formatLastSeen(d.LastSeen),
	}
}

func deviceCountFooter(shown, total int) string {
	return fmt.Sprintf("%d of %d devices", shown, total)
}

func onlineLabel(online bool) string {
	if online {
		return "yes"
	}

	return "no"
}

func osLabel(info shellhub.DeviceInfo) string {
	name := info.PrettyName
	if name == "" {
		name = info.ID
	}

	if name == "" {
		return "-"
	}

	if info.Arch != "" && info.Arch != "unknown" {
		return name + " · " + info.Arch
	}

	return name
}

func formatLastSeen(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d d ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}
