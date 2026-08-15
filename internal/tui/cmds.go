package tui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/znaniye/shellhubctl/internal/auth"
	"github.com/znaniye/shellhubctl/internal/shellhub"
)

func listNamespacesCmd(ctx context.Context, c *shellhub.Client) tea.Cmd {
	return func() tea.Msg {
		namespaces, total, err := c.ListNamespaces(ctx, 1, 100)
		if err != nil {
			return errMsg{err: err}
		}

		return namespacesLoadedMsg{namespaces: namespaces, total: total}
	}
}

func switchNamespaceCmd(ctx context.Context, c *shellhub.Client, store *auth.Store, ns shellhub.Namespace) tea.Cmd {
	return func() tea.Msg {
		ua, err := c.SwitchNamespace(ctx, ns.TenantID)
		if err != nil {
			return errMsg{err: err}
		}

		sess, err := sessionFromAuth(c, ua)
		if err != nil {
			return errMsg{err: err}
		}

		if err := store.Save(sess); err != nil {
			return errMsg{err: fmt.Errorf("could not save the session: %w", err)}
		}

		return namespaceSelectedMsg{namespace: ns, session: sess}
	}
}

func listDevicesCmd(ctx context.Context, c *shellhub.Client) tea.Cmd {
	return func() tea.Msg {
		devices, total, err := c.ListDevices(ctx, shellhub.DeviceListOptions{
			PerPage: 100,
			SortBy:  "name",
			OrderBy: "asc",
		})
		if err != nil {
			return errMsg{err: err}
		}

		slices.SortStableFunc(devices, func(a, b shellhub.Device) int {
			if a.Online != b.Online {
				if a.Online {
					return -1
				}

				return 1
			}

			return strings.Compare(a.Name, b.Name)
		})

		return devicesLoadedMsg{devices: devices, total: total}
	}
}

func sessionFromAuth(c *shellhub.Client, ua *shellhub.UserAuth) (*auth.Session, error) {
	expiresAt, err := auth.TokenExpiry(ua.Token)
	if err != nil {
		return nil, fmt.Errorf("tui: parse token expiry: %w", err)
	}

	return &auth.Session{
		Server:    c.Server(),
		Token:     ua.Token,
		UserID:    ua.ID,
		Username:  ua.User,
		Email:     ua.Email,
		Tenant:    ua.Tenant,
		Role:      ua.Role,
		ExpiresAt: expiresAt,
	}, nil
}
