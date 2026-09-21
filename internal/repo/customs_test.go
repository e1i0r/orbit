package repo

// What the history says a project does.

import (
	"fmt"
	"strings"
	"testing"
)

// commitsWith is a history, one commit per list of files.
func commitsWith(commits ...[]string) [][]string { return commits }

// TestTestsTravelIsMeasuredAndNotAssumed.
//
// The documents say what somebody wanted; this says what the team kept doing.
// A project whose changes come with tests and one whose changes do not are
// both answered, because the second is what says a written rule is no longer
// true.
func TestTestsTravelIsMeasuredAndNotAssumed(t *testing.T) {
	var with, without [][]string

	for range enoughCommits {
		with = append(with, []string{"internal/db/read.go", "internal/db/read_test.go"})
		without = append(without, []string{"internal/db/read.go"})
	}

	held := testsTravel(with)
	if len(held) != 1 || !held[0].Holds() {
		t.Fatalf("a project that always tests reads as %+v", held)
	}

	if held[0].Where != "internal" || held[0].Times != held[0].Of {
		t.Errorf("it reads as %+v", held[0])
	}

	missing := testsTravel(without)
	if len(missing) != 1 || missing[0].Holds() {
		t.Fatalf("a project that never tests reads as %+v", missing)
	}
}

// TestAHistoryTooShortToReadSaysNothing, which is what a repository somebody
// started last week is — and that is an answer, not an empty list.
func TestAHistoryTooShortToReadSaysNothing(t *testing.T) {
	short := commitsWith([]string{"internal/db/read.go"}, []string{"internal/db/write.go"})

	if got := testsTravel(short); len(got) != 0 {
		t.Errorf("two commits were read as %+v", got)
	}
}

// TestACommitOfOnlyTestsSaysNothingAboutWhetherTestsTravel, because there was
// no code in it for a test to travel with.
func TestACommitOfOnlyTestsSaysNothingAboutWhetherTestsTravel(t *testing.T) {
	var commits [][]string

	for range enoughCommits {
		commits = append(commits, []string{"internal/db/read_test.go"})
	}

	if got := testsTravel(commits); len(got) != 0 {
		t.Errorf("a history of nothing but tests reads as %+v", got)
	}
}

// TestACrowdedCommitIsNotEvidence. A squashed merge, a formatting pass or a
// directory rename touches everything and says nothing about which of those
// belong together.
func TestACrowdedCommitIsNotEvidence(t *testing.T) {
	huge := make([]string, 0, crowdedCommit+1)
	for i := range crowdedCommit + 1 {
		huge = append(huge, "internal/db/"+string(rune('a'+i%26))+".go")
	}

	var commits [][]string
	for range enoughCommits {
		commits = append(commits, huge)
	}

	if got := testsTravel(commits); len(got) != 0 {
		t.Errorf("a history of squashed merges reads as %+v", got)
	}
}

// TestTheShapesAMessageCanKeep, which are the two common enough to be worth
// looking for. A project keeping one nobody put here reads as keeping none,
// which offers no rule rather than a wrong one.
func TestTheShapesAMessageCanKeep(t *testing.T) {
	for _, one := range []struct {
		subject      string
		is, ticketed bool
	}{
		{"fix: the fuzz tests hang", true, false},
		{"feat(ui): a ringed mark over version", true, false},
		{"PAY-105: skipping a gate", false, true},
		{"ABC-12 tidy the reader", false, true},
		{"Handle an error once", false, false},
		{"Merge pull request #157", false, false},
		// The letters at the ends of the alphabet are letters. A reading
		// that stopped one short of them would call `zip:` no shape and
		// `AZ-1` no ticket, and a project that keeps either would read as
		// keeping nothing.
		{"az: the ends of the alphabet", true, false},
		{"AZ-90: the ends of the alphabet", false, true},
		// And the characters beside them are not. They sit either side of
		// the letters in ASCII, which is what the reading is really about.
		{"a`z: not a word", false, false},
		{"a{z: not a word", false, false},
		{"A[Z-1: not a key", false, false},
		{"A@Z-1: not a key", false, false},
		// The digits at the ends of their own run, and the characters
		// beside them.
		{"PAY-09: the ends of the digits", false, true},
		{"PAY-0/9: not a number", false, false},
		{"PAY-0:9 not a number", false, false},
		// A head of exactly the length one may be is one; a character more
		// is a sentence with a colon in it, not a shape.
		{strings.Repeat("a", 24) + ": still a shape", true, false},
		{strings.Repeat("a", 25) + ": a sentence", false, false},
	} {
		if got := conventional(one.subject); got != one.is {
			t.Errorf("%q reads as conventional=%v", one.subject, got)
		}

		if got := ticketed(one.subject); got != one.ticketed {
			t.Errorf("%q reads as ticketed=%v", one.subject, got)
		}
	}
}

