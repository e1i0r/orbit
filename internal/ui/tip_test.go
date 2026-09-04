package ui

// Asking the window what a key does, without leaving the board.

import (
	"strings"
	"testing"

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
