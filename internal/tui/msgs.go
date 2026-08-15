package tui

import (
	"github.com/znaniye/shellhubctl/internal/auth"
	"github.com/znaniye/shellhubctl/internal/shellhub"
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

type sshExitedMsg struct {
	err error
}
