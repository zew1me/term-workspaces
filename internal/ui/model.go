package ui

import (
	"fmt"
	"strings"
)

type Model struct {
	tabs      []string
	activeTab int
	sections  map[string][]string
}

func NewDummyModel() Model {
	return Model{
		tabs: []string{
			"PR Queue",
			"Open Sessions",
			"Events",
		},
		activeTab: 0,
		sections: map[string][]string{
			"PR Queue": {
				"#211 owner/repo feature/ui-scaffold",
				"#198 owner/repo fix/session-reconcile",
				"#176 owner/repo docs/dashboard",
			},
			"Open Sessions": {
				"task-owner-repo-211 pane=901 workspace=task-owner-repo-211",
				"task-owner-repo-198 pane=902 workspace=task-owner-repo-198",
			},
			"Events": {
				"[ok] loaded local cache",
				"[ok] fetched task aliases",
				"[ok] rendered dummy dashboard",
			},
		},
	}
}

func (m Model) NextTab() Model {
	m.activeTab = (m.activeTab + 1) % len(m.tabs)
	return m
}

func (m Model) PrevTab() Model {
	m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
	return m
}

func (m Model) SelectTab(index int) Model {
	if index >= 0 && index < len(m.tabs) {
		m.activeTab = index
	}
	return m
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("ttt UI (scaffold)\n")
	b.WriteString("keys: tab/shift+tab move | 1/2/3 jump | q quit\n\n")
	for i, tab := range m.tabs {
		if i == m.activeTab {
			b.WriteString(fmt.Sprintf("[ %s ] ", tab))
		} else {
			b.WriteString(fmt.Sprintf("  %s   ", tab))
		}
	}
	b.WriteString("\n\n")

	active := m.tabs[m.activeTab]
	for _, row := range m.sections[active] {
		b.WriteString("- ")
		b.WriteString(row)
		b.WriteString("\n")
	}
	return b.String()
}
