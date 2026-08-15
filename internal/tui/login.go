package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/znaniye/shellhubctl/internal/auth"
	"github.com/znaniye/shellhubctl/internal/shellhub"
)

type loginModel struct {
	pctx *ProgramContext

	identifier textinput.Model
	password   textinput.Model

	focused int
	loading bool
	err     string
}

func newLoginModel(pctx *ProgramContext) loginModel {
	identifier := textinput.New()
	identifier.Placeholder = "username or email"
	identifier.CharLimit = 128

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword
	password.CharLimit = 128

	m := loginModel{
		pctx:       pctx,
		identifier: identifier,
		password:   password,
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

			return *m, loginCmd(
				m.pctx.Ctx, m.pctx.Client, m.pctx.Store, m.identifier.Value(), m.password.Value(),
			)
		}
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

func (m loginModel) body(spinner string) string {
	var body strings.Builder

	body.WriteString(m.identifier.View())
	body.WriteString("\n")
	body.WriteString(m.password.View())

	if m.loading {
		body.WriteString("\n\n")
		body.WriteString(spinner)
		body.WriteString(" authenticating…")
	}

	if m.err != "" {
		body.WriteString("\n\n")
		body.WriteString(m.pctx.Styles.Error.Render(m.err))
	}

	return body.String()
}
