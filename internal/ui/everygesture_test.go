package ui

// Every key that changes a task says so, pressed as a person presses it.
//
// This exists because the same bug has now shipped twice. Both times a
// gesture was built correctly and wired to nothing a person touches: the
// compose form's clear button, drawn and hit-tested and never routed, and
// the signalled cancel, which `x` never reached because the confirm went
// down a line of its own.
//
// Both times the tests were right about the function and silent about the
// keystroke. So this presses the keys.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// TestEveryGestureKeyEitherActsOrSaysWhyNot.
//
// The invariant, which is stronger than any list of verbs and states: a
// key a reader presses on a task never does nothing silently. Either the
// window is now waiting on a gesture and says so, or it says why it
// refused. Both of the shipped bugs were the third case.
//
// Every task on the board is tried with every key, so this does not
// depend on the fixture holding a paused task or a finished one.
func TestEveryGestureKeyEitherActsOrSaysWhyNot(t *testing.T) {
	keys := []struct {
		key     string
		confirm bool
		verb    string
	}{
		{key: "p", verb: gesturePause},
		{key: "r", verb: gestureResume},
		{key: "h", verb: gestureContinue},
		{key: "s", confirm: true, verb: gestureSkip},
		{key: "x", confirm: true, verb: gestureCancel},
		{key: "b", confirm: true, verb: gestureRequeue},
		{key: "d", verb: gestureRead},
	}

	base := armed(t)

	acted := map[string]bool{}

	for _, task := range base.board.Tasks {
		for _, c := range keys {
			m := armed(t)
			m.detail, m.screen = task.ID, screenList

			// A task in a band this window has collapsed has no row to
			// put the cursor on, and a key pressed on no row is not this
			// test's subject.
			at, on := cursorOn(m, task.ID)
			if !on {
				continue
			}

			m.cursor = at

			m, _ = pressed(t, m, c.key)
			if c.confirm && m.confirm != confirmNone {
				m, _ = pressed(t, m, confirmYes)
			}

			switch {
			case m.await.id == task.ID && m.await.verb == c.verb:
				acted[c.verb] = true

				if word, on := m.awaitedWord(task.ID); !on || word == "" {
					t.Errorf("%q on %s armed %s and the status cell says nothing", c.key, task.ID, c.verb)
				}

				if m.message == "" {
					t.Errorf("%q on %s armed %s and the band says nothing", c.key, task.ID, c.verb)
				}
			case m.message != "":
				// Refused, and said why. That is an answer.
			default:
				t.Errorf("%q on %s (%v) did nothing and said nothing",
					c.key, task.ID, view.BandOf(task))
			}
		}
	}

	// And every verb was reachable on at least one task, or this proved
	// only that nothing crashes.
	for _, c := range keys {
		if !acted[c.verb] {
			t.Errorf("no task on the board let %q through, so %s was never tested",
				c.key, c.verb)
		}
	}
}

// cursorOn is the row one task is on, and whether it has one.
func cursorOn(m Model, id string) (int, bool) {
	for i, r := range m.rows() {
		if !r.head && !r.blank && r.task.ID == id {
			return i, true
		}
	}

	return 0, false
}

// TestPauseSaysWhenItLands.
//
// A cancel kills a process. A pause cannot: the run is inside an engine
// call, there is no way to suspend that and have anything resume cleanly,
// and the word is read at the next phase boundary — which on a long phase
// is minutes away. A row saying only "pausing…" for ten of them reads as a
// gesture that did not take, so it says what it is waiting for.
func TestPauseSaysWhenItLands(t *testing.T) {
	m := armed(t)

	id, ok := aTaskIn(m, view.Running)
	if !ok {
		t.Skip("no running task on the fixture board")
	}

	m = m.awaiting(id, gesturePause)

	word, _ := m.awaitedWord(id)
	if !strings.Contains(word, "end") {
		t.Errorf("the cell says %q, want it to say the pause lands at the end of the phase", word)
	}

	if !strings.Contains(m.message, "end") {
		t.Errorf("the band says %q, want it to say when the pause lands", m.message)
	}
}

// armed is a window with every port a gesture might reach, all of them
// answering yes and none of them doing anything, and a board carrying a
// task in each of the states a verb needs.
//
// The fixture board has none that are parked or finished-and-unread, so
// four of the seven keys were refused everywhere and the coverage check
// below caught that this test was proving nothing about them.
func armed(t *testing.T) Model {
	t.Helper()

	m, _ := testModel(t, 120, 40)
	m.board.Tasks = append(m.board.Tasks,
		// Parked: stopped at a gate with its process still there, which
		// is what resume, skip and hand back are all about.
		view.Task{
			ID: "PARKED-1", Repo: "payments", Engine: "claude", Band: view.NeedsYou,
			Live: view.LiveHeld, Reason: view.Reason{Key: view.ReasonGate},
			Since: ago(time.Minute), Started: ago(10 * time.Minute),
		},
		// Finished and not yet read, which is the whole of what d is for.
		view.Task{
			ID: "UNREAD-1", Repo: "payments", Engine: "claude", Band: view.Done,
			Since: ago(time.Minute), Started: ago(10 * time.Minute),
		},
	)

	// Hand back also wants an engine that can carry a session on and a
	// keyboard this window took.
	m.taken = map[string]bool{"PARKED-1": true}
	m.opts.CanResume = func(string) bool { return true }

	// Every band open, or a task in a collapsed one has no row for the
	// cursor to be on and the key is never pressed at it.
	m.expanded = map[view.Band]bool{
		view.ToDo: true, view.NeedsYou: true, view.Running: true, view.Done: true,
	}

	m.opts.Stop = func(view.Task) error { return nil }
	m.opts.Control = func(view.Task, string) error { return nil }
	m.opts.Requeue = func(view.Task) error { return nil }
	m.opts.MarkRead = func(view.Task) error { return nil }
	m.opts.DeleteTask = func(view.Task) error { return nil }

	return m
}

// aTaskIn is the id of a task the fixture board has in one band.
func aTaskIn(m Model, band view.Band) (string, bool) {
	for _, t := range m.board.Tasks {
		if view.BandOf(t) == band {
			return t.ID, true
		}
	}

	return "", false
}
