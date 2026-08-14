package tui

import (
	"context"
	"errors"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/znaniye/shellhub-tui/internal/ssh"
)

func sshConnectCmd(ctx context.Context, target ssh.Config) tea.Cmd {
	name, args := target.Command()

	return tea.ExecProcess(exec.CommandContext(ctx, name, args...), func(err error) tea.Msg {
		return sshExitedMsg{err: err}
	})
}

func sshExitError(err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, exec.ErrNotFound) {
		return "ssh is not installed on this machine"
	}

	return "ssh session failed: " + err.Error()
}
