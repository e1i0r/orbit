package compose

// Emptying a field, by the button and by the key.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/point"
)

// held is what the field the caret is in currently holds.
func held(s State) string {
	if in := s.active(); in != nil {
		return in.String()
	}

	return ""
}

// TestClearEmptiesTheFieldTheCaretIsIn.
func TestClearEmptiesTheFieldTheCaretIsIn(t *testing.T) {
	s, e := form(t)
	s.field = composeID
	s = typed(s, e, "hello")

	if held(s) != "hello" {
		t.Fatalf("the fixture typed %q, want hello", held(s))
	}

	if got := held(s.cleared()); got != "" {
		t.Errorf("after clear the field holds %q, want it empty", got)
	}
}

// TestBothWaysToClearAgree: the key and the button are one gesture, so
// neither may work while the other does.
func TestBothWaysToClearAgree(t *testing.T) {
	byKey, e := form(t)
	byKey.field = composeID
	byKey = typed(byKey, e, "throw me away")
	byKey, _ = byKey.Key(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, e)

	if got := held(byKey); got != "" {
		t.Errorf("^U left %q in the field", got)
	}

	byClick, e := form(t)
	byClick.field = composeID
	byClick = typed(byClick, e, "throw me away")
	byClick, _ = byClick.Click(point.Target{Kind: point.ComposeClear}, e)

	if got := held(byClick); got != "" {
		t.Errorf("a click on clear left %q in the field", got)
	}
}

// TestTheClearButtonIsDrawnAndAnswersWhereItIsDrawn.
//
// The two halves of a button: the cells it occupies and the cells that
// listen. They are worked out in different files from the same widths, and
// a button that is drawn one cell from where it listens is the defect the
// click map exists to find.
func TestTheClearButtonIsDrawnAndAnswersWhereItIsDrawn(t *testing.T) {
	s, e := form(t)
	s.field = composeID

	drawn := strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n")
	if !strings.Contains(drawn, "Clear") {
		t.Fatalf("the form does not draw a clear button:\n%s", drawn)
	}

	// Walk the top row of the box and find which columns answer clear.
	var first, last int

	found := false

	for x := range e.Frame.Body.W {
		if s.onComposeClear(x, e) {
			if !found {
				first, found = x, true
			}

			last = x
		}
	}

	if !found {
		t.Fatal("no column on the box's top row answers the clear button")
	}

	// It must not overlap paste, which sits immediately before it.
	for x := first; x <= last; x++ {
		if s.onComposePaste(x, e) {
			t.Errorf("column %d answers both paste and clear", x)
		}
	}
}
