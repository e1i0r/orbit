package supervisor

// What the four gestures leave behind.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestWritingAFactLeavesItInTheThread.
//
// A rule written in a conversation has to stay in the conversation. Without
// this, typing "/aware the fuzz tests hang" wrote a file somewhere, flashed a
// line at the foot of the screen for twenty seconds, and left the thread
// exactly as it was — so the one place the operator was looking had no record
// that anything had happened at all.
func TestWritingAFactLeavesItInTheThread(t *testing.T) {
	kept := &held{}
	s, e := opened(t, kept)
	e.Learn = func(bool, string, string, string) error { return nil }

	_, _ = s.Say("/aware the fuzz tests hang sometimes", e)

	if len(kept.wrote) != 1 {
		t.Fatalf("writing a fact put %d lines in the thread, want one: %+v", len(kept.wrote), kept.wrote)
	}

	if !strings.Contains(kept.wrote[0].Text, "the fuzz tests hang sometimes") {
		t.Errorf("the line in the thread does not say what was written down: %q", kept.wrote[0].Text)
	}
}

// TestWritingAFactAsksTheEngineNothing. It is a thing to write down, not a
// question — spending a model call on it would cost money to be told "ok".
func TestWritingAFactAsksTheEngineNothing(t *testing.T) {
	asked := false

	s, e := opened(t, &held{})
	e.Learn = func(bool, string, string, string) error { return nil }
	e.Ask = func(string, string) tea.Cmd {
		asked = true

		return func() tea.Msg { return nil }
	}

	_, out := s.Say("/rule coverage stays above 90%", e)

	if asked {
		t.Error("writing a fact down asked the engine to answer it")
	}

	if out.Asking {
		t.Error("writing a fact down left the window waiting for an engine")
	}
}

// TestAMessageStillReachesTheEngine, so that reading the first word of a
// line has not quietly turned the conversation into a command line.
func TestAMessageStillReachesTheEngine(t *testing.T) {
	s, e := opened(t, &held{})

	_, out := s.Say("what happened while I was out?", e)
	if !out.Asking || out.Cmd == nil {
		t.Error("an ordinary message did not go to the engine")
	}
}
