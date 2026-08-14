package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/znaniye/shellhub-tui/internal/auth"
	"github.com/znaniye/shellhub-tui/internal/shellhub"
)

type loginModel struct {
	ctx    context.Context
	client *shellhub.Client
	store  *auth.Store

	identifier textinput.Model
	password   textinput.Model
	spinner    spinner.Model
	help       help.Model

	focused int
	loading bool
	err     string

	width  int
	height int
}

func newLoginModel(ctx context.Context, c *shellhub.Client, store *auth.Store) loginModel {
	identifier := textinput.New()
	identifier.Placeholder = "username or email"
	identifier.CharLimit = 128

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword
	password.CharLimit = 128

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	h := help.New()
	h.Width = defaultWidth

	m := loginModel{
		ctx:        ctx,
		client:     c,
		store:      store,
		identifier: identifier,
		password:   password,
		spinner:    sp,
		help:       h,
		width:      defaultWidth,
		height:     defaultHeight,
	}
	m.focus()

	return m
}

func (m *loginModel) focus() {
	m.identifier.Blur()
	m.password.Blur()

	if m.focused == 0 {
		m.identifier.Focus()
	} else {
		m.password.Focus()
	}
}

func (m *loginModel) focusNext() {
	m.focused = (m.focused + 1) % 2
	m.focus()
}

func (m *loginModel) focusPrev() {
	m.focused = (m.focused - 1 + 2) % 2
	m.focus()
}

func (m loginModel) canSubmit() bool {
	return m.identifier.Value() != "" && m.password.Value() != ""
}

func loginCmd(ctx context.Context, c *shellhub.Client, store *auth.Store, identifier, password string) tea.Cmd {
	return func() tea.Msg {
		sess, err := auth.LoginWithPassword(ctx, c, identifier, password)
		if err != nil {
			return errMsg{err: err}
		}

		if err := store.Save(sess); err != nil {
			return errMsg{err: fmt.Errorf("could not save the session: %w", err)}
		}

		return loginSuccessMsg{session: sess}
	}
}

func (m loginModel) initCmd() tea.Cmd {
	return nil
}

func (m *loginModel) Update(msg tea.Msg) (loginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.Width = msg.Width

		return *m, nil
	case tea.KeyMsg:
		if m.loading {
			return *m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return *m, tea.Quit
		case "tab":
			m.focusNext()

			return *m, nil
		case "shift+tab":
			m.focusPrev()

			return *m, nil
		case "enter":
			if !m.canSubmit() {
				return *m, nil
			}

			m.loading = true
			m.err = ""

			return *m, tea.Batch(
				loginCmd(m.ctx, m.client, m.store, m.identifier.Value(), m.password.Value()),
				m.spinner.Tick,
			)
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
	}

	var cmd tea.Cmd

	if m.focused == 0 {
		m.identifier, cmd = m.identifier.Update(msg)
	} else {
		m.password, cmd = m.password.Update(msg)
	}

	return *m, cmd
}

func (m loginModel) headerView() string {
	return strings.Join([]string{
		titleStyle.Render("shellhub-tui"),
		subtitleStyle.Render("sign in with your ShellHub account"),
		"",
	}, "\n")
}

func (m loginModel) footerView() string {
	return strings.Join([]string{
		"",
		m.help.View(loginKeys),
	}, "\n")
}

func (m loginModel) View() string {
	var body strings.Builder

	body.WriteString(m.identifier.View())
	body.WriteString("\n")
	body.WriteString(m.password.View())

	if m.loading {
		body.WriteString("\n\n")
		body.WriteString(m.spinner.View())
		body.WriteString(" authenticating…")
	}

	if m.err != "" {
		body.WriteString("\n\n")
		body.WriteString(errorStyle.Render(m.err))
	}

	content := strings.Join([]string{m.headerView(), body.String()}, "\n")

	return frame(content, m.footerView(), m.width, m.height)
}