// TestATestIsRecognisedInTheSpellingsLanguagesUse, and a path in none of them
// reads as not a test — which offers no rule rather than a wrong one.
func TestATestIsRecognisedInTheSpellingsLanguagesUse(t *testing.T) {
	for _, one := range []string{
		"internal/db/read_test.go", "src/read.test.ts", "spec/read_spec.rb",
		"tests/test_read.py", "app/test/read.java",
	} {
		if !aTest(one) {
			t.Errorf("%q does not read as a test", one)
		}
	}

	for _, one := range []string{"internal/db/read.go", "README.md", "latest/thing.go"} {
		if aTest(one) {
			t.Errorf("%q reads as a test", one)
		}
	}
}

// TestACommitExactlyAsCrowdedAsOneMayBeIsStillEvidence.
//
// The number is where a commit stops being one change and starts being a
// squashed merge or a rename, and it is somebody's judgement — so it has to
// be read at its own edge. One file too strict and the largest ordinary
// change in the repository is thrown away, with nothing anywhere saying so.
func TestACommitExactlyAsCrowdedAsOneMayBeIsStillEvidence(t *testing.T) {
	filled := func(n int) []string {
		files := make([]string, 0, n)
		for i := range n - 1 {
			files = append(files, fmt.Sprintf("internal/db/f%02d.go", i))
		}

		return append(files, "internal/db/read_test.go")
	}

	var atTheEdge, over [][]string

	for range enoughCommits {
		atTheEdge = append(atTheEdge, filled(crowdedCommit))
		over = append(over, filled(crowdedCommit+1))
	}

	held := testsTravel(atTheEdge)
	if len(held) != 1 || !held[0].Holds() {
		t.Errorf("commits of exactly %d files read as %+v", crowdedCommit, held)
	}

	if got := testsTravel(over); len(got) != 0 {
		t.Errorf("commits of %d files read as %+v", crowdedCommit+1, got)
	}
}

// TestWhatHoldsOftenEnoughToBeACustom.
//
// Four in five: a rule the team keeps most of the time is still the rule,
// and demanding every commit would find nothing in any repository with
// people in it. A reading one commit stricter would answer that a project
// keeping its own rule four times in five does not keep it.
func TestWhatHoldsOftenEnoughToBeACustom(t *testing.T) {
	for _, one := range []struct {
		times, of int
		holds     bool
	}{
		{times: 8, of: 10, holds: true},
		{times: 79, of: 100, holds: false},
		{times: 10, of: 10, holds: true},
		{times: 0, of: 10, holds: false},
		// Nothing was ever measured, so nothing holds — rather than a
		// division by nothing reading as a custom the project always keeps.
		{times: 0, of: 0, holds: false},
		{times: 3, of: 0, holds: false},
	} {
		c := Custom{Times: one.times, Of: one.of}
		if got := c.Holds(); got != one.holds {
			t.Errorf("%d of %d holds: %v, want %v (ratio %.3f)", one.times, one.of, got, one.holds, c.Ratio())
		}
	}
}
