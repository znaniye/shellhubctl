package tui

import (
	"context"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

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
	screenNamespaces
	screenDevices
)

type model struct {
	ctx     context.Context
	client  *shellhub.Client
	store   *auth.Store
	session *auth.Session

	namespace shellhub.Namespace

	screen screen

	width  int
	height int

	login      loginModel
	namespaces namespacesModel
	devices    devicesModel
}

func newModel(ctx context.Context, opts Options) model {
	m := model{
		ctx:     ctx,
		client:  opts.Client,
		store:   opts.Store,
		session: opts.Session,
		width:   defaultWidth,
		height:  defaultHeight,
		login:   newLoginModel(ctx, opts.Client, opts.Store),
	}

	if opts.Session == nil {
		m.screen = screenLogin
	} else {
		m.screen = screenNamespaces
		m.namespaces = newNamespacesModel(ctx, opts.Client, opts.Store, opts.Session.Username, m.width, m.height)
	}

	return m
}

func (m model) Init() tea.Cmd {
	switch m.screen {
	case screenNamespaces:
		return m.namespaces.initCmd()
	case screenDevices:
		return m.devices.initCmd()
	default:
		return m.login.initCmd()
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		return m.forwardSize(msg)
	case loginSuccessMsg:
		m.session = msg.session
		m.screen = screenNamespaces
		m.namespaces = newNamespacesModel(m.ctx, m.client, m.store, msg.session.Username, m.width, m.height)

		return m, m.namespaces.initCmd()
	case namespaceSelectedMsg:
		m.session = msg.session
		m.namespace = msg.namespace
		m.screen = screenDevices
		m.devices = newDevicesModel(m.ctx, m.client, m.store, msg.namespace.Name, m.width, m.height)

		return m, m.devices.initCmd()
	case backMsg:
		m.screen = screenNamespaces

		return m, nil
	}

	switch m.screen {
	case screenLogin:
		var cmd tea.Cmd

		m.login, cmd = m.login.Update(msg)

		return m, cmd
	case screenNamespaces:
		var cmd tea.Cmd

		m.namespaces, cmd = m.namespaces.Update(msg)

		return m, cmd
	default:
		var cmd tea.Cmd

		m.devices, cmd = m.devices.Update(msg)

		return m, cmd
	}
}

func (m model) forwardSize(msg tea.WindowSizeMsg) (model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	m.login, cmd = m.login.Update(msg)
	cmds = append(cmds, cmd)

	m.namespaces, cmd = m.namespaces.Update(msg)
	cmds = append(cmds, cmd)

	m.devices, cmd = m.devices.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	switch m.screen {
	case screenLogin:
		return m.login.View()
	case screenNamespaces:
		return m.namespaces.View()
	default:
		return m.devices.View()
	}
}

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

func frame(content, footer string, width, height int) string {
	contentLines := clampWidth(content, width)
	footerLines := clampWidth(footer, width)

	if height <= 0 {
		return strings.Join(slices.Concat(contentLines, footerLines), "\n")
	}

	if len(footerLines) > height {
		footerLines = footerLines[len(footerLines)-height:]
	}

	room := height - len(footerLines)
	if len(contentLines) > room {
		contentLines = contentLines[:room]
	}

	lines := make([]string, height)
	copy(lines, contentLines)
	copy(lines[room:], footerLines)

	return strings.Join(lines, "\n")
}
