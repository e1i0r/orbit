package repo

// What a change reaches beyond the files it touched: the file that usually
// comes along and did not, and what the tests among those say they hold.

import (
	"path/filepath"
	"strings"
	"testing"
)

// coupled is a repository whose history always changes pricing and its test
// together, and a worktree that has changed only pricing.
func coupled(t *testing.T) (Repo, string) {
	t.Helper()

	dir := t.TempDir()
	r := Repo{Path: dir, Name: "sums", Base: "main"}

	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "a@b.c"},
		{"config", "user.name", "a"},
	} {
		if _, err := git(dir, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}

	// Ten commits that touch the two files together: enough history for a
	// pattern rather than a coincidence.
	for i := range 10 {
		write(t, filepath.Join(dir, "pricing.go"), strings.Repeat("// price\n", i+1))
		write(t, filepath.Join(dir, "pricing_test.go"),
			"package sums\n\nfunc TestARefundIsNeverNegative(t *testing.T) {}\n"+
				strings.Repeat("// held\n", i+1))

		if _, err := git(dir, "add", "-A"); err != nil {
			t.Fatalf("add: %v", err)
		}

		if _, err := git(dir, "commit", "-m", "together"); err != nil {
			t.Fatalf("commit: %v", err)
		}
	}

	wt := filepath.Join(t.TempDir(), "work")
	if err := r.AddWorktree(wt, "orbit/task"); err != nil {
		t.Fatalf("worktree: %v", err)
	}

	// The work touches one of the pair and not the other.
	write(t, filepath.Join(wt, "pricing.go"), "// a new price\n")

	return r, wt
}

// TestTheFileThatUsuallyComesAlongAndDidNotIsTheReading. The diff says what
// the work touched; this says what it left behind.
func TestTheFileThatUsuallyComesAlongAndDidNotIsTheReading(t *testing.T) {
	r, wt := coupled(t)

	got, err := r.Impact(wt)
	if err != nil {
		t.Fatalf("Impact: %v", err)
	}

	if len(got.Changed) != 1 || got.Changed[0] != "pricing.go" {
		t.Fatalf("the work touched %v", got.Changed)
	}

	if got.Commits == 0 {
		t.Error("the reading says it read no history at all")
	}

	if !got.Anything() {
		t.Fatalf("the reading found nothing worth an eye: %+v", got)
	}

	if got.Coupled[0].File != "pricing_test.go" {
		t.Errorf("the strongest warning is %+v, want the test that always comes along", got.Coupled[0])
	}

	// And what that test says it holds is read out of its own names: the
	// cheapest specification any repository has.
	if len(got.Contracts) == 0 {
		t.Fatalf("the test that was left out says nothing: %+v", got)
	}

	if !strings.Contains(got.Contracts[0].Says, "refund") {
		t.Errorf("the contract reads %q", got.Contracts[0].Says)
	}
}

// TestAWorktreeWithNothingChangedReachesNothing, which is not a failure: it
// is a run that has not written anything yet.
func TestAWorktreeWithNothingChangedReachesNothing(t *testing.T) {
	r, wt := changed(t)

	got, err := r.Impact(wt)
	if err != nil {
		t.Fatalf("Impact: %v", err)
	}

	if got.Anything() || len(got.Changed) != 0 {
		t.Errorf("an untouched worktree reaches %+v", got)
	}
}

// TestATestFileIsRecognisedByEveryConventionThatCoversMostRepositories.
func TestATestFileIsRecognisedByEveryConventionThatCoversMostRepositories(t *testing.T) {
	for _, c := range []struct {
		path string
		want bool
	}{
		{"internal/ui/rows_test.go", true},
		{"tests/test_pricing.py", true},
		{"spec/pricing_spec.rb", true},
		{"src/pricing.test.ts", true},
		{"internal/ui/rows.go", false},
		{"README.md", false},
	} {
		if got := looksLikeTests(c.path); got != c.want {
			t.Errorf("looksLikeTests(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// TestAFileTheHistoryKnowsAndTheCheckoutDoesNotIsNotAContract. Somebody
// deleted it; that is not a fault and not something it holds.
func TestAFileTheHistoryKnowsAndTheCheckoutDoesNotIsNotAContract(t *testing.T) {
	r, _ := coupled(t)

	got := r.contracts([]Coupled{{File: "gone_test.go", Times: 9, Of: 10}})
	if len(got) != 0 {
		t.Errorf("a file that is not there says %+v", got)
	}
}

// TestAPlaceAReviewCommentIsAboutIsSaidAsAPersonWouldSayIt.
func TestAPlaceAReviewCommentIsAboutIsSaidAsAPersonWouldSayIt(t *testing.T) {
	for _, c := range []struct {
		in   Comment
		want string
	}{
		{Comment{}, "the pull request"},
		{Comment{Path: "internal/ui/rows.go"}, "internal/ui/rows.go"},
		{Comment{Path: "internal/ui/rows.go", Line: 42}, "internal/ui/rows.go:42"},
	} {
		if got := c.in.Where(); got != c.want {
			t.Errorf("Where(%+v) = %q, want %q", c.in, got, c.want)
		}
	}
}
