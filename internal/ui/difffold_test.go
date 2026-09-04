package ui

// Folding a file of the diff. Every other head in this window is an arrow the
// pointer can go for; the diff had a word, a key, and no way in with a mouse.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/words"
)

// twoFiles is a diff of two files, so that folding one can be seen to leave
// the other alone.
const twoFiles = `diff --git a/one.go b/one.go
--- a/one.go
+++ b/one.go
@@ -1,2 +1,3 @@
 package one
+func first() {}
diff --git a/two.go b/two.go
--- a/two.go
+++ b/two.go
@@ -1,2 +1,3 @@
 package two
+func second() {}
`

// onDiff is the window on the diff tab over a diff a test wrote, and the rows
// of the screen it draws — which is what a pointer is aimed at.
func onDiff(t *testing.T, text string) (Model, []string) {
	t.Helper()

	m, _ := openIn(t, words.For("en"), "ACME-2662", nil, text)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 36})
	m = asModel(t, next)
	m = showing(t, m, tabDiff)

	return m, screenRows(m)
}

// cardRow is the row the file's own card opens on. The bar above the pane names
// the file the reader is on as well, and it is not the row that folds.
func cardRow(t *testing.T, lines []string, name string) int {
	t.Helper()

	for i, l := range lines {
		if strings.Contains(l, "┌── ") && strings.Contains(l, name) {
			return i
		}
	}

	t.Fatalf("%s has no card on the diff:\n%s", name, strings.Join(lines, "\n"))

	return -1
}

// TestAFileOfTheDiffWearsTheArrowEveryOtherHeadWears. A reader who has learned
// one gesture on the timeline should not have to learn a second one here.
func TestAFileOfTheDiffWearsTheArrowEveryOtherHeadWears(t *testing.T) {
	m, lines := onDiff(t, twoFiles)

	head := cardRow(t, lines, "one.go")

	// The card closes on the pane it is drawn in. A corner pushed past the
	// right edge is cut off it, and the box a reader is reading has no side.
	if !strings.HasSuffix(strings.TrimRight(ansi.Strip(lines[head]), " "), "──┐") {
		t.Errorf("the card does not close inside the pane: %q", ansi.Strip(lines[head]))
	}

	if !strings.Contains(lines[head], foldOpen) {
		t.Errorf("an open file does not say it is open: %q", ansi.Strip(lines[head]))
	}

	// The pointer is offered the row the arrow is on, and it is offered the
	// first file rather than whichever one the arithmetic landed on.
	at := m.hit(30, head)
	if at.Kind != TargetPaneRow || at.Pane != 0 {
		t.Fatalf("pointing at the file = %+v, want file 0 of the diff", at)
	}

	shut := clicked(t, m, at)

	rows := shut.diffLines()
	if y := cardRow(t, rows, "one.go"); !strings.Contains(rows[y], foldShut) {
		t.Errorf("clicking the file did not close it:\n%s", strings.Join(rows, "\n"))
	}

	if rowOf(rows, "func first") >= 0 {
		t.Errorf("the file that was closed is still showing its lines:\n%s", strings.Join(rows, "\n"))
	}

	// The other file is untouched: a fold is one file's, and a gesture that
	// took the whole diff with it would be the [a] key by accident.
	if rowOf(rows, "func second") < 0 {
		t.Errorf("closing one file closed the next one too:\n%s", strings.Join(rows, "\n"))
	}
}

// TestClosingAFileLeavesTheWindowItWasClosedOnAlone. A Model is copied by
// every method that returns one and a map is not, so a collapse written into
// the map in place would reach the window the reader is still looking at.
func TestClosingAFileLeavesTheWindowItWasClosedOnAlone(t *testing.T) {
	m, lines := onDiff(t, twoFiles)

	// One file is closed first, so that there is a map to leak through: a
	// window nobody has folded anything in has none, and a nil map is
	// replaced rather than written to whichever way this is done.
	m = clicked(t, m, m.hit(30, cardRow(t, lines, "one.go")))
	lines = screenRows(m)

	before := strings.Join(m.diffLines(), "\n")

	_ = clicked(t, m, m.hit(30, cardRow(t, lines, "two.go")))

	if after := strings.Join(m.diffLines(), "\n"); after != before {
		t.Errorf("closing a file on one window closed it on the window it came from:\n%s", after)
	}
}

// TestTheFileTheReaderIsOnIsTheFileTheKeyFolds. The pointer names a file and
// the key names none, so the key takes whichever one the pane is scrolled to.
func TestTheFileTheReaderIsOnIsTheFileTheKeyFolds(t *testing.T) {
	m, _ := onDiff(t, twoFiles)

	m = m.toggleCollapseCurrentFile()
	if !m.collapsedFiles["one.go"] {
		t.Errorf("the file at the top of the pane was not the one folded: %v", m.collapsedFiles)
	}

	if m.collapsedFiles["two.go"] {
		t.Errorf("folding the file at the top took the next one with it: %v", m.collapsedFiles)
	}

	// All of them, and then none: the reader who folds every file is asking
	// for the list of what changed, and asking twice is asking to go back.
	m = m.toggleCollapseAll()
	if !m.collapsedFiles["two.go"] {
		t.Errorf("folding every file left one open: %v", m.collapsedFiles)
	}

	if m = m.toggleCollapseAll(); m.collapsedFiles["one.go"] || m.collapsedFiles["two.go"] {
		t.Errorf("folding every file twice did not open them again: %v", m.collapsedFiles)
	}
}
