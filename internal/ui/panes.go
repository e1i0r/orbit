package ui

// The eleven detail panes: overview, flow, gates, cost, refused, timeline,
// report, artifacts, notes, diff, and thinking.

// tab is which of the eleven panes is showing.
type tab int

const (
	tabOverview tab = iota
	tabFlow
	tabGates
	tabCost
	tabRefused
	tabTimeline
	tabReport
	tabArtifacts
	tabNotes
	tabDiff
	tabThinking
	tabCount
)

const (
	tabLog      = tabTimeline
	tabEvidence = tabReport
)

// paneKey returns the single keystroke that opens tab t directly.
//
// One function answers for every site — keyboard dispatch, mouse routing,
// tab strip drawing, and the help overlay — so the eleven keys (1-9, 0, w)
// never drift apart across the UI.
func paneKey(t tab) string {
	switch t {
	case tabOverview:
		return "1"
	case tabFlow:
		return "2"
	case tabGates:
		return "3"
	case tabCost:
		return "4"
	case tabRefused:
		return "5"
	case tabTimeline:
		return "6"
	case tabReport:
		return "7"
	case tabArtifacts:
		return "8"
	case tabNotes:
		return "9"
	case tabDiff:
		return "0"
	case tabThinking:
		return "w"
	default:
		return ""
	}
}

// keyToPane translates a single typed keystroke to its corresponding pane tab.
func keyToPane(k string) (tab, bool) {
	switch k {
	case "1":
		return tabOverview, true
	case "2":
		return tabFlow, true
	case "3":
		return tabGates, true
	case "4":
		return tabCost, true
	case "5":
		return tabRefused, true
	case "6":
		return tabTimeline, true
	case "7":
		return tabReport, true
	case "8":
		return tabArtifacts, true
	case "9":
		return tabNotes, true
	case "0":
		return tabDiff, true
	case "w", "W":
		return tabThinking, true
	default:
		return 0, false
	}
}

// tabName is one tab and what it is called in the reader's language.
type tabName struct {
	tab  tab
	text string
}

// tabNames returns the eleven tabs in order.
func (m Model) tabNames() []tabName {
	p := m.opts.Words

	return []tabName{
		{tabOverview, p.T("tab.overview", "overview")},
		{tabFlow, p.T("tab.flow", "flow")},
		{tabGates, p.T("tab.gates", "gates")},
		{tabCost, p.T("tab.cost", "cost")},
		{tabRefused, p.T("tab.refused", "refused")},
		{tabTimeline, p.T("tab.timeline", "timeline")},
		{tabReport, p.T("tab.report", "report")},
		{tabArtifacts, p.T("tab.artifacts", "artifacts")},
		{tabNotes, p.T("tab.notes", "notes")},
		{tabDiff, p.T("tab.diff", "diff")},
		{tabThinking, p.T("tab.thinking", "thinking")},
	}
}

// syncPanes rebuilds all eleven panes and resizes them to the detail body region.
func (m Model) syncPanes() Model {
	w := max(m.frame.Body.W, 1)

	timeline, timelineHeads, timelineSeams := m.logRows()
	report, reportSeams := m.reportRows()
	thinking, thinkingHeads := m.thinkingRows()
	flowTree, flowHeads := m.flowRows()
	gates, gateHeads := m.gatesRows()
	refused, refusedHeads := m.refusedRows()
	notes, noteHeads := m.notesRows()
	diff, diffHeads := m.diffRows()
	artifacts, artifactHeads := m.artifactsRows()

	m.heads[tabTimeline], m.heads[tabThinking] = timelineHeads, thinkingHeads
	m.heads[tabFlow], m.heads[tabGates] = flowHeads, gateHeads
	m.heads[tabRefused], m.heads[tabNotes] = refusedHeads, noteHeads
	m.heads[tabDiff], m.heads[tabArtifacts] = diffHeads, artifactHeads
	m.seams[tabTimeline], m.seams[tabReport] = timelineSeams, reportSeams

	content := [tabCount][]string{
		tabOverview:  m.overviewLines(),
		tabFlow:      flowTree,
		tabGates:     gates,
		tabCost:      m.costLines(),
		tabRefused:   refused,
		tabTimeline:  timeline,
		tabReport:    report,
		tabArtifacts: artifacts,
		tabNotes:     notes,
		tabDiff:      diff,
		tabThinking:  thinking,
	}
	for i := range m.panes {
		// Sized to the rows it is drawn into, so that the last page of a
		// pane is a page the reader can reach and the scroll bar's floor
		// is the end of the document.
		_, h := m.paneBandFor(tab(i))

		vp := m.panes[i]
		vp.SoftWrap = false
		vp.SetWidth(w)
		vp.SetHeight(max(h, 1))
		vp.SetContentLines(content[i])
		m.panes[i] = vp
	}

	if m.following {
		vp := m.panes[tabTimeline]
		vp.GotoBottom()
		m.panes[tabTimeline] = vp
	}

	return m
}

// tabMenuEntries is the menu of a task's panes: the same list the tab strip
// draws, with a sentence under each name saying what is in it. It is here
// rather than in menu.go because it is built out of this file's two answers
// — which panes this task has, and which key each one is on.
func (m Model) tabMenuEntries() []menuEntry {
	p := m.opts.Words
	descs := map[tab]string{
		tabOverview:  p.T("tab_desc.overview", "general status, live activity and metrics summary"),
		tabFlow:      p.T("tab_desc.flow", "task pipeline phases and execution plan"),
		tabGates:     p.T("tab_desc.gates", "automated quality gates, linters and validation status"),
		tabCost:      p.T("tab_desc.cost", "token usage and monetary cost breakdown per phase"),
		tabRefused:   p.T("tab_desc.refused", "denied tool invocations and permission rejections"),
		tabTimeline:  p.T("tab_desc.timeline", "complete live event timeline and phase history"),
		tabReport:    p.T("tab_desc.report", "final solution report, summary and review conclusions"),
		tabArtifacts: p.T("tab_desc.artifacts", "raw tool output and generated artifacts"),
		tabNotes:     p.T("tab_desc.notes", "operator notes and interactive dialogue history"),
		tabDiff:      p.T("tab_desc.diff", "git working tree diff and code modifications"),
		tabThinking:  p.T("tab_desc.thinking", "extended model thinking, chain of thought and reasoning"),
	}

	var out []menuEntry

	for _, n := range m.tabNames() {
		tVal := n.tab
		k := paneKey(tVal)
		out = append(out, menuEntry{
			glyph:  "[" + k + "]",
			title:  n.text,
			detail: descs[tVal],
			tab:    &tVal,
		})
	}

	return out
}
