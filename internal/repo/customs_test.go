package repo

// What the history says a project does.

import "testing"

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
		{"FRA-105: skipping a gate", false, true},
		{"ABC-12 tidy the reader", false, true},
		{"Handle an error once", false, false},
		{"Merge pull request #157", false, false},
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
