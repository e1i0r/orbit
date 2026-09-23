package panes

// The button on a delivery verb's node: what it offers, and when.

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// button is the last sub-item of a node, drawn.
func button(e Env, st handStep) string {
	rows := subRows(e.handSubItems(st), "   ")

	return ansi.Strip(rows[len(rows)-1])
}

// TestAFailedDeliveryVerbOffersToBeAskedAgain.
//
// The node went red with the reason on it and nothing else. Elio, looking
// at a failed CREATE PR: "I don't see the retry button here." It used to
// name the key it was offered under, which read as instructions and not as
// something to press. Now it is the thing to press.
func TestAFailedDeliveryVerbOffersToBeAskedAgain(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(2 * time.Minute)},
		{
			Kind: "deliver.answered", Verb: "CREATE PR", At: ago(time.Minute),
			Cause: "it said it had started the work and would finish later",
		},
	})

	steps := e.byHand()
	if len(steps) != 1 {
		t.Fatalf("the record holds %d hand steps, want one", len(steps))
	}

	if drawn := button(e, steps[0]); !strings.Contains(drawn, "↻") {
		t.Errorf("a failed CREATE PR offers no way to ask again: %q", drawn)
	}
}

// TestAVerbThatWorkedIsAskedForAgainAndNotRetried: the two endings are
// different sentences to a reader and the same call underneath, and a
// reader who reads "try it again" about something that worked goes looking
// for what went wrong.
func TestAVerbThatWorkedIsAskedForAgainAndNotRetried(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(2 * time.Minute)},
		{
			Kind: "deliver.answered", Verb: "CREATE PR", At: ago(time.Minute),
			Text: "https://github.com/e1i0r/orbit/pull/184",
		},
	})

	drawn := button(e, e.byHand()[0])
	if !strings.Contains(drawn, "ask") {
		t.Errorf("a verb that worked offers no way to ask for it again: %q", drawn)
	}

	if strings.Contains(drawn, "try") {
		t.Errorf("a verb that worked is offered as a retry: %q", drawn)
	}
}

// TestAVerbStillOutOffersToBeStopped is the hole Elio fell into: RESOLVE
// COMMENTS out on a task he had already cancelled, the node on ⚡ with a
// spinner beside it, and nothing anywhere to press.
func TestAVerbStillOutOffersToBeStopped(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "RESOLVE COMMENTS", By: "supervisor", At: ago(2 * time.Minute)},
	})

	if drawn := button(e, e.byHand()[0]); !strings.Contains(drawn, "■") {
		t.Errorf("a verb still out cannot be stopped: %q", drawn)
	}
}

// TestTheHandButtonAnswersWhereItIsDrawn: the tree hands the hit test one
// map of buttons, and a verb's node numbers itself past the last phase.
func TestTheHandButtonAnswersWhereItIsDrawn(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(2 * time.Minute)},
	})
	e.RowOpen = func(row int) bool { return row == len(e.Flow.Phases) }

	rows, _, buttons := pipeline(e)

	at, on := -1, false

	for row, node := range buttons {
		if node >= len(e.Flow.Phases) {
			at, on = row, true
		}
	}

	if !on {
		t.Fatalf("no button on a verb's node:\n%s", ansi.Strip(strings.Join(rows, "\n")))
	}

	if drawn := ansi.Strip(rows[at]); !strings.Contains(drawn, "■") {
		t.Errorf("the row the hit test answers is not the button: %q", drawn)
	}
}

// TestHandsAreNumberedPastTheLastPhase: the window turns a button press
// back into a verb by subtracting the phase count, so the two have to
// agree about the order.
func TestHandsAreNumberedPastTheLastPhase(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(3 * time.Minute)},
		{Kind: "deliver.answered", Verb: "CREATE PR", At: ago(2 * time.Minute), Text: "opened"},
		{Kind: "deliver.asked", Verb: "MORE TESTS", By: "supervisor", At: ago(time.Minute)},
	})

	hands := Hands(e)
	if len(hands) != 2 {
		t.Fatalf("Hands answers %d verbs, want two", len(hands))
	}

	if hands[0].Verb != "CREATE PR" || hands[1].Verb != "MORE TESTS" {
		t.Errorf("Hands is out of the order the tree draws: %q then %q", hands[0].Verb, hands[1].Verb)
	}

	if hands[0].Ended.IsZero() {
		t.Error("a verb that came back reads as still out")
	}

	if !hands[1].Ended.IsZero() {
		t.Error("a verb still out reads as come back")
	}
}
