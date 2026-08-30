package ui

// A run marker that is there and cannot be read, and what the window does
// about it: nothing, in as many words. Every verb below asks whether a
// process holds the task, nothing can answer, and both guesses cost
// something — so each of them refuses with the same reason, and the reason
// asks the reader to look at the file.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// unreadable is a run in the fixtures whose marker has stopped answering.
func unreadable() view.Task {
	return view.Task{ID: "ACME-11", Band: view.Running, Live: view.LiveUnknown, Attempt: 1, Engine: "claude"}
}

// TestEveryVerbAboutAProcessRefusesAnUnreadableMarker. The five verbs ask one
// question between them, so they refuse with one reason rather than five ways
// of saying nobody knows.
func TestEveryVerbAboutAProcessRefusesAnUnreadableMarker(t *testing.T) {
	english, _ := printers(t)

	keys := NewKeys(english)
	for _, verb := range []string{"p", "r", "x", "t", "D"} {
		a := find(keys.Affordances(unreadable(), Conditions{CanResume: true}), verb)
		if a.OK {
			t.Errorf("%q is offered on a task nobody can say anything about", verb)
		}

		if a.WhyNot.Name != whyMarkerUnreadable {
			t.Errorf("%q refuses with %q, want %q", verb, a.WhyNot.Name, whyMarkerUnreadable)
		}
	}
}

// TestStartingIsRefusedOnAMarkerNobodyCouldRead is the brake the third state
// was added for. Liveness was a bool, an unreadable marker fell to the false
// side of it, and false is what this screen checks before it opens — so the
// one task orbit could say nothing about was the one task it would start a
// second engine on.
func TestStartingIsRefusedOnAMarkerNobodyCouldRead(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	const id = "ACME-2705" // running in the fixtures, and here beyond reading

	for i := range m.board.Tasks {
		if m.board.Tasks[i].ID == id {
			m.board.Tasks[i].Live = view.LiveUnknown
		}
	}

	next, _ := onto(t, m, id).openStart()

	after := asModel(t, next)
	if after.screen != screenList {
		t.Errorf("the start dialog opened on screen %v, want the list left where it was", after.screen)
	}

	if !strings.Contains(after.message, id) || !strings.Contains(after.message, "run marker") {
		t.Errorf("the window said %q, want a sentence naming the task and its marker", after.message)
	}
}
