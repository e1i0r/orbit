package repo

// What else usually changes when this file changes.

import (
	"strings"
	"testing"
)

// history is n commits that touched these files together.
func history(n int, files ...string) [][]string {
	var out [][]string
	for range n {
		out = append(out, files)
	}

	return out
}

// TestTheFileThatAlwaysComesAlongIsTheWarning. The diff says what the work
// touched; this says what it forgot.
func TestTheFileThatAlwaysComesAlongIsTheWarning(t *testing.T) {
	commits := append(history(9, "pricing.py", "invoice.py"), history(1, "pricing.py")...)

	got := coChanged(commits, []string{"pricing.py"})
	if len(got) != 1 {
		t.Fatalf("got %d warnings: %+v", len(got), got)
	}

	if got[0].File != "invoice.py" || got[0].Times != 9 || got[0].Of != 10 {
		t.Errorf("the warning is %+v", got[0])
	}

	if r := got[0].Ratio(); r < 0.89 || r > 0.91 {
		t.Errorf("it follows %.2f of the time, want 0.9", r)
	}
}

// TestAFileTheWorkAlreadyTouchedIsNotAWarning, because the question is what
// was left out.
func TestAFileTheWorkAlreadyTouchedIsNotAWarning(t *testing.T) {
	commits := history(9, "pricing.py", "invoice.py")

	if got := coChanged(commits, []string{"pricing.py", "invoice.py"}); len(got) != 0 {
		t.Errorf("a file that was changed is warned about: %+v", got)
	}
}

// TestTwoOutOfTwoIsNotAPattern. A hundred percent of a pair of commits is
// how a list of warnings fills with noise.
func TestTwoOutOfTwoIsNotAPattern(t *testing.T) {
	commits := history(2, "pricing.py", "unrelated.py")

	if got := coChanged(commits, []string{"pricing.py"}); len(got) != 0 {
		t.Errorf("two commits made a rule: %+v", got)
	}
}

// TestACrowdedCommitSaysNothingAboutWhatBelongsTogether: a squashed merge or
// a formatting pass touches everything, and everything is not coupled.
func TestACrowdedCommitSaysNothingAboutWhatBelongsTogether(t *testing.T) {
	var everything []string
	for i := range crowdedCommit + 1 {
		everything = append(everything, "file"+string(rune('a'+i%26))+string(rune('0'+i/26))+".go")
	}

	everything[0] = "pricing.py"

	commits := append(history(8, everything...), history(8, "pricing.py")...)

	if got := coChanged(commits, []string{"pricing.py"}); len(got) != 0 {
		t.Errorf("a crowded commit coupled everything to everything: %+v", got)
	}
}

// TestAFileIsSaidOnceAgainstWhatItFollowsMostClosely, not once per changed
// file it happens to appear beside.
func TestAFileIsSaidOnceAgainstWhatItFollowsMostClosely(t *testing.T) {
	commits := append(history(9, "pricing.py", "invoice.py"), history(9, "fees.py", "invoice.py", "other.py")...)
	commits = append(commits, history(9, "fees.py")...)

	got := coChanged(commits, []string{"pricing.py", "fees.py"})

	seen := map[string]int{}
	for _, c := range got {
		seen[c.File]++
	}

	if seen["invoice.py"] != 1 {
		t.Errorf("invoice.py is warned about %d times: %+v", seen["invoice.py"], got)
	}

	for _, c := range got {
		if c.File == "invoice.py" && c.With != "pricing.py" {
			t.Errorf("invoice.py is said against %q, want the one it follows most closely", c.With)
		}
	}
}

// TestTheHistoryIsReadCommitByCommit, and a commit message with a blank line
// in it does not become a list of files.
func TestTheHistoryIsReadCommitByCommit(t *testing.T) {
	out := strings.Join([]string{
		"\x00abc123",
		"fix the thing",
		"",
		"and explain why",
		"pricing.py",
		"invoice.py",
		"\x00def456",
		"another",
		"pricing.py",
	}, "\n")

	commits := commitsOf(out)
	if len(commits) != 2 {
		t.Fatalf("read %d commits: %+v", len(commits), commits)
	}

	// The message lines are in the block too — what matters is that the
	// second commit is separate and the files of the first are all there.
	if !contains(commits[0], "pricing.py") || !contains(commits[0], "invoice.py") {
		t.Errorf("the first commit is %+v", commits[0])
	}

	if !contains(commits[1], "pricing.py") {
		t.Errorf("the second commit is %+v", commits[1])
	}
}

func contains(all []string, one string) bool {
	for _, s := range all {
		if s == one {
			return true
		}
	}

	return false
}
