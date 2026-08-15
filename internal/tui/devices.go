package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/znaniye/shellhubctl/internal/shellhub"
	"github.com/znaniye/shellhubctl/internal/ssh"
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
	pctx   *ProgramContext
	nsName string

	table table.Model

	devices []shellhub.Device

	userInput textinput.Model

	prompting  bool
	connErr    string
	connDevice shellhub.Device
	lastUser   string

	hasTable bool
	loading  bool
	empty    bool
	err      string
	total    int
}

func newDevicesModel(pctx *ProgramContext, nsName string) devicesModel {
	return devicesModel{
		pctx:    pctx,
		nsName:  nsName,
		loading: true,
	}
}

func (m devicesModel) initCmd() tea.Cmd {
	return listDevicesCmd(m.pctx.Ctx, m.pctx.Client)
}

func (m devicesModel) capturingInput() bool {
	return m.prompting
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
	t.SetStyles(m.pctx.Styles.Table)

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

	surplus := m.pctx.MainContentWidth - fixed - minNameWidth
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
	cols[nameColumn].Width = m.pctx.MainContentWidth - fixed - osGrow

	return cols
}

func (m devicesModel) countView() string {
	return m.pctx.Styles.Hint.Render(deviceCountFooter(len(m.table.Rows()), m.total))
}

func (m devicesModel) tableHeight() int {
	height := m.pctx.BodyHeight - lipgloss.Height(m.countView())

	if notice := m.noticeView(); notice != "" {
		height -= lipgloss.Height(notice)
	}

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
	m.table.SetWidth(m.pctx.MainContentWidth)
	m.table.SetHeight(m.tableHeight())
}

func (m *devicesModel) reload() tea.Cmd {
	m.loading = true
	m.err = ""

	return listDevicesCmd(m.pctx.Ctx, m.pctx.Client)
}

func (m *devicesModel) Update(msg tea.Msg) (devicesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return *m, nil
		}

		if m.prompting {
			return m.promptUpdate(msg)
		}

		if msg.String() == "enter" {
			m.setConnErr("")

			if m.hasTable && m.table.Cursor() >= 0 && m.table.Cursor() < len(m.devices) {
				dev := m.devices[m.table.Cursor()]
				if !dev.Online {
					m.setConnErr("device " + dev.Name + " is offline")

					return *m, nil
				}

				m.connectTo(dev)
			}

			return *m, nil
		}
	case errMsg:
		m.loading = false
		m.err = msg.err.Error()

		return *m, nil
	case sshExitedMsg:
		m.prompting = false

		if err := sshExitError(msg.err); err != "" {
			m.setConnErr(err)
		}

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

func (m *devicesModel) setConnErr(err string) {
	m.connErr = err

	m.resize()
}

func (m *devicesModel) connectTo(dev shellhub.Device) {
	m.setConnErr("")
	m.connDevice = dev

	m.beginPrompt()
}

func (m *devicesModel) beginPrompt() {
	input := textinput.New()
	input.Placeholder = "root"
	input.CharLimit = 64

	user := m.lastUser
	if user == "" {
		user = "root"
	}

	input.SetValue(user)
	input.Focus()

	m.userInput = input
	m.prompting = true
}

func (m *devicesModel) promptUpdate(msg tea.KeyMsg) (devicesModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		user := strings.TrimSpace(m.userInput.Value())
		if user == "" {
			return *m, nil
		}

		m.lastUser = user
		m.prompting = false

		cfg := ssh.Config{
			User:      user,
			Namespace: m.nsName,
			Device:    m.connDevice.Name,
			Host:      m.pctx.Client.Host(),
		}

		if err := cfg.Validate(); err != nil {
			m.setConnErr(err.Error())

			return *m, nil
		}

		return *m, sshConnectCmd(m.pctx.Ctx, cfg)
	case "esc":
		m.prompting = false

		return *m, nil
	case "ctrl+c":
		return *m, tea.Quit
	}

	var cmd tea.Cmd

	m.userInput, cmd = m.userInput.Update(msg)

	return *m, cmd
}

func (m devicesModel) body(spinner string) string {
	switch {
	case m.prompting:
		return m.connectView()
	case m.loading:
		return spinner + " loading devices…"
	case m.err != "":
		return m.pctx.Styles.Error.Render(m.err) + "\n\n" +
			m.pctx.Styles.Hint.Render("press r to retry or q to quit")
	case m.empty:
		return m.pctx.Styles.Info.Render("no accepted device in this namespace.") + "\n\n" +
			m.pctx.Styles.Hint.Render("press ←/→ to change namespace or q to quit")
	case m.hasTable:
		return m.table.View() + "\n" + m.countView() + m.noticeView()
	}

	return ""
}

func (m devicesModel) connectView() string {
	return strings.Join([]string{
		m.pctx.Styles.Title.Render("connect to " + m.connDevice.Name),
		m.pctx.Styles.Subtitle.Render("namespace " + m.nsName),
		"",
		"OS user on the device:",
		m.userInput.View(),
	}, "\n")
}

func (m devicesModel) noticeView() string {
	if m.connErr == "" {
		return ""
	}

	return "\n" + m.pctx.Styles.Error.Render(m.connErr)
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
