//go:build integration

package integration

// The `coverage` flow: implement, then a loop that goes round until the
// tests pass and the coverage clears ninety, then a review that waits.
//
// This is the one that exercises the most of Orbit at once, and the checks
// are real: `go test ./...` and `go tool cover` run against the module in
// the worktree, so the loop goes round because a test really failed and
// stops because it really passed.

import "testing"

// TestCoverageGoesRoundUntilTheChecksPass.
func TestCoverageGoesRoundUntilTheChecksPass(t *testing.T) {
	b := newBoard(t, map[string]any{
		"1-implement": []any{map[string]any{
			"write": map[string]string{"ledger.go": broken},
			"say":   "changed the total",
			"cost":  0.25,
		}},
		// The first turn does not fix it: a loop that went green on its
		// first turn would prove only that the checks ran once.
		"fix": []any{
			map[string]any{"write": map[string]string{"ledger.go": stillWrong}, "say": "tried the sum"},
			map[string]any{"write": map[string]string{"ledger.go": right}, "say": "the refund is subtracted once"},
		},
		"3-review": []any{map[string]any{"say": "the coverage is real"}},
	})

	b.must(t, "new", "-repo", b.repo, "-id", "LED-5", "-flow", "coverage", "make the total right")

	run := b.start(t, "run", "-repo", b.repo, "LED-5")

	// The review at the end is what the run parks at, so reaching the gate
	// means the loop before it went green.
	b.waitFor(t, "LED-5", "phase.waiting", gateWait)
	b.must(t, "resume", "-repo", b.repo, "LED-5")

	if err := run.Wait(); err != nil {
		t.Fatalf("the run did not end after the review was let go: %v", err)
	}

	events := b.record(t, "LED-5")

	if n := count(events, "loop.checked"); n != 2 {
		t.Errorf("the loop was checked %d times, want the failing turn and the passing one:\n%v",
			n, kinds(events))
	}

	if !holds(events, "task.finished") {
		t.Errorf("the task never finished:\n%v", kinds(events))
	}

	// One implement, two turns of the loop's own phase, one review.
	if n := count(events, "phase.finished"); n != 4 {
		t.Errorf("%d phases finished, want implement, two turns of fix, and the review", n)
	}
}

// TestALoopThatNeverGoesGreenStopsAtItsCap, and leaves the task stuck rather
// than failed: nothing broke, and what is left is a decision.
func TestALoopThatNeverGoesGreenStopsAtItsCap(t *testing.T) {
	b := newBoard(t, map[string]any{
		"1-implement": []any{map[string]any{
			"write": map[string]string{"ledger.go": broken},
			"say":   "changed the total",
		}},
		"fix":      []any{map[string]any{"write": map[string]string{"ledger.go": stillWrong}, "say": "tried again"}},
		"3-review": []any{map[string]any{"say": "never reached"}},
	})

	b.must(t, "new", "-repo", b.repo, "-id", "LED-6", "-flow", "coverage", "make the total right")

	if _, err := b.orbit(t, "run", "-repo", b.repo, "LED-6"); err == nil {
		t.Fatal("a loop that never went green ended without saying so")
	}

	events := b.record(t, "LED-6")

	if !holds(events, "task.stuck") {
		t.Errorf("the task is not stuck:\n%v", kinds(events))
	}

	if holds(events, "task.finished") {
		t.Errorf("the task finished on checks that never passed:\n%v", kinds(events))
	}

	// The loop's own cap, which coverage.json sets to three.
	if n := count(events, "loop.checked"); n != 3 {
		t.Errorf("the loop went round %d times, want the three its cap allows", n)
	}
}

// The three states of the ledger these tests walk through: the change that
// broke the test, one that does not fix it, and the one that does.
const (
	broken = "package ledger\n\n" +
		"// Total adds a charge and a refund.\n" +
		"func Total(charge, refund int) int {\n\treturn charge * refund\n}\n"

	stillWrong = "package ledger\n\n" +
		"// Total adds a charge and a refund.\n" +
		"func Total(charge, refund int) int {\n\treturn charge + refund\n}\n"

	right = "package ledger\n\n" +
		"// Total adds a charge and a refund.\n" +
		"func Total(charge, refund int) int {\n\treturn charge - refund\n}\n"
)
