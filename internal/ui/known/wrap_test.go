package known

// The arrows at the ends of the list.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// TestTheArrowsGoRoundTheEnds. Up from the first rule is the last one and
// down from the last is the first, the way every other list in the window
// goes round.
func TestTheArrowsGoRoundTheEnds(t *testing.T) {
	general := knowledge.Scope{Kind: knowledge.General}
	s, e := onScreen(t, known("one", general), known("two", general), known("three", general))

	s, _ = s.Key(press("up"), e)
	if s.sel != s.last() {
		t.Fatalf("up from the first rule is row %d, want the last, %d", s.sel, s.last())
	}

	s, _ = s.Key(press("down"), e)
	if s.sel != 0 {
		t.Errorf("down from the last rule is row %d, want the first", s.sel)
	}
}
