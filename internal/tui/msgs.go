package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/znaniye/shellhub-tui/internal/auth"
	"github.com/znaniye/shellhub-tui/internal/shellhub"
)

type errMsg struct {
	err error
}

type loginSuccessMsg struct {
	session *auth.Session
}

type namespacesLoadedMsg struct {
	namespaces []shellhub.Namespace
	total      int
}

type namespaceSelectedMsg struct {
	namespace shellhub.Namespace
	session   *auth.Session
}

type devicesLoadedMsg struct {
	devices []shellhub.Device
	total   int
}

type backMsg struct{}

func back() tea.Msg {
	return backMsg{}
}
