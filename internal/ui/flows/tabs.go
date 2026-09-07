package flows

// The designer's three tabs: the fields, the diagram, and saying it in
// words.
//
// One screen and not three, because they are three views of one thing being
// built: the fields are where it is edited, the diagram is where it is read
// back, and the third is where it starts from a sentence. Everything they
// show comes from the same phases in State — nothing is copied between
// them, so there is no version of the flow that is only true on one tab.

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// The tabs, in the order the strip draws them.
const (
	flowTabFields = iota
	flowTabDiagram
	flowTabSay
	flowTabCount
)

// tabNames is what the strip says, in the reader's language.
func (s State) flowTabNames(e Env) []string {
	p := e.Words

	return []string{
		p.T("flows.tab_fields", "Fields"),
		p.T("flows.tab_diagram", "Diagram"),
		p.T("flows.tab_say", "Describe it"),
	}
}

// tabStrip is the row the tabs are drawn on.
func (s State) flowTabsRow(w int, e Env) builderLine {
	var parts []string

	for i, name := range s.flowTabNames(e) {
		if i == s.tab {
			parts = append(parts, theme.Paint(theme.Sel).Bold(true).Render(" "+name+" "))
			continue
		}

		parts = append(parts, theme.Paint(theme.Dim).Render(" "+name+" "))
	}

	ways := theme.Paint(theme.Dim).Render(e.Words.T("flows.tab_ways", "[^←/^→] tab"))

	return builderLine{
		text:  cells.Fit("  "+strings.Join(parts, " ")+"  "+ways, w),
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
func (s State) flowTabAt(x int, e Env) int {
	at := 2

	for i, name := range s.flowTabNames(e) {
		wide := lipgloss.Width(name) + 2
		if x >= at && x < at+wide {
			return i
		}

		at += wide + 1
	}

	return -1
}

// moveTab is the keyboard's way between them.
func (s State) moveFlowTab(d int) State {
	s.tab = (s.tab + d + flowTabCount) % flowTabCount
	s.scroll = 0

	return s
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
