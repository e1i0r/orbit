package supervisor

// Taking a line back out of the thread.
//
// The gesture that reads least and matters most of the ones on this screen:
// a line the supervisor is told is a line every run after it is told, so
// somebody who said the wrong thing needs a way to stop it being said —
// and a screen that pretended to take one back while leaving it in the
// thread would be the worst of the three possible answers.

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// picking is the screen with the thread open and a line under the cursor.
func picking(t *testing.T, kept *held) (State, Env) {
	t.Helper()

	s, e := opened(t, kept)

	s, _ = s.Key(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}, e)
	if !s.picking {
		t.Fatal("ctrl-r did not start picking a line")
	}

	return s, e
}

// saidThree is a thread with three lines in it, oldest first.
func saidThree() *held {
	return &held{lines: []view.SupervisorLine{
		spoke(fixtureNow.Add(-3*time.Minute), "c1", "operator", "use codex here"),
		spoke(fixtureNow.Add(-2*time.Minute), "c1", "zeta", "nothing is stuck"),
		spoke(fixtureNow.Add(-time.Minute), "c1", "operator", "never mind"),
	}}
}

// TestALineTakenBackIsTakenBackInTheRecord. The screen says it and the
// record does it: a gesture that only changed what was drawn would leave
// every future run still being told the line.
func TestALineTakenBackIsTakenBackInTheRecord(t *testing.T) {
	kept := saidThree()

	s, e := picking(t, kept)

	s, out := s.Key(press("enter"), e)

	if len(kept.back) != 1 {
		t.Fatalf("%d lines were taken back in the record, want one", len(kept.back))
	}

	if s.picking {
		t.Error("taking a line back left the screen picking one")
	}

	if !strings.Contains(out.Said, "took that line back") {
		t.Errorf("it said %q, want it to say the line was taken back", out.Said)
	}
}

// TestLeavingThePickTakesNothingBack. Escape is how a reader gets out of
// something they opened by accident, and this is the one gesture on this
// screen that cannot be undone.
func TestLeavingThePickTakesNothingBack(t *testing.T) {
	for _, way := range []tea.KeyPressMsg{
		{Code: tea.KeyEscape},
		{Code: 'r', Mod: tea.ModCtrl},
	} {
		kept := saidThree()

		s, e := picking(t, kept)

		s, _ = s.Key(way, e)

		if s.picking {
			t.Errorf("%v left the screen picking a line", way)
		}

		if len(kept.back) != 0 {
			t.Errorf("%v took a line back: %v", way, kept.back)
		}
	}
}

// TestTheCursorStopsAtBothEndsOfTheThread. It is a list with one row chosen,
// and a cursor that ran off either end would take back a line the reader
// cannot see.
func TestTheCursorStopsAtBothEndsOfTheThread(t *testing.T) {
	s, e := picking(t, saidThree())

	for range 10 {
		s, _ = s.Key(press("down"), e)
	}

	if s.pick != len(s.lines)-1 {
		t.Errorf("the cursor ran to %d, want it to stop at %d", s.pick, len(s.lines)-1)
	}

	for range 10 {
		s, _ = s.Key(press("up"), e)
	}

	if s.pick != 0 {
		t.Errorf("the cursor ran to %d, want it to stop at the top", s.pick)
	}
}

// TestTheWheelMovesThePickAndNotTheThread. The mouse does what the arrows
// do, so the hand does not have to know which of the two it is holding.
func TestTheWheelMovesThePickAndNotTheThread(t *testing.T) {
	s, e := picking(t, saidThree())

	// Picking opens on the newest line, which is the one somebody who just
	// said the wrong thing is reaching for.
	if s.pick != len(s.lines)-1 {
		t.Fatalf("picking opened on line %d, want the newest", s.pick)
	}

	if got := s.Wheel(-1, e); got.pick != s.pick-1 {
		t.Errorf("a notch up left the pick at %d, want %d", got.pick, s.pick-1)
	}

	if got := s.Wheel(-99, e); got.pick != 0 {
		t.Errorf("a notch up from the first left the pick at %d, want the top", got.pick)
	}

	if got := s.Wheel(99, e); got.pick != len(s.lines)-1 {
		t.Errorf("a notch down from the last left the pick at %d, want the end", got.pick)
	}
}

// TestALineAlreadyTakenBackSaysSoRatherThanDoingItTwice. The record is
// append-only and so is what somebody decided about a line: a second
// retraction would be a row saying something was withdrawn that already was.
func TestALineAlreadyTakenBackSaysSoRatherThanDoingItTwice(t *testing.T) {
	kept := saidThree()

	// The newest, because that is the one picking opens on.
	kept.lines[len(kept.lines)-1].Retracted = true

	s, e := picking(t, kept)

	_, out := s.Key(press("enter"), e)

	if len(kept.back) != 0 {
		t.Errorf("a line already taken back was taken back again: %v", kept.back)
	}

	if !strings.Contains(out.Said, "already") {
		t.Errorf("it said %q, want it to say the line was already taken back", out.Said)
	}
}

// TestARetractionThatFailedSaysWhat. Nothing is hidden: a line the record
// would not take back is still in the thread, and a screen that said it had
// gone would be lying about what every future run is told.
func TestARetractionThatFailedSaysWhat(t *testing.T) {
	kept := saidThree()

	s, e := picking(t, kept)
	e.Retract = func(time.Time) error { return errors.New("the record is read-only") }

	_, out := s.Key(press("enter"), e)

	if !strings.Contains(out.Said, "read-only") {
		t.Errorf("it said %q, want what stopped it", out.Said)
	}
}

// TestAWindowBuiltWithNoRetractionSaysNothingAndDoesNothing, rather than
// reaching for a door it was not given.
func TestAWindowBuiltWithNoRetractionSaysNothingAndDoesNothing(t *testing.T) {
	s, e := picking(t, saidThree())
	e.Retract = nil

	next, out := s.Key(press("enter"), e)

	if out.Said != "" {
		t.Errorf("a build with no retraction said %q", out.Said)
	}

	if next.picking {
		t.Error("it left the screen picking a line it cannot take back")
	}
}
