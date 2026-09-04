package ui

// Folding an entry of the record on the timeline: one of them a row, and
// what a reader wants to read they open.

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// reported is what an engine writes when a phase ends: a paragraph, on a tab
// that has one row an entry. It is the reason this folding exists — six rows
// of it were drawn whether anybody was reading them or not.
const reported = "make check is green: fmt, vet, lint with no issues, every package test, " +
	"and go mod tidy -diff. formatBytes rounds at 1024, caps at GB, and reads 0 B " +
	"for a negative count. The table covers both sides of every unit boundary."

// wordyLog is the fixture record with that paragraph at the end of it.
func wordyLog() []view.Entry {
	return append(fixtureEntries(), view.Entry{
		At: ago(20 * time.Minute), Kind: "phase.finished", Phase: "implement",
		Attempt: 2, PhaseN: 1, Cost: 0.31, Session: "8f2c31", Text: reported,
	})
}

// timeline is the window on the timeline tab, and the rows it draws.
func timeline(t *testing.T, entries []view.Entry) (Model, []string) {
	t.Helper()

	m, _ := openWith(t, "ACME-2662", entries)
	m = showing(t, m, tabTimeline)

	return m, screenRows(m)
}

// TestAnEntryWithMoreToShowSaysSo. The arrow is the whole of the
// affordance, and it is a promise: a row that carries one line already shows
// everything it has, and offering to open it is offering nothing.
func TestAnEntryWithMoreToShowSaysSo(t *testing.T) {
	_, lines := timeline(t, wordyLog())

	if rowOf(lines, foldShut) < 0 {
		t.Errorf("no entry of the timeline offers to open:\n%s", strings.Join(lines, "\n"))
	}

	y := rowOf(lines, "claude opus")
	if y < 0 {
		t.Fatalf("the phase that named its engine was not drawn:\n%s", strings.Join(lines, "\n"))
	}

	if strings.Contains(lines[y], foldShut) || strings.Contains(lines[y], foldOpen) {
		t.Errorf("a row with nothing to open offers an arrow anyway: %q", lines[y])
	}

	if !strings.Contains(lines[y], "claude opus") {
		t.Errorf("the row that names the engine lost it: %q", lines[y])
	}
}

// TestAClosedEntryIsOneRow. Closed is the state a tab full of paragraphs has
// to open in, or the folding buys the reader nothing.
func TestAClosedEntryIsOneRow(t *testing.T) {
	_, lines := timeline(t, wordyLog())

	if tail := "unit boundary"; rowOf(lines, tail) >= 0 {
		t.Errorf("a closed entry set the end of its paragraph anyway:\n%s", strings.Join(lines, "\n"))
	}

	head := rowOf(lines, "make check is green")
	if head < 0 {
		t.Fatalf("a closed entry says nothing at all:\n%s", strings.Join(lines, "\n"))
	}

	if !strings.Contains(lines[head], foldShut) {
		t.Errorf("the row a closed entry was drawn on has no arrow: %q", lines[head])
	}
}

// TestPointingAtAnEntryOpensAndClosesIt, end to end: the row the arrow was
// drawn on is the row the pointer is answered at, and what the click acts on
// is the entry that was under it.
func TestPointingAtAnEntryOpensAndClosesIt(t *testing.T) {
	m, lines := timeline(t, wordyLog())

	y := rowOf(lines, foldShut)
	if y < 0 {
		t.Fatalf("no entry of the timeline offers to open:\n%s", strings.Join(lines, "\n"))
	}

	got := m.hit(30, y)
	if got.Kind != TargetPaneRow {
		t.Fatalf("the row that offers to open answers as %+v, want an entry of the log", got)
	}

	next, _ := m.leftClick(got)
	open := asModel(t, next)
	shown := screenRows(open)

	if rowOf(shown, "unit boundary") < 0 {
		t.Errorf("opening an entry did not put the rest of it on the screen:\n%s", strings.Join(shown, "\n"))
	}

	head := rowOf(shown, "make check is green")
	if head < 0 || !strings.Contains(shown[head], foldOpen) {
		t.Errorf("an open entry is not drawn as open: %q", shown[max(head, 0)])
	}

	shut, _ := open.leftClick(got)
	if again := screenRows(asModel(t, shut)); rowOf(again, "unit boundary") >= 0 {
		t.Errorf("clicking an open entry did not close it:\n%s", strings.Join(again, "\n"))
	}
}

