package ui

// The designer's three tabs: the fields, the diagram, and saying it in
// words.
//
// One screen and not three, because they are three views of one thing being
// built: the fields are where it is edited, the diagram is where it is read
// back, and the third is where it starts from a sentence. Everything they
// show comes from the same phases in flowsState — nothing is copied between
// them, so there is no version of the flow that is only true on one tab.

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// The tabs, in the order the strip draws them.
const (
	flowTabFields = iota
	flowTabDiagram
	flowTabSay
	flowTabCount
)

// tabNames is what the strip says, in the reader's language.
func (m Model) flowTabNames() []string {
	p := m.opts.Words

	return []string{
		p.T("flows.tab_fields", "Fields"),
		p.T("flows.tab_diagram", "Diagram"),
		p.T("flows.tab_say", "Describe it"),
	}
}

// tabStrip is the row the tabs are drawn on.
func (m Model) flowTabsRow(w int) builderLine {
	var parts []string

	for i, name := range m.flowTabNames() {
		if i == m.flows.tab {
			parts = append(parts, Paint(Sel).Bold(true).Render(" "+name+" "))
			continue
		}

		parts = append(parts, Paint(Dim).Render(" "+name+" "))
	}

	return builderLine{
		text:  fit("  "+strings.Join(parts, " ")+"  "+Paint(Dim).Render(m.opts.Words.T("flows.tab_ways", "[^←/^→] tab")), w),
		field: noField,
		phase: noPhase,
		pick:  noPick,
		strip: true,
	}
}

// tabAt is which tab the pointer is over, or -1 for the hint beside them.
//
// The widths are measured off the names rather than written down, because a
// translation makes every one of them a different width.
func (m Model) flowTabAt(x int) int {
	at := 2

	for i, name := range m.flowTabNames() {
		wide := lipgloss.Width(name) + 2
		if x >= at && x < at+wide {
			return i
		}

		at += wide + 1
	}

	return -1
}

// moveTab is the keyboard's way between them.
func (m Model) moveFlowTab(d int) Model {
	m.flows.tab = (m.flows.tab + d + flowTabCount) % flowTabCount
	m.flows.scroll = 0

	return m
}

// tabKey is the keys that belong to the strip rather than to what is under
// it: ctrl and an arrow, which no field on any tab uses.
func flowTabKey(msg tea.KeyPressMsg) (int, bool) {
	if msg.Mod&tea.ModCtrl == 0 {
		return 0, false
	}

	switch msg.Code {
	case tea.KeyLeft:
		return -1, true
	case tea.KeyRight:
		return 1, true
	}

	return 0, false
}
