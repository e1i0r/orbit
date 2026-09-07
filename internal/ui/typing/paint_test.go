package typing

// What a field looks like on screen: the caret, the selection, and the runs
// between them.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// plain is the paint an ordinary run gets, which the caller supplies: the
// field does not decide what colour the text it is not selecting is.
func plain(s string) string { return s }

// TestEveryCharacterSurvivesThePaint. Whatever is drawn over it, the line is
// still the value: a paint that dropped a rune would be text the reader
// typed and cannot see.
func TestEveryCharacterSurvivesThePaint(t *testing.T) {
	for _, c := range []struct{ from, to, caret int }{
		{0, 0, 0},
		{0, 4, 4},
		{2, 6, 2},
		{0, 10, 10},
	} {
		got := PaintCells("hola mundo", c.from, c.to, c.caret, plain)
		if bare := ansi.Strip(got); !strings.HasPrefix(bare, "hola mundo") {
			t.Errorf("from=%d to=%d caret=%d left %q", c.from, c.to, c.caret, bare)
		}
	}
}

// TestTheCaretIsFoundEvenAtTheEdgeOfASelection. It is the selection's own
// colours made bold, so that a caret sitting inside one is still something
// the eye can find.
func TestTheCaretIsFoundEvenAtTheEdgeOfASelection(t *testing.T) {
	inside := PaintCells("hola mundo", 0, 4, 2, plain)
	if !strings.Contains(inside, theme.Paint(theme.Sel).Bold(true).Render("l")) {
		t.Errorf("the caret inside a selection is not drawn apart from it: %q", inside)
	}

	// At the end of the line the caret is a cell of its own, so there is
	// somewhere for it to be.
	end := PaintCells("hola", 0, 0, 4, plain)
	if got := ansi.Strip(end); got != "hola " {
		t.Errorf("the caret past the last character left %q", got)
	}
}

// TestCellsOfAKindArePaintedInOneGo, so a line carries a handful of escapes
// rather than one per character.
func TestCellsOfAKindArePaintedInOneGo(t *testing.T) {
	got := PaintCells("hola mundo", 0, 4, 9, plain)

	// Three runs: the selection, the plain stretch, and the caret.
	if n := strings.Count(got, "\x1b["); n > 8 {
		t.Errorf("a line of ten characters carries %d escapes: %q", n, got)
	}
}

// TestAFieldSaysWhetherThereIsAnythingInIt, whatever the caret says.
func TestAFieldSaysWhetherThereIsAnythingInIt(t *testing.T) {
	if !New("").Empty() {
		t.Error("a field with nothing in it says it holds something")
	}

	in := New("something")
	in.MoveTo(0)

	if in.Empty() {
		t.Error("a field with the caret at its head says it is empty")
	}
}
