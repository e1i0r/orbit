package repo

// What else usually changes when this file changes.

import (
	"fmt"
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

// followedIn is n commits that all touched one file, where the first `along`
// of them also touched a second.
func followedIn(n, along int, one, other string) [][]string {
	var out [][]string

	for i := range n {
		files := []string{one}
		if i < along {
			files = append(files, other)
		}

		out = append(out, files)
	}

	return out
}

// TestAWarningIsGivenAtTheFloorAndNotBelowIt.
//
// Two numbers decide it: how often the two were seen together, and how much
// of the time. Both are somebody's judgement, so both have to be read at
// their own edge — one commit stricter and the weakest real pattern in the
// repository is never mentioned, which is the pattern a reader most needs
// telling about because they have not noticed it either.
func TestAWarningIsGivenAtTheFloorAndNotBelowIt(t *testing.T) {
	// Seen together exactly the number of times, in exactly half the
	// commits that touched the first file.
	atBoth := coChanged(followedIn(floorTimes*2, floorTimes, "a.go", "b.go"), []string{"a.go"})
	if len(atBoth) != 1 || atBoth[0].File != "b.go" {
		t.Fatalf("a pattern at both floors reads as %+v", atBoth)
	}

	if atBoth[0].Times != floorTimes || atBoth[0].Ratio() != floorRatio {
		t.Errorf("it reads as %d of %d", atBoth[0].Times, atBoth[0].Of)
	}

	// One commit fewer together is under the floor of times.
	rare := coChanged(followedIn(floorTimes*2, floorTimes-1, "a.go", "c.go"), []string{"a.go"})
	if len(rare) != 0 {
		t.Errorf("a pattern seen %d times reads as %+v", floorTimes-1, rare)
	}

	// And the same number of times over one commit more is under the floor
	// of how much of the time.
	thin := coChanged(followedIn(floorTimes*2+1, floorTimes, "a.go", "d.go"), []string{"a.go"})
	if len(thin) != 0 {
		t.Errorf("a pattern holding %d of %d reads as %+v", floorTimes, floorTimes*2+1, thin)
	}
}

// TestTheStrongestPatternIsSaidFirst.
//
// A reader looks at the top of this list and stops. Ordered by anything
// else, the file they most need telling about sits under two they already
// knew — and where two hold as often as each other, the one seen more times
// is the one with more behind it.
func TestTheStrongestPatternIsSaidFirst(t *testing.T) {
	// close.go came along every time a.go changed; half.go half the time.
	var commits [][]string

	for i := range 8 {
		files := []string{"a.go", "close.go"}
		if i < 4 {
			files = append(files, "half.go")
		}

		commits = append(commits, files)
	}

	got := coChanged(commits, []string{"a.go"})
	if len(got) != 2 {
		t.Fatalf("two patterns read as %+v", got)
	}

	if got[0].File != "close.go" {
		t.Errorf("the list opens with %q, want the pattern that holds most of the time", got[0].File)
	}

	// Two that hold as often as each other — both every time — told apart
	// by how much is behind them. They follow different files, which is the
	// only way one ratio can stand on more commits than another.
	var tied [][]string

	tied = append(tied, history(4, "b.go", "few.go")...)
	tied = append(tied, history(9, "c.go", "many.go")...)

	both := coChanged(tied, []string{"b.go", "c.go"})
	if len(both) != 2 {
		t.Fatalf("two patterns that hold every time read as %+v", both)
	}

	if both[0].Ratio() != both[1].Ratio() {
		t.Fatalf("the two do not hold as often as each other: %+v", both)
	}

	if both[0].File != "many.go" {
		t.Errorf("the list opens with %q, want the one seen more times", both[0].File)
	}
}

// TestOnlySoManyWarningsAreWorthReading, because a list of forty is a list
// nobody reads to the end — and the ones past the cap are the weakest of
// them, which is why they are the ones to drop.
func TestOnlySoManyWarningsAreWorthReading(t *testing.T) {
	// Every one of them came along every time, so only the cap decides how
	// many come back.
	crowd := func(n int) [][]string {
		files := []string{"a.go"}
		for i := range n {
			files = append(files, fmt.Sprintf("f%02d.go", i))
		}

		return history(floorTimes, files...)
	}

	if got := coChanged(crowd(mostCoupled+3), []string{"a.go"}); len(got) != mostCoupled {
		t.Errorf("%d patterns came back, want the %d worth reading", len(got), mostCoupled)
	}

	// And exactly the cap's worth is not cut.
	if all := coChanged(crowd(mostCoupled), []string{"a.go"}); len(all) != mostCoupled {
		t.Errorf("exactly %d patterns came back as %d", mostCoupled, len(all))
	}
}
