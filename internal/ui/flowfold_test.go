package ui

// Folding a phase of the flow: the tree is the shape of a run, and a reader
// who wants the shape does not want every dial every phase was set with
// hanging under it. The tree stays a tree either way — the branches are what
// says which phase a thing belongs to.

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// flowTree is the window on the flow tab, and the rows it draws.
func flowTree(t *testing.T, entries []view.Entry) (Model, []string) {
	t.Helper()

	m, _ := openWith(t, "ACME-2662", entries)
	m = showing(t, m, tabFlow)

	return m, screenRows(m)
}

// node is the row one phase of the tree was drawn on, by the count it
// carries.
func node(t *testing.T, lines []string, n string) int {
	t.Helper()

	y := rowOf(lines, n)
	if y < 0 {
		t.Fatalf("phase %s is not on the tree:\n%s", n, strings.Join(lines, "\n"))
	}

	return y
}

// TestEveryPhaseOfTheTreeOffersToOpen. A phase always names an engine —
// flow.Validate refuses a flow whose phases do not — so there is always
// something under a node, and every node says so and hangs off its branch
// while saying it.
func TestEveryPhaseOfTheTreeOffersToOpen(t *testing.T) {
	_, lines := flowTree(t, fixtureEntries())

	// The branch each node hangs off is the tree: the last phase closes it
	// and the ones above it carry the trunk on down to that one.
	for n, branch := range map[string]string{"[1/3]": "├──", "[2/3]": "├──", "[3/3]": "└──"} {
		y := node(t, lines, n)

		if !strings.Contains(lines[y], foldShut) {
			t.Errorf("the node of %s does not offer to open: %q", n, lines[y])
		}

		if !strings.Contains(lines[y], branch) {
			t.Errorf("the node of %s does not hang off %s: %q", n, branch, lines[y])
		}
	}

	if rowOf(lines, "⚙️") >= 0 {
		t.Errorf("a tree nobody has opened says how a phase was set up anyway:\n%s", strings.Join(lines, "\n"))
	}
}

// TestPointingAtAPhaseOpensAndClosesIt, end to end on the flow tab.
func TestPointingAtAPhaseOpensAndClosesIt(t *testing.T) {
	m, lines := flowTree(t, fixtureEntries())

	at := m.hit(30, node(t, lines, "[1/3]"))
	if at.Kind != TargetPaneRow || at.Pane != 0 {
		t.Fatalf("the first node answers as %+v, want the first phase of the flow", at)
	}

	open := screenRows(clicked(t, m, at))
	if rowOf(open, "⚙️") < 0 {
		t.Errorf("opening a phase did not say how it was set up:\n%s", strings.Join(open, "\n"))
	}

	if y := node(t, open, "[1/3]"); !strings.Contains(open[y], foldOpen) {
		t.Errorf("an open node is not drawn as open: %q", open[y])
	}

	if y := node(t, open, "[2/3]"); !strings.Contains(open[y], foldShut) {
		t.Errorf("opening one phase opened the one below it: %q", open[y])
	}

	if y := node(t, open, "[2/3]"); !strings.Contains(open[y], "├──") {
		t.Errorf("the tree lost its branches when a phase was opened: %q", open[y])
	}

	again := screenRows(clicked(t, clicked(t, m, at), at))
	if rowOf(again, "⚙️") >= 0 {
		t.Errorf("clicking an open phase did not close it:\n%s", strings.Join(again, "\n"))
	}
}

// TestOnlyTheNodeRowAnswersForThePhase. The trunk carries on past a node
// whether it is open or shut, and it is not the node: a reader who points at
// the line between two phases has pointed at neither.
func TestOnlyTheNodeRowAnswersForThePhase(t *testing.T) {
	m, lines := flowTree(t, fixtureEntries())

	if at := m.hit(30, node(t, lines, "[1/3]")+1); at.Kind != TargetPaneBody {
		t.Errorf("the trunk below a node answers as %+v, want the pane itself", at)
	}
}

