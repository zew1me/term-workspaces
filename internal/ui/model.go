package ui

import (
	"fmt"
	"strings"
)

type Model struct {
	tabs          []string
	activeTab     int
	sections      map[string][]string
	selectedByTab map[string]int
	queueTaskIDs  []string
}

type Sections struct {
	PRQueue        []string
	PRQueueTaskIDs []string
	OpenSessions   []string
	Events         []string
}

func NewModelFromSections(input Sections) Model {
	sections := map[string][]string{
		"PR Queue":      fallbackRows(input.PRQueue, "no task aliases found"),
		"Open Sessions": fallbackRows(input.OpenSessions, "no open sessions"),
		"Events":        fallbackRows(input.Events, "no events"),
	}
	return Model{
		tabs: []string{
			"PR Queue",
			"Open Sessions",
			"Events",
		},
		activeTab:    0,
		sections:     sections,
		queueTaskIDs: normalizeTaskIDs(input.PRQueueTaskIDs, len(sections["PR Queue"])),
		selectedByTab: map[string]int{
			"PR Queue":      0,
			"Open Sessions": 0,
			"Events":        0,
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
		PRQueueTaskIDs: []string{
			"task_dummy_211",
			"task_dummy_198",
			"task_dummy_176",
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

func (m Model) ActiveTabIndex() int {
	return m.activeTab
}

func (m Model) ActiveTabName() string {
	if len(m.tabs) == 0 {
		return ""
	}
	safeActiveTab := m.activeTab
	if safeActiveTab < 0 || safeActiveTab >= len(m.tabs) {
		safeActiveTab = 0
	}
	return m.tabs[safeActiveTab]
}

func (m Model) SelectedIndexForTab(tab string) int {
	index, ok := m.selectedByTab[tab]
	if !ok {
		return 0
	}
	return index
}

func (m Model) SetSelectedIndexForTab(tab string, index int) Model {
	rows := m.sections[tab]
	if len(rows) == 0 {
		m.selectedByTab[tab] = 0
		return m
	}
	if index < 0 {
		index = 0
	}
	if index >= len(rows) {
		index = len(rows) - 1
	}
	m.selectedByTab[tab] = index
	return m
}

func (m Model) MoveSelectionDown() Model {
	tab := m.ActiveTabName()
	if tab == "" {
		return m
	}
	return m.SetSelectedIndexForTab(tab, m.SelectedIndexForTab(tab)+1)
}

func (m Model) MoveSelectionUp() Model {
	tab := m.ActiveTabName()
	if tab == "" {
		return m
	}
	return m.SetSelectedIndexForTab(tab, m.SelectedIndexForTab(tab)-1)
}

func (m Model) SelectedTaskID() (string, bool) {
	if m.ActiveTabName() != "PR Queue" {
		return "", false
	}
	index := m.SelectedIndexForTab("PR Queue")
	if index < 0 || index >= len(m.queueTaskIDs) {
		return "", false
	}
	taskID := strings.TrimSpace(m.queueTaskIDs[index])
	if taskID == "" {
		return "", false
	}
	return taskID, true
}

func (m Model) WithEvent(message string) Model {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return m
	}
	current := append([]string{}, m.sections["Events"]...)
	next := make([]string, 0, len(current)+1)
	next = append(next, trimmed)
	next = append(next, current...)
	m.sections["Events"] = next
	return m
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("ttt UI (preview)\n")
	b.WriteString("commands: tab/backtab (or l/h), 1/2/3, j/k, q\n\n")
	if len(m.tabs) == 0 {
		return b.String()
	}

	safeActiveTab := m.activeTab
	if safeActiveTab < 0 || safeActiveTab >= len(m.tabs) {
		safeActiveTab = 0
	}

	for i, tab := range m.tabs {
		if i == safeActiveTab {
			_, _ = fmt.Fprintf(&b, "[ %s ] ", tab)
		} else {
			_, _ = fmt.Fprintf(&b, "  %s   ", tab)
		}
	}
	b.WriteString("\n\n")

	active := m.tabs[safeActiveTab]
	selected := m.SelectedIndexForTab(active)
	for idx, row := range m.sections[active] {
		prefix := "  "
		if idx == selected {
			prefix = "> "
		}
		b.WriteString(prefix)
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

func normalizeTaskIDs(taskIDs []string, rowCount int) []string {
	result := make([]string, rowCount)
	for idx := 0; idx < rowCount; idx++ {
		if idx < len(taskIDs) {
			result[idx] = strings.TrimSpace(taskIDs[idx])
		}
	}
	return result
}
