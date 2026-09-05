package ui

// What the supervisor screen redraws, and what it does not. The scrolling
// itself is supervisorscroll_test.go.

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// TestTheThreadIsRenderedOncePerChangeAndNotOncePerFrame is the cost of
// having the screen open.
//
// Every message is rendered to draw the two dozen rows that fit, and Bubble
// Tea draws on every message it delivers: the half-second board tick, the
// spinner ten times a second, and every key pressed while typing an answer.
// A thread of forty-three messages rendered to a thousand rows a frame, at
// twenty-one times the cost of the board behind it.
func TestTheThreadIsRenderedOncePerChangeAndNotOncePerFrame(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Reader = &threadReader{lines: longThread(40)}
	m = m.openSupervisor()

	first, _ := m.threadLines(96)
	if len(first) == 0 {
		t.Fatal("the thread rendered to nothing")
	}

	second, _ := m.threadLines(96)
	if &first[0] != &second[0] {
		t.Error("the thread was rendered twice for one frame: a redraw that changes " +
			"nothing must give back the rows it already has")
	}
}

// TestAnArrivingMessageIsDrawn is the other half of it: rows kept for a
// thread that has since changed are the wrong rows.
func TestAnArrivingMessageIsDrawn(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	r := &threadReader{lines: longThread(4)}
	m.opts.Reader = r
	m = m.openSupervisor()

	before, _ := m.threadLines(96)

	r.lines = longThread(8)
	m = m.syncSupervisor()

	after, _ := m.threadLines(96)
	if len(after) <= len(before) {
		t.Errorf("the thread drew %d rows once four more messages had arrived, "+
			"and %d before them", len(after), len(before))
	}
}

// TestANarrowerScreenRendersTheThreadAgain: the rows are wrapped to a width,
// so they belong to the width they were wrapped to and to no other.
func TestANarrowerScreenRendersTheThreadAgain(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Reader = &threadReader{lines: wordyThread(6)}
	m = m.openSupervisor()

	wide, _ := m.threadLines(96)
	narrow, _ := m.threadLines(40)

	if slices.Equal(wide, narrow) {
		t.Error("the rows wrapped for a 96 column screen were drawn again on a 40 column one")
	}
}

// wordyThread is a thread whose turns are long enough to be wrapped, which
// is what makes the width they were wrapped to visible in the rows.
func wordyThread(n int) []view.SupervisorLine {
	at := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)

	lines := make([]view.SupervisorLine, 0, n)
	for i := range n {
		lines = append(lines, view.SupervisorLine{
			At: at.Add(time.Duration(i) * time.Minute), By: "operator", Channel: "tui",
			Text: fmt.Sprintf("turn %02d: the webhook retries on 5xx and gives up after "+
				"the fifth attempt, which is what the gate is waiting on", i),
		})
	}

	return lines
}
