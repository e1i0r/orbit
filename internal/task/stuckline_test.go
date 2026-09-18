package task

// What a reader is told when every attempt at a phase was refused.

import (
	"strings"
	"testing"
)

// TestEveryAttemptIsNumberedFromTheFirst.
//
// The reader is looking at why a phase could not get past its gate, and the
// numbers are how they follow it: the first attempt, then what the second
// did differently. Counted from anywhere else, the list reads as though a
// run nobody can see happened before the one they are reading about.
func TestEveryAttemptIsNumberedFromTheFirst(t *testing.T) {
	got := stuckLine("test", 2, []gateRefusal{
		{Gate: "go test ./...", Exit: 1, Output: "one test failed"},
		{Gate: "go test ./...", Exit: 1, Output: "the same test failed"},
	})

	for _, want := range []string{"Attempt 1 — gate", "Attempt 2 — gate"} {
		if !strings.Contains(got, want) {
			t.Errorf("the account of a stuck phase reads:\n%s\nwant it to carry %q", got, want)
		}
	}

	if strings.Contains(got, "Attempt 0") {
		t.Errorf("the attempts are numbered from zero:\n%s", got)
	}
}

// TestAGatesOutputIsCutFromTheFrontAndSaysSo.
//
// The tail rather than the head: a build says what is wrong at the end, and
// the first twenty lines of a test run are the tests that passed. Output
// that fits is handed over untouched, because a note about lines nobody
// removed is a reader going to look for them.
func TestAGatesOutputIsCutFromTheFrontAndSaysSo(t *testing.T) {
	of := func(n int) string {
		lines := make([]string, 0, n)
		for i := range n {
			lines = append(lines, "line "+string(rune('a'+i%26)))
		}

		return strings.Join(lines, "\n")
	}

	// Exactly as many lines as are kept is output that fits: cutting here
	// would print a note saying nothing was left out.
	whole := of(stuckLines)
	if got := lastLines(whole, stuckLines); got != whole {
		t.Errorf("output of exactly %d lines came back as %q", stuckLines, got)
	}

	if got := lastLines("one line", stuckLines); got != "one line" {
		t.Errorf("a single line came back as %q", got)
	}

	// One line more, and the note says how many are missing.
	over := lastLines(of(stuckLines+1), stuckLines)
	if !strings.Contains(over, "the first 1 lines are not repeated here") {
		t.Errorf("output one line over reads %q", over)
	}

	if strings.Count(over, "\n") != stuckLines {
		t.Errorf("what was kept is %d lines, want %d and the note", strings.Count(over, "\n"), stuckLines)
	}
}