// TestOnlyTheRowTheEntryOpensOnAnswersForIt. The rows an open entry runs on
// belong to the pane: a reader dragging a selection across a paragraph would
// otherwise close the paragraph.
func TestOnlyTheRowTheEntryOpensOnAnswersForIt(t *testing.T) {
	m, lines := timeline(t, wordyLog())

	y := rowOf(lines, foldShut)
	if y < 0 {
		t.Fatalf("no entry of the timeline offers to open:\n%s", strings.Join(lines, "\n"))
	}

	next, _ := m.leftClick(m.hit(30, y))
	open := asModel(t, next)

	tail := rowOf(screenRows(open), "unit boundary")
	if tail < 0 {
		t.Fatal("the entry did not open")
	}

	if got := open.hit(30, tail); got.Kind == TargetPaneRow {
		t.Errorf("a row the paragraph wrapped onto answers as the entry's head: %+v", got)
	}
}

// TestTheEntryUnderThePointerIsTheOneThatOpens. The rows the hit test knows
// are the pane's own and a pointer reports the window's, so an entry that has
// been scrolled to a different row has to answer at the row it is on now.
func TestTheEntryUnderThePointerIsTheOneThatOpens(t *testing.T) {
	m, lines := timeline(t, longWordyLog())

	y := rowOf(lines, foldShut)
	if y < 0 {
		t.Fatalf("no entry of the timeline offers to open:\n%s", strings.Join(lines, "\n"))
	}

	want := m.hit(30, y)
	if want.Kind != TargetPaneRow {
		t.Fatalf("the arrow's row answers as %+v, want an entry of the log", want)
	}

	scrolled := scrolledTo(m, lineAt(m)-3)

	moved := rowOf(screenRows(scrolled), foldShut)
	if moved < 0 {
		t.Fatalf("scrolling took the entry off the screen:\n%s", strings.Join(screenRows(scrolled), "\n"))
	}

	if moved == y {
		t.Fatal("scrolling did not move the entry")
	}

	if got := scrolled.hit(30, moved); got != want {
		t.Errorf("after scrolling, the entry answers as %+v, want %+v", got, want)
	}
}

// longWordyLog is a record too long for one screen with the paragraph a few
// rows above the end of it: far enough down to be on the screen, far enough
// up to still be on it after a scroll.
func longWordyLog() []view.Entry {
	out := append(longLog(), view.Entry{
		At: ago(6 * time.Minute), Kind: "phase.finished", Phase: "implement",
		Attempt: 2, PhaseN: 1, Cost: 0.31, Session: "8f2c31", Text: reported,
	})

	for i := range 5 {
		out = append(out, view.Entry{
			At: ago(time.Duration(5-i) * time.Minute), Kind: "phase.started",
			Phase: "gates", Attempt: 2, PhaseN: 2, Engine: "claude", Model: "opus",
		})
	}

	return out
}

// TestExpandOpensEveryEntryAtOnce. [e] is the gesture for the reader who
// wants the whole record rather than one line of it, and closing it again
// leaves the entries the reader opened by hand open.
func TestExpandOpensEveryEntryAtOnce(t *testing.T) {
	m, lines := timeline(t, wordyLog())

	y := rowOf(lines, foldShut)
	if y < 0 {
		t.Fatalf("no entry of the timeline offers to open:\n%s", strings.Join(lines, "\n"))
	}

	all := step(t, m, "e")
	if rowOf(screenRows(all), "unit boundary") < 0 {
		t.Error("[e] did not open the entries")
	}

	if rowOf(screenRows(all), foldShut) >= 0 {
		t.Error("[e] left an entry drawn as closed")
	}

	next, _ := m.leftClick(m.hit(30, y))
	byHand := step(t, asModel(t, next), "e")

	if rowOf(screenRows(step(t, byHand, "e")), "unit boundary") < 0 {
		t.Error("collapsing everything closed an entry the reader had opened by hand")
	}
}

