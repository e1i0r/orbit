package known

// Forgetting a rule from the review, which is the one screen where the
// evidence is already in front of whoever presses the key.

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// reviewing is the screen open on one rule's own page, with a story and a
// port that records what it was asked to forget.
func reviewing(t *testing.T, story []string) (State, *Env, *[]knowledge.Rule) {
	t.Helper()

	one := known("never log a card number", knowledge.Scope{Kind: knowledge.General})
	one.ID = "card-number"

	var gone []knowledge.Rule

	e := world(t, one)
	e.Story = func(knowledge.Rule) []string { return story }
	e.Forget = func(f knowledge.Rule) error {
		gone = append(gone, f)

		return nil
	}

	return Open(e).openDetail(e), &e, &gone
}

// pressed is one keystroke on the open review.
func pressed(s State, e Env, key string) (State, Out) { return s.Key(press(key), e) }

// TestForgettingTakesTwoPresses. A rule is gone for good, and a single
// keystroke that loses something is one somebody hits by accident on the way
// to another.
func TestForgettingTakesTwoPresses(t *testing.T) {
	s, e, gone := reviewing(t, nil)

	s, out := pressed(s, *e, "f")

	if len(*gone) != 0 {
		t.Fatal("one press forgot the rule")
	}

	if !strings.Contains(out.Said, "again") {
		t.Errorf("the first press said %q, want it to ask for a second", out.Said)
	}

	if _, out = pressed(s, *e, "f"); len(*gone) != 1 {
		t.Fatalf("the second press forgot %d rules, want one", len(*gone))
	}

	if (*gone)[0].ID != "card-number" {
		t.Errorf("it forgot %q", (*gone)[0].ID)
	}

	if out.Said == "" {
		t.Error("forgetting said nothing to the band")
	}
}

// TestAnotherKeyDisarmsIt. A press armed and left armed is a rule one
// keystroke from being lost while somebody reads something else.
func TestAnotherKeyDisarmsIt(t *testing.T) {
	s, e, gone := reviewing(t, nil)

	s, _ = pressed(s, *e, "f")

	// Anything at all, and the plainest thing a reader does on this screen
	// is scroll the story under the rule.
	s, _ = s.Key(press("j"), *e)

	if _, _ = pressed(s, *e, "f"); len(*gone) != 0 {
		t.Error("a key in between left the forget armed, and the next press lost the rule")
	}
}

// TestARuleWithAHistoryIsNotOffered. The key is absent rather than present
// and refusing: a control that says no is one somebody goes looking for a
// way to force.
func TestARuleWithAHistoryIsNotOffered(t *testing.T) {
	s, e, gone := reviewing(t, []string{"it stopped the work in ACME-2662"})

	drawn := ansi.Strip(strings.Join(s.View(30, 96, *e), "\n"))
	if strings.Contains(strings.ToLower(drawn), "forget") {
		t.Errorf("the review offers forgetting a rule that has a history:\n%s", drawn)
	}

	if _, _ = pressed(s, *e, "f"); len(*gone) != 0 {
		t.Error("the key forgot a rule that has a history")
	}
}

// TestARuleWithNoHistoryIsOffered, which is the other half of the same
// claim.
func TestARuleWithNoHistoryIsOffered(t *testing.T) {
	s, e, _ := reviewing(t, nil)

	drawn := ansi.Strip(strings.Join(s.View(30, 96, *e), "\n"))
	if !strings.Contains(strings.ToLower(drawn), "forget") {
		t.Errorf("the review does not offer forgetting a rule that did nothing:\n%s", drawn)
	}
}

// TestAPortThatRefusesIsCarriedToTheBand. The window may not ask the record
// anything, so the last word on whether a rule can go is the port's — and a
// refusal that reaches nobody is a key that looks broken.
func TestAPortThatRefusesIsCarriedToTheBand(t *testing.T) {
	s, e, _ := reviewing(t, nil)
	e.Forget = func(knowledge.Rule) error { return errors.New("it has a history: it was paused") }

	s, _ = pressed(s, *e, "f")

	if _, out := pressed(s, *e, "f"); !strings.Contains(out.Said, "paused") {
		t.Errorf("the band says %q, want what the port refused with", out.Said)
	}
}

// TestWithoutAPortTheKeyIsNotThere. A window built with no store can still
// open this screen, and a key that dereferences a nil port is a window that
// dies.
func TestWithoutAPortTheKeyIsNotThere(t *testing.T) {
	s, e, _ := reviewing(t, nil)
	e.Forget = nil

	drawn := ansi.Strip(strings.Join(s.View(30, 96, *e), "\n"))
	if strings.Contains(strings.ToLower(drawn), "forget") {
		t.Errorf("a window with no store offers forgetting:\n%s", drawn)
	}

	if _, out := pressed(s, *e, "f"); out.Said != "" {
		t.Errorf("the key said %q with no port behind it", out.Said)
	}
}
