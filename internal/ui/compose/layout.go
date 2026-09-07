package compose

import (
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/point"
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
func (s State) composeLayout(e Env) composePlan {
	p := composePlan{tabLine: 0, flowSum: -1}
	detail := s.flowDetail(s.chosenFlow(), e.Frame.Body.W, e)
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
	if s.tab == composeTabManual {
		p.id = cur
		cur++
	} else if s.parsedIssue != nil {
		cur += 2
	}

	// The box is one block: the top border carries the label and the paste
	// button, the lines follow, and the bottom border closes it.
	p.boxTop = cur
	p.boxBot = cur + s.composeBoxRowCount(e) + 1
	p.actions = p.boxBot + 2

	return p
}

// hitComposeActions is which button the pointer is over, measured the way
// the row is drawn: two spaces of indent, then each button, three spaces
// apart. See composeRows.
//
// It used to count from zero and fold the indent into the first button's
// width, which put every zone three cells left of what it was pointing at:
// the right-hand end of Save & Run answered cancel. Anything past the last
// button is nothing at all, rather than cancel reaching to the right edge of
// the screen.
func hitComposeActions(p *words.Printer, x int) point.Target {
	at := composeIndent

	for _, b := range []struct {
		text string
		key  string
	}{
		{"[ " + p.T("compose.save_btn", "↵ Save") + " ]", "save"},
		{"[ " + p.T("compose.save_run_btn", "^R Save & Run") + " ]", "save_and_run"},
		{"[ " + p.T("compose.cancel_btn", "esc Cancel") + " ]", "cancel"},
	} {
		wide := lipgloss.Width(b.text)
		if x >= at && x < at+wide {
			return point.Target{Kind: point.ComposeAction, Key: b.key}
		}

		at += wide + composeGap
	}

	return point.Target{}
}

// The row's own spacing, named once so the draw and the hit test cannot
// drift apart.
const (
	composeIndent = 2
	composeGap    = 3
)
