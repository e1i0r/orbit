package knowledge

// Taking a rule off the disk, and the one question every surface folds three
// fields into.
//
// Delete and Standing both read zero. Delete is the only thing here that
// loses something; Standing is what the cockpit and the browser both ask
// about a rule they are looking at, so if it were wrong they would be wrong
// together and agree with each other while doing it.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestARuleIsTakenOffTheDiskAndNotOnlyOutOfTheListing. The file is the rule:
// one left behind is still read, still told to the next run, and still
// refusing work.
func TestARuleIsTakenOffTheDiskAndNotOnlyOutOfTheListing(t *testing.T) {
	s, repo := aRepo(t)

	f := Rule{Scope: Scope{Kind: Repo, Repo: repo}, Source: Human, Phrase: "amounts are cents"}

	where, err := s.Save(f)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	written := only(t, s, repo)

	if err := s.Delete(written); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := os.Stat(where); !os.IsNotExist(err) {
		t.Errorf("the file is still at %q: %v", where, err)
	}

	left, err := s.Load(repo)
	if err != nil {
		t.Fatalf("read them back: %v", err)
	}

	if len(left) != 0 {
		t.Errorf("the rule is still listed: %+v", left)
	}
}

// TestARuleAlreadyGoneIsNotAFailure. Two windows on one workspace can both
// be looking at a rule when one of them removes it, and the second asking
// for something that has already happened has got the outcome it wanted.
func TestARuleAlreadyGoneIsNotAFailure(t *testing.T) {
	s, repo := aRepo(t)

	f := Rule{Scope: Scope{Kind: Repo, Repo: repo}, Source: Human, Phrase: "amounts are cents"}
	if _, err := s.Save(f); err != nil {
		t.Fatalf("save: %v", err)
	}

	written := only(t, s, repo)

	for range 2 {
		if err := s.Delete(written); err != nil {
			t.Errorf("removing a rule that is already gone failed: %v", err)
		}
	}
}

// TestARuleNothingHasEverWrittenIsRemovedByWhereItWouldHaveBeen. A rule that
// arrived from a screen rather than from the disk carries no path of its
// own, so the place is worked out from its own fields — the same way Save
// works it out.
func TestARuleNothingHasEverWrittenIsRemovedByWhereItWouldHaveBeen(t *testing.T) {
	s, repo := aRepo(t)

	f := Rule{
		ID: "aaaa1111", Scope: Scope{Kind: Repo, Repo: repo}, Source: Human,
		Phrase: "amounts are cents",
	}

	if err := s.Delete(f); err != nil {
		t.Errorf("removing a rule that was never written failed: %v", err)
	}

	// And one that is there goes, addressed the same way: the file Save
	// wrote is the file Delete removes.
	where, err := s.Save(f)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	f.from = ""

	if err := s.Delete(f); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := os.Stat(where); !os.IsNotExist(err) {
		t.Errorf("the file is still at %q", filepath.Base(where))
	}
}

// TestWhatARuleIsDoingIsOneAnswerAndNotThreeFields.
//
// Whether it is waiting to be decided about, whether somebody stopped it
// applying, and what it would do if it were applying are three questions in
// the record. Folded here, the cockpit and the browser cannot disagree about
// a rule they are both looking at — and the five answers are exclusive, so
// the order they are asked in is part of the answer.
func TestWhatARuleIsDoingIsOneAnswerAndNotThreeFields(t *testing.T) {
	cases := []struct {
		name string
		rule Rule
		want Standing
	}{
		{
			"waiting outranks everything else",
			Rule{Source: Human, Phrase: "x", Review: true, State: Paused, Check: "make check", Stops: true},
			Waiting,
		},
		{
			"paused outranks what it would have done",
			Rule{Source: Human, Phrase: "x", State: Paused, Check: "make check", Stops: true},
			Stopped,
		},
		{
			"switched off is told to nobody",
			Rule{Source: Human, Phrase: "x", State: Off, Check: "make check", Stops: true},
			Silent,
		},
		{
			"a gate behind it blocks the work",
			Rule{Source: Human, Phrase: "x", Check: "make check", Stops: true},
			Blocks,
		},
		{
			"a command with nothing standing behind it only says its sentence",
			Rule{Source: Human, Phrase: "x", Check: "make check"},
			Says,
		},
		{
			"and most of them only say their sentence",
			Rule{Source: Human, Phrase: "x"},
			Says,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.rule.Standing(); got != c.want {
				t.Errorf("it reads as %v, want %v", got, c.want)
			}
		})
	}
}

// TestANameIsCoinedBeforeTheRuleIsWritten. The caller that has to write down
// what happened to a rule in the same breath as writing the rule cannot read
// the name back out of a path, and reading the file again would be a second
// answer to a question already settled.
func TestANameIsCoinedBeforeTheRuleIsWritten(t *testing.T) {
	seen := map[string]bool{}

	for range 200 {
		name := Name()
		if name == "" {
			t.Fatal("a rule was named nothing")
		}

		if seen[name] {
			t.Fatalf("%q was coined twice", name)
		}

		seen[name] = true
	}
}
