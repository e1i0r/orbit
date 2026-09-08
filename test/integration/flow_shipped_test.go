//go:build integration

package integration

// The two flows left of the five that ship: `careful`, which has a phase
// after its gate, and `tdd-fuzz-pr`, whose three phases feed each other.
//
// A flow is a promise about an order. These walk each one to the end and
// check the order the record kept, because a phase that silently did not
// run is the failure nobody notices until a release.

import "testing"

// TestCarefulRunsItsFixAfterTheReview.
func TestCarefulRunsItsFixAfterTheReview(t *testing.T) {
	b := newBoard(t, map[string]any{
		"implement": []any{map[string]any{
			"write": map[string]string{"ledger.go": right},
			"say":   "the refund is subtracted once",
		}},
		"review": []any{map[string]any{"say": "one thing to tidy"}},
		"fix":    []any{map[string]any{"write": map[string]string{"NOTES.md": "tidied\n"}, "say": "tidied it"}},
	})

	b.must(t, "new", "-repo", b.repo, "-id", "LED-7", "-flow", "careful", "fix the total carefully")

	run := b.start(t, "run", "-repo", b.repo, "LED-7")

	b.waitFor(t, "LED-7", "phase.waiting", gateWait)
	b.must(t, "resume", "-repo", b.repo, "LED-7")

	if err := run.Wait(); err != nil {
		t.Fatalf("the run did not end after the review was let go: %v", err)
	}

	events := b.record(t, "LED-7")
	if !holds(events, "task.finished") {
		t.Fatalf("the task never finished:\n%v", kinds(events))
	}

	if got := phasesFinished(events); len(got) != 3 {
		t.Errorf("the phases that finished were %v, want implement, review and fix", got)
	}

	if last := phasesFinished(events); len(last) > 0 && last[len(last)-1] != "fix" {
		t.Errorf("the last phase to finish was %q, want the fix that follows the review", last[len(last)-1])
	}
}

// TestTddFuzzPrWalksItsThreePhasesInOrder.
func TestTddFuzzPrWalksItsThreePhasesInOrder(t *testing.T) {
	b := newBoard(t, map[string]any{
		"1-plan": []any{map[string]any{"say": "the plan: subtract the refund once"}},
		"2-implement-fuzz": []any{map[string]any{
			"write": map[string]string{"ledger.go": right},
			"say":   "implemented, with a fuzz target",
		}},
		"3-review-pr": []any{map[string]any{"say": "ready for a pull request"}},
	})

	b.must(t, "new", "-repo", b.repo, "-id", "LED-8", "-flow", "tdd-fuzz-pr", "make the total right")

	run := b.start(t, "run", "-repo", b.repo, "LED-8")

	b.waitFor(t, "LED-8", "phase.waiting", gateWait)
	b.must(t, "resume", "-repo", b.repo, "LED-8")

	if err := run.Wait(); err != nil {
		t.Fatalf("the run did not end after the review was let go: %v", err)
	}

	events := b.record(t, "LED-8")

	want := []string{"1-plan", "2-implement-fuzz", "3-review-pr"}
	got := phasesFinished(events)

	if len(got) != len(want) {
		t.Fatalf("the phases that finished were %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("phase %d was %q, want %q", i+1, got[i], want[i])
		}
	}
}

// phasesFinished is the phases that ended, in the order they ended, which is
// the order a flow promises.
func phasesFinished(events []event) []string {
	var out []string

	for _, e := range events {
		if e.Kind == "phase.finished" {
			out = append(out, e.Phase)
		}
	}

	return out
}
