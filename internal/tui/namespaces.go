package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/znaniye/shellhubctl/internal/auth"
	"github.com/znaniye/shellhubctl/internal/shellhub"
)

const (
	minListWidth  = 40
	minListHeight = 3
)

type namespaceItem struct {
	ns shellhub.Namespace
}

func (i namespaceItem) Title() string {
	return i.ns.Name
}

func (i namespaceItem) Description() string {
	return fmt.Sprintf(
		"%d accepted · %d pending · %d rejected",
		i.ns.DevicesAcceptedCount, i.ns.DevicesPendingCount, i.ns.DevicesRejectedCount,
	)
}

func (i namespaceItem) FilterValue() string {
	return i.ns.Name
}

type namespacesModel struct {
	ctx    context.Context
	client *shellhub.Client
	store  *auth.Store
	user   string

	list    list.Model
	spinner spinner.Model
	help    help.Model

	hasList bool
	loading bool
	empty   bool
	err     string

	width  int
	height int
}

func newNamespacesModel(ctx context.Context, c *shellhub.Client, store *auth.Store, user string, width, height int) namespacesModel {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	h := help.New()
	h.Width = width

	return namespacesModel{
		ctx:     ctx,
		client:  c,
		store:   store,
		user:    user,
		spinner: sp,
		help:    h,
		loading: true,
		width:   width,
		height:  height,
	}
}

func (m namespacesModel) initCmd() tea.Cmd {
	return tea.Batch(listNamespacesCmd(m.ctx, m.client), m.spinner.Tick)
}

func (m namespacesModel) filtering() bool {
	return m.hasList && m.list.FilterState() == list.Filtering
}

func (m *namespacesModel) setNamespaces(namespaces []shellhub.Namespace) {
	m.empty = len(namespaces) == 0
	m.hasList = false

	if m.empty {
		return
	}

	items := make([]list.Item, 0, len(namespaces))
	for _, ns := range namespaces {
		items = append(items, namespaceItem{ns: ns})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "namespaces"
	l.SetShowHelp(false)

	m.list = l
	m.hasList = true

	m.resize()
}

func (m namespacesModel) headerView() string {
	subtitle := "pick a namespace to list its devices"
	if m.user != "" {
		subtitle += " · " + m.user
	}

	return strings.Join([]string{
		titleStyle.Render("namespaces"),
		subtitleStyle.Render(subtitle),
		"",
	}, "\n")
}

func (m namespacesModel) footerView() string {
	return strings.Join([]string{
		"",
		m.help.View(navKeys),
	}, "\n")
}

func (m namespacesModel) listWidth() int {
	if m.width < minListWidth {
		return minListWidth
	}

	return m.width
}

func (m namespacesModel) listHeight() int {
	height := m.height - lipgloss.Height(m.headerView()) - lipgloss.Height(m.footerView())
	if height < minListHeight {
		return minListHeight
	}

	return height
}

func (m *namespacesModel) resize() {
	if !m.hasList {
		return
	}

	m.list.SetWidth(m.listWidth())
	m.list.SetHeight(m.listHeight())
}

func (m *namespacesModel) reload() tea.Cmd {
	m.loading = true
	m.err = ""

	return tea.Batch(listNamespacesCmd(m.ctx, m.client), m.spinner.Tick)
}

func (m *namespacesModel) Update(msg tea.Msg) (namespacesModel, tea.Cmd) {
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

		if !m.filtering() {
			switch msg.String() {
			case "q", "ctrl+c", "esc":
				return *m, tea.Quit
			case "r":
				return *m, m.reload()
			case "enter":
				if m.hasList {
					if item, ok := m.list.SelectedItem().(namespaceItem); ok {
						return *m, switchNamespaceCmd(m.ctx, m.client, m.store, item.ns)
					}
				}

				return *m, nil
			}
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
	case namespacesLoadedMsg:
		m.loading = false
		m.err = ""
		m.setNamespaces(msg.namespaces)

		return *m, nil
	}

	if m.hasList {
		var cmd tea.Cmd

		m.list, cmd = m.list.Update(msg)

		return *m, cmd
	}

	return *m, nil
}

func (m namespacesModel) View() string {
	var body string

	switch {
	case m.loading:
		body = m.spinner.View() + " loading namespaces…"
	case m.err != "":
		body = errorStyle.Render(m.err) + "\n\n" +
			hintStyle.Render("press r to retry or q to quit")
	case m.empty:
		body = infoStyle.Render("you are not a member of any namespace yet.") + "\n\n" +
			hintStyle.Render("press q to quit")
	case m.hasList:
		body = m.list.View()
	}

	content := strings.Join([]string{m.headerView(), body}, "\n")

	return frame(content, m.footerView(), m.width, m.height)
}
