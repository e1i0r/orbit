package supervisor

// The screen itself: what typing does, what is drawn, and taking a line back.

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

func TestTypingALineAndSendingItWritesItSignedByTheOperator(t *testing.T) {
	kept := &held{}
	s, e := opened(t, kept)

	for _, k := range []string{"k", "e", "e", "p"} {
		s, _ = s.Key(press(k), e)
	}

	if s.input != "keep" {
		t.Errorf("what was typed reads %q", s.input)
	}

	s, _ = s.Key(press("backspace"), e)
	if s.input != "kee" {
		t.Errorf("after backspace the line reads %q", s.input)
	}

	s, out := s.Key(press("enter"), e)

	if len(kept.wrote) != 1 || kept.wrote[0].Text != "kee" {
		t.Fatalf("the record holds %+v", kept.wrote)
	}

	// The screen signs every message "operator", which is what every other
	// door writes. The thread is one conversation: a particular person's
	// name hardcoded here makes the same person read as two participants
	// depending on which door they came through.
	if kept.wrote[0].By != "operator" {
		t.Errorf("the line is signed %q, want operator", kept.wrote[0].By)
	}

	if s.input != "" {
		t.Errorf("the line was not cleared after sending: %q", s.input)
	}

	if !out.Asking {
		t.Error("the line was written but no question went out")
	}
}

func TestTheScreenDrawsItsTitleAndWhatWasSaid(t *testing.T) {
	s, e := open(t)

	rows := s.rows(25, 100, e)
	if len(rows) != 25 {
		t.Errorf("the screen filled %d rows of the 25 it was given", len(rows))
	}

	if full := strings.Join(rows, "\n"); !strings.Contains(full, "Supervisor") {
		t.Errorf("the screen has no title:\n%s", full)
	}

	s.lines = []view.SupervisorLine{{
		At:      time.Date(2026, 8, 28, 14, 0, 0, 0, time.UTC),
		By:      "supervisor",
		Channel: "autopilot",
		TaskID:  "ORB-1",
		Text:    "task ORB-1 completed all gates",
	}}

	full := strings.Join(s.rows(25, 100, e), "\n")
	if !strings.Contains(full, "ORB-1") || !strings.Contains(full, "completed all gates") {
		t.Errorf("what was said is not on the screen:\n%s", full)
	}
}

// TestAWithdrawnLineIsMarkedAndNotHidden: the screen shows the same thread
// the supervisor's prompt is built from, so a line that no longer steers the
// supervisor must not read on screen like one that does.
func TestAWithdrawnLineIsMarkedAndNotHidden(t *testing.T) {
	s, e := open(t)
	s.lines = []view.SupervisorLine{{
		At:        time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC),
		By:        "elio",
		Channel:   "tui",
		Text:      "the one I regret",
		Retracted: true,
	}}

	full := strings.Join(s.rows(25, 100, e), "\n")
	if !strings.Contains(full, "retract") {
		t.Errorf("a withdrawn line is drawn like any other:\n%s", full)
	}

	if !strings.Contains(full, "the one I regret") {
		t.Errorf("a withdrawn line was hidden instead of marked:\n%s", full)
	}
}

// TestTheScreenFitsTheTerminal. It is drawn as rows of text and not as a
// box, so what has to hold is that it fills exactly the height it was given
// and that no line runs off the right edge — a wrapped reply measured
// against a width nobody clipped it to is how a frame ends up with a tail
// hanging past it.
func TestTheScreenFitsTheTerminal(t *testing.T) {
	s, e := open(t)
	s.lines = []view.SupervisorLine{
		{At: fixtureNow, By: "zeta", Channel: "mcp", Text: "a reply long enough to wrap more than once on a narrow terminal, with a `code span` in it"},
		{At: fixtureNow.Add(time.Minute), By: "operator", Channel: "tui", TaskID: "ORB-1", Text: "short one"},
	}

	for _, size := range []struct{ w, h int }{{120, 34}, {80, 24}, {60, 16}, {200, 50}} {
		rows := s.rows(size.h, size.w, e)
		if len(rows) != size.h {
			t.Errorf("%dx%d: %d rows, want %d", size.w, size.h, len(rows), size.h)
		}

		for i, r := range rows {
			if got := lipgloss.Width(r); got > size.w {
				t.Errorf("%dx%d: row %d is %d cells, wider than the terminal: %q", size.w, size.h, i, got, r)
			}
		}
	}
}

// TestPickingTakesTheChosenLineBack is the retraction: ^R picks, the arrows
// choose, ↵ withdraws.
func TestPickingTakesTheChosenLineBack(t *testing.T) {
	kept := &held{}
	s, e := opened(t, kept)

	first := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	second := first.Add(time.Minute)
	s.lines = []view.SupervisorLine{
		{At: first, By: "operator", Channel: "tui", Text: "the one I regret"},
		{At: second, By: "zeta", Channel: "mcp", Text: "an answer"},
	}

	// Typing is off while picking: r on its own is a letter, ^R is the mode.
	s, _ = s.Key(ctrl('r'), e)
	if !s.picking {
		t.Fatal("^R did not open the picker")
	}

	if s.pick != 1 {
		t.Errorf("the picker opened on line %d, want the last one", s.pick)
	}

	s, _ = s.Key(press("up"), e)
	if s.pick != 0 {
		t.Errorf("up moved to %d, want 0", s.pick)
	}

	// And it stops at the top rather than walking off it.
	s, _ = s.Key(press("up"), e)
	if s.pick != 0 {
		t.Errorf("up past the first line moved to %d", s.pick)
	}

	s, out := s.Key(press("enter"), e)

	if len(kept.back) != 1 || !kept.back[0].Equal(first) {
		t.Errorf("the line taken back was %v, want [%v]", kept.back, first)
	}

	wantBand(t, out, "took that line back")

	if s.picking {
		t.Error("the picker stayed open after taking a line back")
	}

	if s.input != "" {
		t.Errorf("picking typed into the line: %q", s.input)
	}
}

func TestPickingCancelsAndRefusesWhatItCannotDo(t *testing.T) {
	kept := &held{}
	s, e := opened(t, kept)

	// Nothing said yet: there is no line to point at.
	s, _ = s.Key(ctrl('r'), e)
	if s.picking {
		t.Error("^R opened the picker over an empty thread")
	}

	s.lines = []view.SupervisorLine{
		{At: fixtureNow, By: "operator", Channel: "tui", Text: "said once", Retracted: true},
	}

	s, _ = s.Key(ctrl('r'), e)
	s, _ = s.Key(press("esc"), e)

	if s.picking {
		t.Error("esc left the picker open")
	}

	if len(kept.back) != 0 {
		t.Error("esc took a line back")
	}

	// A line already withdrawn is not withdrawn twice.
	s, _ = s.Key(ctrl('r'), e)

	_, out := s.Key(press("enter"), e)

	if len(kept.back) != 0 {
		t.Errorf("a line already taken back was retracted again: %v", kept.back)
	}

	wantBand(t, out, "already taken back")
}
