package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/znaniye/shellhubctl/internal/shellhub"
)

const tabSeparator = " │ "

func tabLabels(namespaces []shellhub.Namespace) []string {
	labels := make([]string, len(namespaces))
	for i, ns := range namespaces {
		labels[i] = fmt.Sprintf("%s %d", ns.Name, ns.DevicesAcceptedCount)
	}

	return labels
}

func tabWindow(labels []string, current, width int) (int, int) {
	if len(labels) == 0 {
		return 0, 0
	}

	current = min(max(current, 0), len(labels)-1)

	start, end := current, current+1
	total := lipgloss.Width(labels[current])
	separator := lipgloss.Width(tabSeparator)

	for {
		grew := false

		if end < len(labels) && total+separator+lipgloss.Width(labels[end]) <= width {
			total += separator + lipgloss.Width(labels[end])
			end++
			grew = true
		}

		if start > 0 && total+separator+lipgloss.Width(labels[start-1]) <= width {
			total += separator + lipgloss.Width(labels[start-1])
			start--
			grew = true
		}

		if !grew {
			return start, end
		}
	}
}

func (m model) tabsView() string {
	labels := tabLabels(m.namespaces)
	if len(labels) == 0 {
		return ""
	}

	start, end := tabWindow(labels, m.current, m.pctx.MainContentWidth)

	tabs := make([]string, 0, end-start)

	for i := start; i < end; i++ {
		style := m.pctx.Styles.TabInactive
		if i == m.current {
			style = m.pctx.Styles.TabActive
		}

		tabs = append(tabs, style.Render(labels[i]))
	}

	row := strings.Join(tabs, m.pctx.Styles.TabSeparator.Render(tabSeparator))

	return ansi.Truncate(row, m.pctx.MainContentWidth, "")
}
