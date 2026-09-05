package ui

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
	m, _ := testModel(t, 100, 30)

	var said []string

	m.opts.RecordSupervisor = func(_, _, message string) error {
		said = append(said, message)

		return nil
	}
	m.opts.Learn = func(bool, string, string, string) error { return nil }

	m = m.openSupervisor()
	m, _ = m.sendSupervisorMessage("/aware the fuzz tests hang sometimes")

	if len(said) != 1 {
		t.Fatalf("writing a fact put %d lines in the thread, want one: %v", len(said), said)
	}

	if !strings.Contains(said[0], "the fuzz tests hang sometimes") {
		t.Errorf("the line in the thread does not say what was written down: %q", said[0])
	}
}

// TestWritingAFactAsksTheEngineNothing. It is a thing to write down, not a
// question — spending a model call on it would cost money to be told "ok".
func TestWritingAFactAsksTheEngineNothing(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	asked := false

	m.opts.RecordSupervisor = func(string, string, string) error { return nil }
	m.opts.Learn = func(bool, string, string, string) error { return nil }
	m.opts.AskSupervisor = func(string, string) (string, error) {
		asked = true

		return "", nil
	}

	m = m.openSupervisor()

	next, cmd := m.sendSupervisorMessage("/rule coverage stays above 90%")
	if cmd != nil {
		cmd() // whatever it wants to do, let it
	}

	if asked {
		t.Error("writing a fact down asked the engine to answer it")
	}

	if next.supervisorBusy {
		t.Error("writing a fact down left the window waiting for an engine")
	}
}

// TestAMessageStillReachesTheEngine, so that reading the first word of a
// line has not quietly turned the conversation into a command line.
func TestAMessageStillReachesTheEngine(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m.opts.RecordSupervisor = func(string, string, string) error { return nil }
	m = m.openSupervisor()

	next, cmd := m.sendSupervisorMessage("what happened while I was out?")
	if !next.supervisorBusy || cmd == nil {
		t.Error("an ordinary message did not go to the engine")
	}

	_ = tea.Batch(cmd)
}
