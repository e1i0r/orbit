package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/words"
)

// composePlan defines the calculated row positions for all fields in the composer.
type composePlan struct {
	tabLine    int
	url        int // for URL tab
	flow       int
	flowSum    int // first row of the flow detail, -1 when there is none
	flowRows   int // how many rows that detail takes
	id         int // for Manual tab
	textHeader int // for Manual tab
	textBoxTop int // for Manual tab
	textBoxBot int // for Manual tab
	actions    int
}

// composeLayout calculates pure geometric row positions based on active flow and tab.
func (m Model) composeLayout() composePlan {
	p := composePlan{tabLine: 0, flowSum: -1}
	detail := m.flowDetail(m.compose.chosenFlow(), m.width)
	hasSum := len(detail) > 0
	p.flowRows = len(detail)

	if m.compose.tab == composeTabManual {
		p.flow = 2

		cur := 3
		if hasSum {
			p.flowSum = cur
			cur += p.flowRows
		}

		p.id = cur
		cur++
		p.textHeader = cur
		p.textBoxTop = cur + 1
		p.textBoxBot = cur + 5
		p.actions = p.textBoxBot + 2
	} else {
		p.url = 2
		p.flow = 3

		cur := 4
		if hasSum {
			p.flowSum = cur
			cur += p.flowRows
		}

		if m.compose.parsedIssue != nil {
			cur += 2
		}

		p.actions = cur + 1
	}

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
