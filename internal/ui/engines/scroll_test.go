package engines

// The knobs list is taller than the screen — opencode alone answers to
// sixty-four models — so it scrolls under whatever is chosen.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/roster"
)

// enginesLongList is one engine with more models than a short terminal can
// draw at once, which is what every engine's catalogue looks like now that
// each one is the whole list rather than a shortlist.
func enginesLongList() []roster.Engine {
	models := []roster.Choice{{ID: "", Label: "default"}}

	for i := 1; i < 40; i++ {
		id := fmt.Sprintf("model-%02d", i)
		models = append(models, roster.Choice{ID: id, Label: id})
	}

	return []roster.Engine{{Name: "opencode", Available: true, Models: models}}
}

// knobsOnALongList is that catalogue on a screen too short for it, with the
// engine already chosen so its models are the rows being walked.
func knobsOnALongList(t *testing.T) (State, Env) {
	t.Helper()

	e := world(t)
	e.Engines = enginesLongList
	e.Settled = "opencode"

	s := Open(0, Knobs{Engine: "opencode"}).foldEngine("opencode", true)

	return s, e
}

// selectedKnob is what the cursor is on, asked of the rows rather than of
// the screen: the test needs to know what should be drawn before it looks.
func selectedKnob(t *testing.T, s State, e Env) string {
	t.Helper()

	rows := s.collectEngineRows(e)

	idxs := selectableEngineIndices(rows)
	if s.sel < 0 || s.sel >= len(idxs) {
		t.Fatalf("the selection is %d, and there are %d rows to be on", s.sel, len(idxs))
	}

	return strings.TrimSpace(rows[idxs[s.sel]].title)
}

// markedKnob is the drawn line the cursor is on, or nothing when the cursor
// is on a row the screen scrolled past.
func markedKnob(drawn []string) string {
	for _, line := range drawn {
		if strings.Contains(line, cells.Mark) {
			return line
		}
	}

	return ""
}

// TestTheKnobsListFollowsTheChoiceDown: forty models do not fit on twenty
// rows, and a cursor walked off the bottom of the screen is a reader
// pressing ↓ at a list that has stopped answering.
func TestTheKnobsListFollowsTheChoiceDown(t *testing.T) {
	s, e := knobsOnALongList(t)

	for range 30 {
		s, _ = s.Key(press("down"), e)

		want := selectedKnob(t, s, e)

		shown := drawn(s, e)
		if !strings.Contains(markedKnob(shown), want) {
			t.Fatalf("%q is chosen and the screen is showing:\n%s", want, strings.Join(shown, "\n"))
		}
	}
}

// TestTheKnobsListComesBackUpWithIt. Walking back is the half a list that
// only scrolled one way would leave the reader stranded in.
func TestTheKnobsListComesBackUpWithIt(t *testing.T) {
	s, e := knobsOnALongList(t)
	s.sel = 30
	s = s.keepEngineRowSeen(e)

	for range 30 {
		s, _ = s.Key(press("up"), e)

		want := selectedKnob(t, s, e)

		shown := drawn(s, e)
		if !strings.Contains(markedKnob(shown), want) {
			t.Fatalf("%q is chosen and the screen is showing:\n%s", want, strings.Join(shown, "\n"))
		}
	}
}

// TestAKnobIsWhereItWasDrawnAfterScrolling: the pointer reads the same
// offset the drawing used, or a click lands on whichever row used to be
// there.
func TestAKnobIsWhereItWasDrawnAfterScrolling(t *testing.T) {
	s, e := knobsOnALongList(t)
	s.sel = 25
	s = s.keepEngineRowSeen(e)

	if s.offset == 0 {
		t.Fatal("a selection twenty-five rows down left the list at the top")
	}

	rows := s.collectEngineRows(e)
	idxs := selectableEngineIndices(rows)

	got := s.hit(10, e.Frame.Body.Y+engineTop+1, e)
	if got.Kind != point.EngineRow {
		t.Fatalf("a click on a drawn row = %+v, want the row under it", got)
	}

	shown := drawn(s, e)

	want := strings.TrimSpace(rows[idxs[got.Pane]].title)
	if !strings.Contains(shown[engineTop+1], want) {
		t.Errorf("the click answered %q, and the line it landed on reads %q", want, shown[engineTop+1])
	}
}

// TestTheWheelWalksTheKnobsAndStops at either end: the arrows wrap because
// a reader asked for one more row, and a wheel that wrapped would take the
// list out from under a hand that is still turning it.
func TestTheWheelWalksTheKnobsAndStops(t *testing.T) {
	s, e := knobsOnALongList(t)

	n := len(selectableEngineIndices(s.collectEngineRows(e)))

	if end := s.Wheel(1000, e); end.sel != n-1 {
		t.Errorf("wheeling past the bottom = row %d, want the last one, %d", end.sel, n-1)
	}

	if top := s.Wheel(-1000, e); top.sel != 0 {
		t.Errorf("wheeling past the top = row %d, want the first one", top.sel)
	}
}
