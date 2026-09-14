package learn

// What a rule has been through, written down and read back.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// TestKeepingARuleIsTheFirstThingThatHappenedToIt, which is where every
// rule's history starts and what everything after it hangs from.
func TestKeepingARuleIsTheFirstThingThatHappenedToIt(t *testing.T) {
	s := root(t)
	repo := aCheckout(t)

	one := waitingFromARun(t, s, repo, "internal/db", "every query is a named constant")

	if err := Keep(s, one.At, one.Text, "", Place{}); err != nil {
		t.Fatalf("keep: %v", err)
	}

	kept := filed(t, s, repo)[0]
	if kept.ID == "" {
		t.Fatal("a rule Orbit wrote has no name")
	}

	turns, err := History(s, kept.ID)
	if err != nil {
		t.Fatalf("read what happened to it: %v", err)
	}

	if len(turns) != 1 || turns[0].What != Written {
		t.Fatalf("what happened to it reads as %+v", turns)
	}

	if turns[0].By != Operator {
		t.Errorf("it says %q kept it", turns[0].By)
	}
}

// TestWhatChangedIsWrittenDownOnePieceAtATime.
//
// Reworded and moved somewhere else are two different things to have done.
// They are asked about separately when somebody is deciding whether to keep
// a rule, and an edit that did both is an edit that did both.
func TestWhatChangedIsWrittenDownOnePieceAtATime(t *testing.T) {
	s := root(t)
	repo := aCheckout(t)

	was := knowledge.Fact{
		ID: "abc12345", Source: knowledge.Human, Phrase: "amounts are cents",
		Scope: knowledge.Scope{Kind: knowledge.Repo, Repo: repo},
	}

	now := was
	now.Phrase = "every amount in this project is in cents"
	now.Scope = knowledge.Scope{Kind: knowledge.Dir, Repo: repo, Path: "internal/db"}
	now.State = knowledge.Off

	if err := Changed(s, was, now, Turn{By: Operator}); err != nil {
		t.Fatalf("write down what changed: %v", err)
	}

	turns, err := History(s, was.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(turns) != 3 {
		t.Fatalf("one edit that did three things reads as %d turns: %+v", len(turns), turns)
	}

	what := map[string]string{}
	for _, one := range turns {
		what[one.What] = one.Was
	}

	// The old sentence is the whole reason the rewording is worth keeping:
	// without it the row says something changed and not what.
	if what[Reworded] != "amounts are cents" {
		t.Errorf("the rewording remembers %q", what[Reworded])
	}

	if what[MovedTo] != "the whole checkout" {
		t.Errorf("the move remembers %q", what[MovedTo])
	}

	if _, off := what[TurnedOff]; !off {
		t.Errorf("switching it off is not in %+v", turns)
	}
}

// TestAnEditThatChangedNothingWritesNothing, so that opening a rule and
// closing it again does not fill its history with rows saying so.
func TestAnEditThatChangedNothingWritesNothing(t *testing.T) {
	s := root(t)

	same := knowledge.Fact{
		ID: "abc12345", Source: knowledge.Human, Phrase: "amounts are cents",
		Scope: knowledge.Scope{Kind: knowledge.General},
	}

	if err := Changed(s, same, same, Turn{By: Operator}); err != nil {
		t.Fatalf("write down what changed: %v", err)
	}

	turns, err := History(s, same.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(turns) != 0 {
		t.Errorf("an edit that changed nothing wrote %+v", turns)
	}
}

// TestARuleWithNoNameWritesNoHistoryAndIsNotAnError.
//
// A rule somebody wrote by hand has no name until Orbit writes it, which is
// allowed — and refusing the edit they actually asked for, because there was
// nowhere to file a note about it, would be the wrong half to protect.
func TestARuleWithNoNameWritesNoHistoryAndIsNotAnError(t *testing.T) {
	s := root(t)

	byHand := knowledge.Fact{
		Source: knowledge.Human, Phrase: "never push on a Friday",
		Scope: knowledge.Scope{Kind: knowledge.General},
	}

	now := byHand
	now.Phrase = "never push on a Friday afternoon"

	if err := Changed(s, byHand, now, Turn{By: Operator}); err != nil {
		t.Errorf("editing a rule with no name was refused: %v", err)
	}
}

// TestARuleNobodyHasTouchedHasNoHistory, and that is an answer rather than a
// failure: it is what every rule looks like the day it is kept.
func TestARuleNobodyHasTouchedHasNoHistory(t *testing.T) {
	s := root(t)

	turns, err := History(s, "nothing12")
	if err != nil {
		t.Fatalf("reading the history of a rule nobody has touched: %v", err)
	}

	if len(turns) != 0 {
		t.Errorf("it reads as %+v", turns)
	}
}
