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

type Sections struct {
	PRQueue      []string
	OpenSessions []string
	Events       []string
}

func NewModelFromSections(input Sections) Model {
	return Model{
		tabs: []string{
			"PR Queue",
			"Open Sessions",
			"Events",
		},
		activeTab: 0,
		sections: map[string][]string{
			"PR Queue":      fallbackRows(input.PRQueue, "no task aliases found"),
			"Open Sessions": fallbackRows(input.OpenSessions, "no open sessions"),
			"Events":        fallbackRows(input.Events, "no events"),
		},
	}
}

func NewDummyModel() Model {
	return NewModelFromSections(Sections{
		PRQueue: []string{
			"#211 owner/repo feature/ui-scaffold",
			"#198 owner/repo fix/session-reconcile",
			"#176 owner/repo docs/dashboard",
		},
		OpenSessions: []string{
			"task-owner-repo-211 pane=901 workspace=task-owner-repo-211",
			"task-owner-repo-198 pane=902 workspace=task-owner-repo-198",
		},
		Events: []string{
			"[ok] loaded local cache",
			"[ok] fetched task aliases",
			"[ok] rendered dummy dashboard",
		},
	})
}

func (m Model) NextTab() Model {
	if len(m.tabs) == 0 {
		return m
	}
	m.activeTab = (m.activeTab + 1) % len(m.tabs)
	return m
}

func (m Model) PrevTab() Model {
	if len(m.tabs) == 0 {
		return m
	}
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
	b.WriteString("ttt UI (preview)\n")
	b.WriteString("keys: tab/shift+tab move | 1/2/3 jump | q quit\n\n")
	if len(m.tabs) == 0 {
		return b.String()
	}

	safeActiveTab := m.activeTab
	if safeActiveTab < 0 || safeActiveTab >= len(m.tabs) {
		safeActiveTab = 0
	}

	for i, tab := range m.tabs {
		if i == safeActiveTab {
			b.WriteString(fmt.Sprintf("[ %s ] ", tab))
		} else {
			b.WriteString(fmt.Sprintf("  %s   ", tab))
		}
	}
	b.WriteString("\n\n")

	active := m.tabs[safeActiveTab]
	for _, row := range m.sections[active] {
		b.WriteString("- ")
		b.WriteString(row)
		b.WriteString("\n")
	}
	return b.String()
}

func fallbackRows(rows []string, fallback string) []string {
	if len(rows) == 0 {
		return []string{fallback}
	}
	return rows
}
