package learn

// What may be taken off the disk, and what may not.

import (
	"errors"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
)

// wrote puts one rule on disk and answers it with the name it was given.
func wrote(t *testing.T, s *store.Store, phrase string) knowledge.Rule {
	t.Helper()

	f := knowledge.Rule{
		Phrase: phrase,
		Scope:  knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human,
		At:     time.Now().UTC(),
	}

	where, err := knowledge.NewStore(s.Root()).Save(f)
	if err != nil {
		t.Fatalf("save the rule: %v", err)
	}

	for _, one := range read(t, s) {
		if one.Phrase == phrase {
			return one
		}
	}

	t.Fatalf("the rule saved at %q did not read back", where)

	return knowledge.Rule{}
}

// read is every rule on disk.
func read(t *testing.T, s *store.Store) []knowledge.Rule {
	t.Helper()

	all, err := knowledge.NewStore(s.Root()).Load("")
	if err != nil {
		t.Fatalf("read the rules: %v", err)
	}

	return all
}

// TestARuleThatDidNothingCanBeForgotten. A duplicate of one already there, a
// test rule, one that came out wrong: leaving those in the list for ever is
// a screen that fills with things nobody meant.
func TestARuleThatDidNothingCanBeForgotten(t *testing.T) {
	s := root(t)
	f := wrote(t, s, "never log a card number")

	if err := Forget(s, f, Operator); err != nil {
		t.Fatalf("a rule that did nothing was not forgotten: %v", err)
	}

	if all := read(t, s); len(all) != 0 {
		t.Errorf("the rule is still on disk: %+v", all)
	}
}

// TestForgettingIsWrittenDown. Something vanishing with no trace at all is
// the only outcome worse than not being able to remove it.
func TestForgettingIsWrittenDown(t *testing.T) {
	s := root(t)
	f := wrote(t, s, "never log a card number")

	if err := Forget(s, f, Operator); err != nil {
		t.Fatalf("forget: %v", err)
	}

	turns, err := History(s, f.ID)
	if err != nil {
		t.Fatalf("read the history back: %v", err)
	}

	var gone *Turn

	for i, one := range turns {
		if one.What == Forgotten {
			gone = &turns[i]
		}
	}

	if gone == nil {
		t.Fatalf("nothing in the record says the rule was forgotten: %+v", turns)
	}

	// The sentence travels with the row, because the file that held it is
	// gone and the name on its own says nothing to a reader.
	if gone.Was != "never log a card number" {
		t.Errorf("the row does not carry what the rule said: %q", gone.Was)
	}
}

// TestARuleWithAHistoryIsRefused, and the refusal says what it did — "no" on
// its own leaves somebody wondering whether the rule is special or the
// button is broken.
func TestARuleWithAHistoryIsRefused(t *testing.T) {
	s := root(t)
	f := wrote(t, s, "coverage stays above 90%")

	// Paused, which is one of the things that counts as a history.
	was := Turn{Rule: f.ID, At: time.Now().UTC(), What: Paused, By: Operator, Was: "later"}
	if err := Happened(s, was); err != nil {
		t.Fatalf("write the pause down: %v", err)
	}

	err := Forget(s, f, Operator)

	var did DidSomethingError
	if !errors.As(err, &did) {
		t.Fatalf("a rule with a history was forgotten anyway: %v", err)
	}

	if did.What != Paused {
		t.Errorf("the refusal says it was %q, want the thing it actually did", did.What)
	}

	if all := read(t, s); len(all) != 1 {
		t.Errorf("a refused forget removed the rule anyway: %+v", all)
	}
}

// TestBeingWrittenDownIsNotAHistory. Every rule starts by being kept, so a
// rule whose whole story is that somebody wrote it is exactly the one worth
// being able to take back.
func TestBeingWrittenDownIsNotAHistory(t *testing.T) {
	s := root(t)
	f := wrote(t, s, "never log a card number")

	kept := Turn{Rule: f.ID, At: time.Now().UTC(), What: Written, By: Operator}
	if err := Happened(s, kept); err != nil {
		t.Fatalf("write the keeping down: %v", err)
	}

	did, what, err := Did(s, f)
	if err != nil {
		t.Fatalf("ask whether it did anything: %v", err)
	}

	if did {
		t.Errorf("being written down counted as a history: %q", what)
	}
}
