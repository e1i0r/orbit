package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/words"
)

// composePlan defines the calculated row positions for all fields in the composer.
type composePlan struct {
	tabLine  int
	flow     int
	flowSum  int // first row of the flow detail, -1 when there is none
	flowRows int // how many rows that detail takes
	id       int // for Manual tab
	boxTop   int // the label, the top border and the paste button share a row
	boxBot   int // the bottom border of it
	actions  int
}

// composeLayout calculates pure geometric row positions based on active flow and tab.
func (m Model) composeLayout() composePlan {
	p := composePlan{tabLine: 0, flowSum: -1}
	detail := m.flowDetail(m.compose.chosenFlow(), m.width)
	hasSum := len(detail) > 0
	p.flowRows = len(detail)

	p.flow = 2

	cur := 3
	if hasSum {
		p.flowSum = cur
		cur += p.flowRows
	}

	// Both tabs end in a box — the task on one, the URL on the other — and
	// what is above it is what differs: the id on the manual tab, and on
	// the other the preview of the issue a URL was recognised as.
	if m.compose.tab == composeTabManual {
		p.id = cur
		cur++
	} else if m.compose.parsedIssue != nil {
		cur += 2
	}

	// The box is one block: the top border carries the label and the paste
	// button, the lines follow, and the bottom border closes it.
	p.boxTop = cur
	p.boxBot = cur + m.composeBoxRowCount() + 1
	p.actions = p.boxBot + 2

	return p
}

// hitComposeActions calculates button hitboxes based on translated button widths.
func hitComposeActions(p *words.Printer, x int) Target {
	saveText := "[ " + p.T("compose.save_btn", "↵ Save") + " ]"
	runText := "[ " + p.T("compose.save_run_btn", "^R Save & Run") + " ]"

	saveW := lipgloss.Width(saveText) + 2
	runW := lipgloss.Width(runText) + 3

	if x < saveW {
		return Target{Kind: TargetComposeAction, Key: "save"}
	} else if x < saveW+runW {
		return Target{Kind: TargetComposeAction, Key: "save_and_run"}
	}

	return Target{Kind: TargetComposeAction, Key: "cancel"}
}