// TestARowIsWrappedToTheMeasureAndNotCutToIt.
//
// A tool call is written down as the arguments it was made with, and a path
// or a JSON document has no space in it to wrap at. Cutting such a row to
// the measure kept it off the margin the scroll bar is drawn in, and put
// what it cut on no row at all: the fold counts rows, so it was told there
// was nothing to open, and the rest of the command was on no screen the
// reader could reach. The word is broken at the measure instead, and the
// rows that come back are still inside it.
func TestARowIsWrappedToTheMeasureAndNotCutToIt(t *testing.T) {
	long := strings.Repeat("some/deep/path/", 20)

	m, _ := timeline(t, append(fixtureEntries(),
		view.Entry{
			At: ago(19 * time.Minute), Kind: "phase.tool_call", Phase: "implement",
			Attempt: 2, Tool: "Edit", Text: `{"file_path":"` + long + `bytes.go"}`,
		},
		view.Entry{
			At: ago(18 * time.Minute), Kind: "phase.finished", Phase: "implement",
			Attempt: 2, PhaseN: 1, Text: long,
		},
		// A row that folds and whose first line is the unbreakable one: the
		// head a closed entry shows is cut like every other row.
		view.Entry{
			At: ago(17 * time.Minute), Kind: "phase.finished", Phase: "implement",
			Attempt: 2, PhaseN: 1, Text: long + " and the rest of the sentence after it",
		},
	))

	opened := step(t, m, "e")

	// The rows as the pane holds them, and not as the window pads them out
	// and hangs a scroll bar off the end: closed on its head, and open on
	// every row it opens onto.
	for _, lines := range [][]string{m.logLines(), opened.logLines()} {
		for i, l := range lines {
			if !strings.Contains(l, "deep/path") {
				continue
			}

			if w := lipgloss.Width(strings.TrimRight(l, " ")); w > m.frame.Body.W-2 {
				t.Errorf("row %d runs to %d cells on a pane of %d: %q", i, w, m.frame.Body.W, l)
			}
		}
	}

	// Cut, the entry was one row and the fold was told there was nothing
	// under it. Wrapped, it has rows to open onto, and says so.
	closed := m.logLines()

	head := rowOf(closed, "Edit:")
	if head < 0 {
		t.Fatalf("the tool call was not drawn:\n%s", strings.Join(closed, "\n"))
	}

	if !strings.Contains(closed[head], foldShut) {
		t.Errorf("the row that could not be wrapped offers nothing to open: %q", closed[head])
	}

	// And what the wrap broke is all there once the entry is open: the head
	// of the path is on the first row, and the file it ends at is on a row
	// of its own further down.
	whole := strings.Join(strings.Fields(ansi.Strip(strings.Join(opened.logLines(), " "))), " ")
	for _, want := range []string{"bytes.go", "and the rest of the sentence after it"} {
		if !strings.Contains(whole, want) {
			t.Errorf("the timeline lost %q off the end of a row it could not wrap:\n%s",
				want, strings.Join(opened.logLines(), "\n"))
		}
	}
}

// TestOnlyAPaneThatDrawsEntriesFolds. The rows a fold is looked up in are
// the ones the pane was drawn as, so answering with them on a tab that draws
// no entries is answering with a row that is not on the screen.
func TestOnlyAPaneThatDrawsEntriesFolds(t *testing.T) {
	m, _ := openWith(t, "ACME-2662", wordyLog())
	m = showing(t, m, tabTimeline)
	m = showing(t, m, tabOverview)

	lines := screenRows(m)

	for y := m.frame.Body.Y; y < m.frame.Body.Y+m.frame.Body.H-1 && y < len(lines); y++ {
		if got := m.hit(30, y); got.Kind == TargetPaneRow {
			t.Fatalf("row %d of the overview answers as an entry of the log: %+v", y, got)
		}
	}
}

// TestTheTimelineNamesTheRepositoryThatJoined. A task that reached into
// three repositories has three of these rows, and the whole of what a reader
// wants from them is which repositories — a row reading "repository joined"
// over an empty cell is the one fact left out.
func TestTheTimelineNamesTheRepositoryThatJoined(t *testing.T) {
	_, lines := timeline(t, append(fixtureEntries(), view.Entry{
		At: ago(5 * time.Minute), Kind: "repo.joined", Attempt: 2, Repo: "ledger",
	}))

	if rowOf(lines, "ledger") < 0 {
		t.Errorf("the timeline does not say which repository joined:\n%s", strings.Join(lines, "\n"))
	}
}
