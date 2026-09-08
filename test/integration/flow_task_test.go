//go:build integration

package integration

// The `task` flow, which is the default: implement, then a review that
// stops for a person.
//
// The gate is the whole point of this one, and the run does not end at it —
// it parks and waits, which is what a person walking away from a terminal
// relies on. So the run is started in the background here and let go the way
// a reader lets it go.

import "testing"

// TestTaskStopsAtItsReviewGateAndGoesOnWhenLetGo.
func TestTaskStopsAtItsReviewGateAndGoesOnWhenLetGo(t *testing.T) {
	b := newBoard(t, map[string]any{
		"implement": []any{map[string]any{
			"write": map[string]string{"ledger.go": fixed},
			"say":   "the refund is added now",
			"cost":  0.25,
		}},
		"review": []any{map[string]any{"say": "it reads right", "cost": 0.10}},
	})

	b.must(t, "new", "-repo", b.repo, "-id", "LED-3", "-flow", "task", "fix the total")

	run := b.start(t, "run", "-repo", b.repo, "LED-3")

	b.waitFor(t, "LED-3", "phase.waiting", gateWait)

	held := b.record(t, "LED-3")
	if holds(held, "task.finished") {
		t.Fatalf("the run went through the review nobody agreed to:\n%v", kinds(held))
	}

	if n := count(held, "phase.finished"); n != 1 {
		t.Errorf("%d phases finished before the gate, want the one before it", n)
	}

	b.must(t, "resume", "-repo", b.repo, "LED-3")

	if err := run.Wait(); err != nil {
		t.Fatalf("the run did not end after the gate was let go: %v", err)
	}

	events := b.record(t, "LED-3")
	if !holds(events, "task.finished") {
		t.Errorf("the task never finished:\n%v", kinds(events))
	}

	if n := count(events, "phase.finished"); n != 2 {
		t.Errorf("%d phases finished, want the implement and the review", n)
	}
}

// TestAGateCanBeSkipped, which is the other word a reader has: the phase in
// front of the wait does not run at all, and nothing is written about work
// that never happened.
func TestAGateCanBeSkipped(t *testing.T) {
	b := newBoard(t, map[string]any{
		"implement": []any{map[string]any{
			"write": map[string]string{"ledger.go": fixed},
			"say":   "done",
		}},
		"review": []any{map[string]any{"say": "should never run"}},
	})

	b.must(t, "new", "-repo", b.repo, "-id", "LED-4", "-flow", "task", "fix the total")

	run := b.start(t, "run", "-repo", b.repo, "LED-4")

	b.waitFor(t, "LED-4", "phase.waiting", gateWait)
	b.must(t, "skip", "-repo", b.repo, "LED-4")

	if err := run.Wait(); err != nil {
		t.Fatalf("the run did not end after the phase was skipped: %v", err)
	}

	events := b.record(t, "LED-4")
	if n := count(events, "phase.finished"); n != 1 {
		t.Errorf("%d phases finished, want only the one before the skipped gate", n)
	}

	for _, e := range events {
		if e.Kind == "phase.finished" && e.Phase == "review" {
			t.Error("the skipped phase wrote down work it never did")
		}
	}
}

// fixed is the ledger with the bug taken out, which is what a phase of these
// tests writes.
const fixed = "package ledger\n\n" +
	"// Total adds a charge and a refund.\n" +
	"func Total(charge, refund int) int {\n\treturn charge + refund\n}\n"
