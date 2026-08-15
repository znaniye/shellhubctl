package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/znaniye/shellhubctl/internal/auth"
	"github.com/znaniye/shellhubctl/internal/shellhub"
)

type Options struct {
	Client  *shellhub.Client
	Store   *auth.Store
	Session *auth.Session
}

func Run(ctx context.Context, opts Options) error {
	m := newModel(ctx, opts)

	p := tea.NewProgram(m, tea.WithContext(ctx), tea.WithAltScreen())

	_, err := p.Run()

	return err
}

const (
	defaultWidth  = 80
	defaultHeight = 24
)

type screen int

const (
	screenLogin screen = iota
	screenDashboard
)

type model struct {
	pctx    *ProgramContext
	session *auth.Session

	screen screen

	help     help.Model
	spinner  spinner.Model
	spinning bool

	namespaces []shellhub.Namespace
	current    int
	nsLoading  bool
	nsErr      string

	login      loginModel
	devices    devicesModel
	hasDevices bool
}

func newModel(ctx context.Context, opts Options) model {
	pctx := newProgramContext(ctx, opts)

	m := model{
		pctx:    pctx,
		session: opts.Session,
		help:    help.New(),
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
		login:   newLoginModel(pctx),
	}

	if opts.Session == nil {
		m.screen = screenLogin
	} else {
		m.screen = screenDashboard
		m.nsLoading = true
		m.spinning = true
	}

	m.layout()

	return m
}

func (m model) Init() tea.Cmd {
	if m.screen == screenDashboard {
		return tea.Batch(listNamespacesCmd(m.pctx.Ctx, m.pctx.Client), m.spinner.Tick)
	}

	return m.login.initCmd()
}

func (m model) loading() bool {
	return m.nsLoading || m.login.loading || (m.hasDevices && m.devices.loading)
}

func (m *model) startSpinner() tea.Cmd {
	if m.spinning {
		return nil
	}

	m.spinning = true

	return m.spinner.Tick
}

func (m model) capturingInput() bool {
	return m.hasDevices && m.devices.capturingInput()
}

func (m *model) layout() {
	m.help.Width = m.pctx.MainContentWidth

	m.pctx.setBodyHeight(
		m.pctx.ScreenHeight - lipgloss.Height(m.headerView()) - lipgloss.Height(m.footerView()),
	)

	m.devices.resize()
}

func (m *model) selectNamespace(index int) tea.Cmd {
	m.current = index
	m.devices = newDevicesModel(m.pctx, m.namespaces[index].Name)
	m.hasDevices = true

	m.layout()

	return tea.Batch(
		switchNamespaceCmd(m.pctx.Ctx, m.pctx.Client, m.pctx.Store, m.namespaces[index]),
		m.startSpinner(),
	)
}

func (m model) initialNamespace() int {
	if m.session == nil {
		return 0
	}

	for i, ns := range m.namespaces {
		if ns.TenantID == m.session.Tenant {
			return i
		}
	}

	return 0
}

func (m *model) moveTab(delta int) tea.Cmd {
	if len(m.namespaces) < 2 {
		return nil
	}

	return m.selectNamespace((m.current + delta + len(m.namespaces)) % len(m.namespaces))
}

func (m *model) loadNamespaces() tea.Cmd {
	m.nsLoading = true
	m.nsErr = ""

	return tea.Batch(listNamespacesCmd(m.pctx.Ctx, m.pctx.Client), m.startSpinner())
}

func (m *model) refresh() tea.Cmd {
	if m.nsErr != "" || len(m.namespaces) == 0 {
		return m.loadNamespaces()
	}

	if !m.hasDevices {
		return nil
	}

	cmd := m.devices.reload()

	m.layout()

	return tea.Batch(cmd, m.startSpinner())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.pctx.SetSize(msg.Width, msg.Height)
		m.layout()

		return m, nil
	case spinner.TickMsg:
		if !m.loading() {
			m.spinning = false

			return m, nil
		}

		var cmd tea.Cmd

		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	case tea.KeyMsg:
		if m.screen == screenDashboard && !m.capturingInput() {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "?":
				m.help.ShowAll = !m.help.ShowAll
				m.layout()

				return m, nil
			case "left", "[":
				return m, m.moveTab(-1)
			case "right", "]":
				return m, m.moveTab(1)
			case "r":
				return m, m.refresh()
			}
		}
	case loginSuccessMsg:
		m.session = msg.session
		m.screen = screenDashboard

		cmd := m.loadNamespaces()

		m.layout()

		return m, cmd
	case namespacesLoadedMsg:
		m.nsLoading = false
		m.nsErr = ""
		m.namespaces = msg.namespaces

		m.layout()

		if len(m.namespaces) == 0 {
			return m, nil
		}

		return m, m.selectNamespace(m.initialNamespace())
	case namespaceSelectedMsg:
		m.session = msg.session

		return m, tea.Batch(m.devices.initCmd(), m.startSpinner())
	case errMsg:
		if m.screen == screenDashboard && m.nsLoading {
			m.nsLoading = false
			m.nsErr = msg.err.Error()

			m.layout()

			return m, nil
		}
	}

	if m.screen == screenLogin {
		var cmd tea.Cmd

		m.login, cmd = m.login.Update(msg)

		return m, tea.Batch(cmd, m.tick())
	}

	if !m.hasDevices {
		return m, nil
	}

	var cmd tea.Cmd

	m.devices, cmd = m.devices.Update(msg)

	m.layout()

	return m, tea.Batch(cmd, m.tick())
}

func (m *model) tick() tea.Cmd {
	if !m.loading() {
		return nil
	}

	return m.startSpinner()
}

func (m model) keyMap() help.KeyMap {
	switch {
	case m.screen == screenLogin:
		return loginKeys
	case m.capturingInput():
		return promptKeys
	default:
		return dashboardKeys
	}
}

func (m model) statusView() string {
	if m.session == nil || m.session.Username == "" {
		return ""
	}

	return m.pctx.Styles.Hint.Render(m.session.Username)
}

func (m model) contextView() string {
	if m.screen == screenLogin {
		return m.pctx.Styles.Subtitle.Render("sign in with your ShellHub account")
	}

	return m.tabsView()
}

func (m model) headerView() string {
	return strings.Join([]string{
		placeApart(
			m.pctx.MainContentWidth,
			m.pctx.Styles.Brand.Render("shellhubctl"),
			m.statusView(),
		),
		m.contextView(),
		m.pctx.rule(),
	}, "\n")
}

func (m model) footerView() string {
	return m.pctx.Styles.FooterBar.
		Width(m.pctx.MainContentWidth).
		Render(m.help.View(m.keyMap()))
}

func (m model) dashboardView() string {
	switch {
	case m.nsLoading:
		return m.spinner.View() + " loading namespaces…"
	case m.nsErr != "":
		return m.pctx.Styles.Error.Render(m.nsErr) + "\n\n" +
			m.pctx.Styles.Hint.Render("press r to retry or q to quit")
	case len(m.namespaces) == 0:
		return m.pctx.Styles.Info.Render("you are not a member of any namespace yet.") + "\n\n" +
			m.pctx.Styles.Hint.Render("press q to quit")
	case m.hasDevices:
		return m.devices.body(m.spinner.View())
	}

	return ""
}

func (m model) bodyView() string {
	if m.screen == screenLogin {
		return m.login.body(m.spinner.View())
	}

	return m.dashboardView()
}

func (m model) View() string {
	if m.pctx.TooSmall() {
		return m.pctx.tooSmallView()
	}

	return m.pctx.render(m.headerView(), m.bodyView(), m.footerView())
}
