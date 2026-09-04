package ui

// Asking the window what a key does, without leaving the board.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// TestAQuestionMarkAsksWhichKey: ? does not act, it waits, and the band says
// what it is waiting for.
func TestAQuestionMarkAsksWhichKey(t *testing.T) {
	m, got := parkedModel(t)
	m = onRow(t, m, "ACME-2705")

	armed, cmd := advance(t, m, press("?"))
	if cmd != nil || got.word != "" {
		t.Fatalf("'?' did something: wrote %q", got.word)
	}

	if !armed.tip.armed {
		t.Fatal("'?' left nothing waiting for a key")
	}

	if band := ansi.Strip(armed.bandLine(120)); !strings.Contains(band, "which key?") {
		t.Errorf("the band says %q, want the question '?' raised", band)
	}
}

// TestTheKeyAskedAboutIsExplainedAndNotPressed.
//
// This is the whole point of the gesture: s under a parked run is a verb
// that would skip a phase, and asked about it says what it would do instead
// of doing it.
func TestTheKeyAskedAboutIsExplainedAndNotPressed(t *testing.T) {
	m, got := parkedModel(t)
	m = onRow(t, m, "ACME-2705")

	armed, _ := advance(t, m, press("?"))

	after, cmd := advance(t, armed, press("s"))
	if cmd != nil || got.word != "" || after.confirm != confirmNone {
		t.Fatalf("the key asked about was pressed: word %q, confirm %v", got.word, after.confirm)
	}

	if band := ansi.Strip(after.bandLine(120)); !strings.Contains(band, "without the phase it is waiting in front of") {
		t.Errorf("the band says %q, want what s does", band)
	}

	if after.tip.armed {
		t.Error("one answer left the question still up")
	}
}

// TestAskingTwiceOpensTheWholeCheatSheet: ? about ? is the overlay it used
// to be one press away, and the prompt says so.
func TestAskingTwiceOpensTheWholeCheatSheet(t *testing.T) {
	m, _ := parkedModel(t)

	armed, _ := advance(t, m, press("?"))

	after, _ := advance(t, armed, press("?"))
	if after.screen != screenHelp {
		t.Errorf("screen after '? ?' = %v, want the help overlay", after.screen)
	}
}

// TestEscapeLeavesTheQuestionWithoutAnswering it, and takes the prompt off
// the band with it: a window still asking "which key?" after esc is a window
// that has stopped listening.
func TestEscapeLeavesTheQuestionWithoutAnswering(t *testing.T) {
	m, _ := parkedModel(t)

	armed, _ := advance(t, m, press("?"))

	after, _ := advance(t, armed, press("esc"))
	if after.tip.armed {
		t.Error("esc left the question up")
	}

	if band := ansi.Strip(after.bandLine(120)); strings.Contains(band, "which key?") {
		t.Errorf("the band still says %q", band)
	}
}

// TestAKeyThatDoesNothingSaysSo. A reader asking about a key is told what it
// does, and "nothing" is one of the answers — the silence it replaces is
// what a broken keyboard feels like.
func TestAKeyThatDoesNothingSaysSo(t *testing.T) {
	m, _ := parkedModel(t)

	armed, _ := advance(t, m, press("?"))

	after, _ := advance(t, armed, press("§"))
	if band := ansi.Strip(after.bandLine(120)); !strings.Contains(band, "nothing in this window answers") {
		t.Errorf("the band says %q, want the answer about a key nothing is bound to", band)
	}
}

// TestPointingAtAHintExplainsIt, which is the same answer ? gives, without
// the keystroke: the reader is already looking at the hint.
func TestPointingAtAHintExplainsIt(t *testing.T) {
	m, got := parkedModel(t)
	m = onRow(t, m, "ACME-2705")

	y := m.frame.Bar.Y

	_, hints, _ := m.barLayout(m.frame.Bar.W)

	x, found := 0, false

	for _, h := range hints {
		if h.key == "r" {
			x, found = h.x, true
		}
	}

	if !found {
		t.Fatal("the bar offers no r to point at")
	}

	on := m.hover(tea.Mouse{X: x, Y: y})
	if band := ansi.Strip(on.bandLine(120)); !strings.Contains(band, "beginning with the phase") {
		t.Errorf("the band says %q, want what the hint under the pointer does", band)
	}

	if got.word != "" {
		t.Errorf("pointing at the hint wrote %q", got.word)
	}

	off := on.hover(tea.Mouse{X: x, Y: y + 5})
	if band := ansi.Strip(off.bandLine(120)); strings.Contains(band, "beginning with the phase") {
		t.Errorf("the sentence stayed behind the pointer: %q", band)
	}
}

// TestTheCheatSheetSaysWhatTheVerbsDo, in the same words the band does.
//
// The line this replaces named p, u and s for pause, unblock and note: two
// of the three had not been those keys for a long time, and nothing failed
// when they changed. Reading the sheet out of the bindings is what makes
// that impossible.
func TestTheCheatSheetSaysWhatTheVerbsDo(t *testing.T) {
	m, _ := parkedModel(t)

	sheet := ansi.Strip(strings.Join(m.openHelp().helpRows(80, 140), "\n"))

	for _, b := range m.keys.taskVerbs() {
		if !strings.Contains(sheet, "["+b.Help().Key+"]") {
			t.Errorf("the cheat sheet does not offer %q", b.Help().Key)
		}

		if said := m.meaning(firstKey(b)); !strings.Contains(sheet, said) {
			t.Errorf("the cheat sheet does not say %q", said)
		}
	}
}