// TestEachPaneRemembersItsOwnOpenRows. The tree counts phases and the
// timeline counts entries of the record, so the first row of one is not the
// first row of the other and opening either must not open both.
func TestEachPaneRemembersItsOwnOpenRows(t *testing.T) {
	m, lines := flowTree(t, oneWordyEntry(reported))

	m = clicked(t, m, m.hit(30, node(t, lines, "[1/3]")))
	if rowOf(screenRows(m), "⚙️") < 0 {
		t.Fatal("the first phase of the tree did not open")
	}

	rows := screenRows(showing(t, m, tabTimeline))
	if rowOf(rows, "unit boundary") >= 0 {
		t.Errorf("opening the first phase of the tree opened the first entry of the timeline:\n%s",
			strings.Join(rows, "\n"))
	}
}

// TestAnOutcomeIsWrappedAndCutToTheTree. What a phase wrote is the one thing
// under a node that came from outside this program: a paragraph of it set on
// one row would lose everything past the margin, and a path with nothing to
// break at would run off the pane even wrapped.
func TestAnOutcomeIsWrappedAndCutToTheTree(t *testing.T) {
	// An unbreakable run at either end of the paragraph, so that the row the
	// label is on and the rows under it are both rows that have to be cut,
	// and a blank line between them of the kind an engine leaves behind.
	run := strings.Repeat("x", 200)
	m, lines := flowTree(t, oneWordyEntry(run+"\n\n"+reported+"\n \n"+run))

	m = clicked(t, m, m.hit(30, node(t, lines, "[1/3]")))
	rows := screenRows(m)

	head := rowOf(rows, "make check is green")
	if head < 0 {
		t.Fatalf("the outcome of the phase is not on the tree:\n%s", strings.Join(rows, "\n"))
	}

	if strings.Contains(rows[head], "unit boundary") {
		t.Errorf("the outcome was set on one row instead of wrapped to the tree: %q", rows[head])
	}

	if rowOf(rows, "unit boundary") < 0 {
		t.Errorf("wrapping the outcome lost the end of it:\n%s", strings.Join(rows, "\n"))
	}

	// Measured on the pane's own rows and not on the screen: the frame cuts
	// what it is handed to the width it has, so a row too wide for the pane
	// looks exactly like one that fits by the time it is drawn.
	for i, l := range m.flowLines() {
		plain := strings.TrimRight(ansi.Strip(l), " ")

		if w := lipgloss.Width(plain); w > m.frame.Body.W {
			t.Errorf("row %d of the tree runs to %d cells on a pane of %d: %q", i, w, m.frame.Body.W, plain)
		}

		// A blank line in what a phase wrote is not a row of the tree: drawn
		// as one it is a branch hanging off nothing.
		if strings.HasSuffix(plain, "──") {
			t.Errorf("row %d of the tree is a branch with nothing on it: %q", i, plain)
		}
	}
}

// oneWordyEntry is a record of one phase that finished with something long
// to say, so that the first row of the timeline and the first node of the
// tree are both rows that fold.
func oneWordyEntry(text string) []view.Entry {
	return []view.Entry{{
		At: ago(20 * time.Minute), Kind: "phase.finished", Phase: "implement",
		Attempt: 1, PhaseN: 1, Text: text,
	}}
}

// TestTheTailOfAnOutcomeHangsOffTheRowItContinues. A branch in front of a row
// says a further thing is under this phase. The second row of a wrapped
// paragraph is not a further thing, it is the rest of the first one, and the
// branch that closes the node is the one under the last thing there is.
func TestTheTailOfAnOutcomeHangsOffTheRowItContinues(t *testing.T) {
	run := strings.Repeat("x", 200)
	m, lines := flowTree(t, oneWordyEntry(run+" "+reported))

	m = clicked(t, m, m.hit(30, node(t, lines, "[1/3]")))
	rows := m.flowLines()

	label := rowOf(rows, "📋")
	if label < 0 || label+1 >= len(rows) {
		t.Fatalf("the outcome of the phase has no tail on the tree:\n%s", strings.Join(rows, "\n"))
	}

	if tail := ansi.Strip(rows[label+1]); strings.Contains(tail, "──") {
		t.Errorf("the rest of the outcome hangs off a branch of its own: %q", tail)
	}

	// The node still closes, and on the row that starts the last thing under
	// it rather than on whatever that thing's last row happens to be.
	if got := ansi.Strip(rows[label]); !strings.Contains(got, "└──") {
		t.Errorf("the last thing under the phase does not close its branch: %q", got)
	}
}
