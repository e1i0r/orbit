package learn

// Taking a rule off the disk, and the one condition under which that is
// allowed.
//
// Switching a rule off and losing the record that it existed are different
// things, and this package is where the difference is kept. A rule somebody
// decided and stopped wanting is switched off: it stays where it is, nothing
// is told it, and what it put you through is still readable. A rule that
// never should have been written — a duplicate of one already there, a test
// rule, one that came out wrong — is not that, and leaving it in the list
// for ever is a screen that fills with things nobody meant.
//
// What tells the two apart is the record, not an opinion: a rule it has
// anything to say about has a history, and a rule it has nothing to say
// about has none to lose.

import (
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
)

// DidSomethingError is a rule that has a history, refusing to be forgotten.
//
// A type and not a sentence, because the sentence a reader sees is in their
// own language and this package may not reach the catalogue. What it carries
// is the one thing the sentence needs: the first thing the rule did, so that
// "no" is not the whole of the answer.
type DidSomethingError struct{ What string }

func (e DidSomethingError) Error() string {
	return fmt.Sprintf("the rule has a history: it was %s", e.What)
}

// Did says whether a rule has ever done anything, and what it did first.
//
// Being written down is not doing anything: it is how every rule starts, and
// a rule whose whole story is that somebody wrote it is exactly the one worth
// being able to take back. Everything else counts — refusing work, being
// paused, being reworded, being moved, being switched off and on again.
//
// A rule with no name has no history by construction: names are coined when
// a rule is saved, and nothing in the record can be about one that was never
// given one.
func Did(s *store.Store, f knowledge.Rule) (bool, string, error) {
	if f.ID == "" {
		return false, "", nil
	}

	turns, err := History(s, f.ID)
	if err != nil {
		return false, "", err
	}

	for _, t := range turns {
		if t.What != Written {
			return true, t.What, nil
		}
	}

	return false, "", nil
}

// Forget takes a rule that never did anything off the disk, and refuses
// one that did.
//
// Not Drop, which this package already uses for the other half of the same
// screen: dropping is saying a sentence in the tray was never a rule, and it
// removes nothing. Two words because they are two things.
//
// The refusal names what the rule did, because "no" on its own leaves
// somebody wondering whether the rule is special or the button is broken.
//
// The turn is written before the file goes, for the reason Keep writes the
// fact before it clears the tray: the other order loses the row when the
// write fails, and a rule gone from the disk with nothing in the record
// saying so is the one outcome worse than not being able to remove it.
func Forget(s *store.Store, f knowledge.Rule, by string) error {
	did, what, err := Did(s, f)
	if err != nil {
		return err
	}

	if did {
		return DidSomethingError{What: what}
	}

	gone := Turn{Rule: f.ID, At: time.Now().UTC(), What: Forgotten, By: by, Was: f.Phrase}
	if err := Happened(s, gone); err != nil {
		return err
	}

	return knowledge.NewStore(s.Root()).Delete(f)
}
