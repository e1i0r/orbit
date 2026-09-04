package ui

// The knobs list is taller than the screen — opencode alone answers to
// sixty-four models — so it scrolls under whatever is chosen.

import (
	"fmt"
	"strings"
	"testing"
)

// enginesLongList is one engine with more models than a short terminal can
// draw at once, which is what every engine's catalogue looks like now that
// each one is the whole list rather than a shortlist.
func enginesLongList() []EngineInfo {
	models := []ChoiceInfo{{ID: "", Label: "default"}}

	for i := 1; i < 40; i++ {
		id := fmt.Sprintf("model-%02d", i)
		models = append(models, ChoiceInfo{ID: id, Label: id})
	}

	return []EngineInfo{{Name: "opencode", Available: true, Models: models}}
}

// knobsOnALongList is that roster on a screen too short for it, with the
// engine already chosen so its models are the rows being walked.
func knobsOnALongList(t *testing.T) Model {
	t.Helper()

	m, _ := testModel(t, 100, 20)
	m.opts.Engines = enginesLongList
	m.knobs.Engine = "opencode"

	return m.openEngines()
}

// selectedKnob is what the cursor is on, asked of the rows rather than of
// the screen: the test needs to know what should be drawn before it looks.
func selectedKnob(t *testing.T, m Model) string {
	t.Helper()

	rows := m.collectEngineRows()

	idxs := m.selectableEngineIndices(rows)
	if m.engines.sel < 0 || m.engines.sel >= len(idxs) {
		t.Fatalf("the selection is %d, and there are %d rows to be on", m.engines.sel, len(idxs))
	}

	return strings.TrimSpace(rows[idxs[m.engines.sel]].title)
}

// markedKnob is the drawn line the cursor is on, or nothing when the cursor
// is on a row the screen scrolled past.
func markedKnob(drawn []string) string {
	for _, line := range drawn {
		if strings.Contains(line, markGlyph) {
			return line
		}
	}

	return ""
}

// TestTheKnobsListFollowsTheChoiceDown: forty models do not fit on twenty
// rows, and a cursor walked off the bottom of the screen is a reader
// pressing ↓ at a list that has stopped answering.
func TestTheKnobsListFollowsTheChoiceDown(t *testing.T) {
	m := knobsOnALongList(t)

	for range 30 {
		next, _ := m.enginesKey(press("down"))
		m = asModel(t, next)

		want := selectedKnob(t, m)

		drawn := m.enginesRows(m.frame.Body.H, m.frame.Body.W)
		if !strings.Contains(markedKnob(drawn), want) {
			t.Fatalf("%q is chosen and the screen is showing:\n%s", want, strings.Join(drawn, "\n"))
		}
	}
}

// TestTheKnobsListComesBackUpWithIt. Walking back is the half a list that
// only scrolled one way would leave the reader stranded in.
func TestTheKnobsListComesBackUpWithIt(t *testing.T) {
	m := knobsOnALongList(t)
	m.engines.sel = 30
	m = m.keepEngineRowSeen()

	for range 30 {
		next, _ := m.enginesKey(press("up"))
		m = asModel(t, next)

		want := selectedKnob(t, m)

		drawn := m.enginesRows(m.frame.Body.H, m.frame.Body.W)
		if !strings.Contains(markedKnob(drawn), want) {
			t.Fatalf("%q is chosen and the screen is showing:\n%s", want, strings.Join(drawn, "\n"))
		}
	}
}

// TestAKnobIsWhereItWasDrawnAfterScrolling: the pointer reads the same
// offset the drawing used, or a click lands on whichever row used to be
// there.
func TestAKnobIsWhereItWasDrawnAfterScrolling(t *testing.T) {
	m := knobsOnALongList(t)
	m.engines.sel = 25
	m = m.keepEngineRowSeen()

	if m.engines.offset == 0 {
		t.Fatal("a selection twenty-five rows down left the list at the top")
	}

	rows := m.collectEngineRows()
	idxs := m.selectableEngineIndices(rows)

	got := m.hitEngines(10, m.frame.Body.Y+engineTop+1)
	if got.Kind != TargetEngineRow {
		t.Fatalf("a click on a drawn row = %+v, want the row under it", got)
	}

	drawn := m.enginesRows(m.frame.Body.H, m.frame.Body.W)
	if want := strings.TrimSpace(rows[idxs[got.Pane]].title); !strings.Contains(drawn[engineTop+1], want) {
		t.Errorf("the click answered %q, and the line it landed on reads %q", want, drawn[engineTop+1])
	}
}

// TestTheWheelWalksTheKnobsAndStops at either end: the arrows wrap because
// a reader asked for one more row, and a wheel that wrapped would take the
// list out from under a hand that is still turning it.
func TestTheWheelWalksTheKnobsAndStops(t *testing.T) {
	m := knobsOnALongList(t)

	n := len(m.selectableEngineIndices(m.collectEngineRows()))

	if end := m.pickEngineRow(1000); end.engines.sel != n-1 {
		t.Errorf("wheeling past the bottom = row %d, want the last one, %d", end.engines.sel, n-1)
	}

	if top := m.pickEngineRow(-1000); top.engines.sel != 0 {
		t.Errorf("wheeling past the top = row %d, want the first one", top.engines.sel)
	}
}
