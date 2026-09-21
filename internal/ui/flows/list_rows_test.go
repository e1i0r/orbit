package flows

// The list of flows, row by row: what each one says, and which of them is
// on the screen when there are more than there is room for.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/layout"
)

// TestEveryFlowSaysWhatItIsAndWhatItDoes. A row is the flow's name, where
// it came from, what it is for, and a line per phase: what runs it, whether
// it is fed the last output, and whether it stops for a human. Each of
// those is a fact somebody chooses a flow by.
func TestEveryFlowSaysWhatItIsAndWhatItDoes(t *testing.T) {
	dir := t.TempDir()
	writeFlowFile(t, dir, "mine", `{
		"name":"mine",
		"description":"the one I wrote",
		"phases":[
			{"name":"implement","engine":"zeta","model":"one","prompt":"write it"},
			{"name":"review","engine":"zeta","wait":true,"feed_output":true}
		]}`)

	e := world(t)
	e.Flows = flowsTestDir(dir)

	s := Open(FromBoard, e)

	var said strings.Builder

	for _, l := range s.flowsListLines(160, e) {
		said.WriteString(ansi.Strip(l.text) + "\n")
	}

	for _, want := range []string{
		"mine",               // the name
		"the one I wrote",    // what it is for
		"1. implement",       // a line per phase, numbered from one
		"zeta / one",         // what runs it
		"write it",           // and what it was told
		"2. review",          //
		"[feeds input]",      // that it is handed the last output
		"stops for human",    // and that it waits
		"runs automatically", // where it does not
	} {
		if !strings.Contains(said.String(), want) {
			t.Errorf("the list does not say %q:\n%s", want, said.String())
		}
	}
}

// TestAFlowThatWillNotParseSaysWhy. "There is a file called that" is what
// the reader is asking, so the row is drawn and it says what is wrong with
// it rather than the flow going missing.
func TestAFlowThatWillNotParseSaysWhy(t *testing.T) {
	dir := t.TempDir()
	writeFlowFile(t, dir, "broken", `{"name":"broken","phases":[`)

	e := world(t)
	e.Flows = flowsTestDir(dir)

	s := Open(FromBoard, e)

	var said strings.Builder

	for _, l := range s.flowsListLines(160, e) {
		said.WriteString(ansi.Strip(l.text) + "\n")
	}

	if !strings.Contains(said.String(), "broken") {
		t.Errorf("a flow that will not parse is missing from the list:\n%s", said.String())
	}

	// Something about what is wrong, and no phases: there are none to read.
	if !strings.Contains(said.String(), "broken\n") && !strings.Contains(said.String(), "broken ") {
		t.Errorf("the broken flow's row says nothing:\n%s", said.String())
	}
}

// TestTheFlowTheCursorIsOnIsOnTheScreen, at every height and on every row —
// including the create button, which is what "nothing chosen" means here.
func TestTheFlowTheCursorIsOnIsOnTheScreen(t *testing.T) {
	for _, h := range []int{8, 12, 20, 34, 60} {
		e := world(t)

		frame, err := layout.Fit(110, h+10)
		if err != nil {
			continue
		}

		e.Frame = frame

		s := Open(FromBoard, e)

		for sel := -1; sel < len(s.listed); sel++ {
			at := s
			at.sel = sel

			rows := at.View(e.Frame.Body.H, e.Frame.Body.W, e)
			if len(rows) != e.Frame.Body.H {
				t.Fatalf("a body of %d rows drew %d", e.Frame.Body.H, len(rows))
			}

			drawn := ansi.Strip(strings.Join(rows, "\n"))

			want := "Create Custom Flow"
			if sel >= 0 {
				want = at.listed[sel].Name
			}

			if !strings.Contains(drawn, want) {
				t.Errorf("at %d rows with %d chosen, %q is not on the screen:\n%s",
					e.Frame.Body.H, sel, want, drawn)
			}

			// The way out is the last row, whatever the list is doing.
			if last := ansi.Strip(rows[len(rows)-1]); !strings.Contains(last, "back") {
				t.Errorf("at %d rows the last line is %q, want the ways out",
					e.Frame.Body.H, last)
			}
		}
	}
}

// TestTheRailIsThereOnlyWhenThereIsMoreThanFits, and it says where in the
// list the page is.
func TestTheRailIsThereOnlyWhenThereIsMoreThanFits(t *testing.T) {
	e := world(t)

	s := Open(FromBoard, e)
	lines := s.flowsListLines(e.Frame.Body.W, e)

	// Taller than the list: no rail.
	tall := len(lines) + 4

	roomy := ansi.Strip(strings.Join(s.View(tall, e.Frame.Body.W, e), ""))
	if strings.Contains(roomy, "┃") {
		t.Error("a list with room to spare draws a rail beside it")
	}

	// Shorter: a rail, and it moves with the page.
	short := 10

	top := ansi.Strip(strings.Join(s.View(short, e.Frame.Body.W, e), "\n"))
	if !strings.Contains(top, "┃") && !strings.Contains(top, "│") {
		t.Errorf("a list longer than the screen draws no rail:\n%s", top)
	}
}
