package known

// Pausing a rule from the screen: the one gesture here that changes a rule
// somebody already agreed to, and the only reversible one.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// TestPausingTakesAReasonAndAsksAboutItLater.
//
// The cheapest thing this screen can do to a rule and the only reversible
// one: it stops the rule applying, says what for, and asks somebody to come
// back to it. A pause with no reason is refused, because a pause with no
// reason is the switch beside it under another name.
func TestPausingTakesAReasonAndAsksAboutItLater(t *testing.T) {
	var paused []knowledge.Fact

	s, e := onScreen(t, knowledge.Fact{
		ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human, Phrase: "of everything",
	})
	e.Replace = func(_, now knowledge.Fact) error {
		paused = append(paused, now)

		return nil
	}

	s, _ = s.Key(press("p"), e)
	if !s.editing {
		t.Fatal("p opened no line to type the reason into")
	}

	// Nothing typed is refused, and nothing is written.
	if _, out := s.Key(press("enter"), e); out.Said == "" || len(paused) != 0 {
		t.Errorf("a pause with no reason wrote %d rules and said %q", len(paused), out.Said)
	}

	s = typed(s, e, "while we get the coverage up")

	if _, out := s.Key(press("enter"), e); out.Said == "" {
		t.Error("pausing said nothing to the reader")
	}

	if len(paused) != 1 {
		t.Fatalf("%d rules were paused", len(paused))
	}

	now := paused[0]
	if now.State != knowledge.Paused || now.Why == "" || !now.Review {
		t.Errorf("it was written as %v, for %q, review=%v", now.State, now.Why, now.Review)
	}
}

// TestARuleAlreadyPausedIsNotPausedAgain, so that the one gesture that
// changes a rule cannot quietly overwrite the reason already on it.
func TestARuleAlreadyPausedIsNotPausedAgain(t *testing.T) {
	already := knowledge.Fact{
		ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human, Phrase: "of everything",
		State: knowledge.Paused, Why: "until we are past 80",
	}

	s, e := onScreen(t, already)

	if after, _ := s.Key(press("p"), e); after.editing {
		t.Error("p opened a line over a rule that is already paused")
	}
}
