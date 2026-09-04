package ui

// Folding a block of the thinking pane: the same gesture the timeline makes,
// on the tab that reads the record as reasoning. A phase's thinking is
// paragraphs, and a pane that sets every one of them at once is a pane
// nobody finds the phase they were looking for on.

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// thinking is the window on the thinking tab, and the rows it draws.
func thinking(t *testing.T, entries []view.Entry) (Model, []string) {
	t.Helper()

	m, _ := openWith(t, "ACME-2662", entries)
	m = showing(t, m, tabThinking)

	return m, screenRows(m)
}

// TestAThoughtWithMoreToShowSaysSo. The same promise the timeline makes, on
// the pane that reads the record as reasoning: an arrow is offered only
// where opening one puts something new on the screen, and a block that has
// nothing to open still begins in the column the others do.
func TestAThoughtWithMoreToShowSaysSo(t *testing.T) {
	_, lines := thinking(t, fixtureEntries())

	first := rowOf(lines, "wrote retry.go")
	if first < 1 {
		t.Fatalf("the reasoning of the first phase is not on the pane:\n%s", strings.Join(lines, "\n"))
	}

	if !strings.Contains(lines[first-1], foldShut) {
		t.Errorf("a block with more to say does not offer to open: %q", lines[first-1])
	}

	if rowOf(lines, "added a backoff") >= 0 {
		t.Errorf("a closed block said the rest of it anyway:\n%s", strings.Join(lines, "\n"))
	}

	solo := rowOf(lines, "unreachable code")
	if solo < 1 {
		t.Fatalf("the reasoning of the failed phase is not on the pane:\n%s", strings.Join(lines, "\n"))
	}

	if strings.Contains(lines[solo-1], foldShut) || strings.Contains(lines[solo-1], foldOpen) {
		t.Errorf("a block with nothing to open offers an arrow anyway: %q", lines[solo-1])
	}

	// In cells and not in bytes: an arrow is one column and three of them.
	bullet := func(l string) int {
		head, _, _ := strings.Cut(l, "●")
		return lipgloss.Width(head)
	}

	if bullet(lines[first-1]) != bullet(lines[solo-1]) {
		t.Errorf("the two blocks begin in different columns:\n%q\n%q", lines[first-1], lines[solo-1])
	}
}

// TestPointingAtAThoughtOpensAndClosesIt, end to end on the thinking pane.
func TestPointingAtAThoughtOpensAndClosesIt(t *testing.T) {
	m, lines := thinking(t, fixtureEntries())

	y := rowOf(lines, foldShut)
	if y < 0 {
		t.Fatalf("no block of the thinking pane offers to open:\n%s", strings.Join(lines, "\n"))
	}

	at := m.hit(30, y)
	if at.Kind != TargetPaneRow {
		t.Fatalf("the row that offers to open answers as %+v, want an entry of the record", at)
	}

	shown := screenRows(clicked(t, m, at))
	if rowOf(shown, "added a backoff") < 0 {
		t.Errorf("opening block did not put rest on screen:\n%s", strings.Join(shown, "\n"))
	}

	head := rowOf(shown, "wrote retry.go") - 1
	if head < 0 || !strings.Contains(shown[head], foldOpen) {
		t.Errorf("an open block is not drawn as open: %q", shown[max(head, 0)])
	}

	again := screenRows(clicked(t, clicked(t, m, at), at))
	if rowOf(again, "added a backoff") >= 0 {
		t.Errorf("clicking an open block did not close it:\n%s", strings.Join(again, "\n"))
	}
}

// TestAThoughtIsWrappedAndCutToThePaneItIsDrawnOn. The reasoning pane is
// where an engine's longest sentences land: a paragraph set on one row would
// lose everything past the margin, and a path with nothing to wrap at would
// be drawn over the margin the scroll bar lives in.
func TestAThoughtIsWrappedAndCutToThePaneItIsDrawnOn(t *testing.T) {
	unbreakable := strings.Repeat("some/deep/path/", 20)

	m, _ := thinking(t, []view.Entry{
		{
			At: ago(20 * time.Minute), Kind: "phase.finished", Phase: "implement",
			Attempt: 1, PhaseN: 1, Text: reported,
		},
		{
			At: ago(18 * time.Minute), Kind: "phase.finished", Phase: "gates",
			Attempt: 1, PhaseN: 2, Text: unbreakable,
		},
	})

	open := screenRows(step(t, m, "e"))

	head, tail := rowOf(open, "make check is green"), rowOf(open, "unit boundary")
	if head < 0 || tail < 0 {
		t.Fatalf("an open block did not set the whole of its paragraph:\n%s", strings.Join(open, "\n"))
	}

	if head == tail {
		t.Error("a paragraph was set on one row rather than wrapped to the pane")
	}

	for _, lines := range [][]string{screenRows(m), open} {
		for i, l := range lines {
			if !strings.Contains(l, "deep/path") && !strings.Contains(l, "make check") {
				continue
			}

			// The row as the pane holds it, not as the window pads it out or
			// tracks scroll on the right margin.
			clean := strings.TrimRight(strings.TrimRight(l, scrollRail+scrollThumb), " ")
			if w := lipgloss.Width(clean); w > m.frame.Body.W-2 {
				t.Errorf("row %d runs to %d cells on a pane of %d: %q", i, w, m.frame.Body.W, l)
			}
		}
	}
}
