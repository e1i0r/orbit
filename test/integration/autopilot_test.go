//go:build integration

package integration

// Autopilot, and tasks running beside each other — the two the landing shows
// that are about a board rather than about one flow.

import (
	"sync"
	"testing"
)

// TestAutopilotWalksThroughAGateTheFlowAsked. The switch is what turns a
// board somebody drives into a board that runs itself, and the whole of what
// it does is lift the gates a flow asked for.
func TestAutopilotWalksThroughAGateTheFlowAsked(t *testing.T) {
	b := newBoard(t, map[string]any{
		"implement": []any{map[string]any{
			"write": map[string]string{"ledger.go": right},
			"say":   "done",
		}},
		"review": []any{map[string]any{"say": "it reads right"}},
	})

	b.must(t, "settings", "autopilot", "on")
	b.must(t, "new", "-repo", b.repo, "-id", "LED-9", "-flow", "task", "fix the total")
	b.must(t, "run", "-repo", b.repo, "LED-9")

	events := b.record(t, "LED-9")

	if holds(events, "phase.waiting") {
		t.Errorf("the run stopped at a gate autopilot was meant to lift:\n%v", kinds(events))
	}

	if !holds(events, "task.finished") {
		t.Errorf("the task never finished:\n%v", kinds(events))
	}

	if n := count(events, "phase.finished"); n != 2 {
		t.Errorf("%d phases finished, want both of them without anybody asked", n)
	}
}

// TestTwoTasksRunSideBySide, each in its own worktree: the record of one is
// not the record of the other, and neither has to wait for the other.
func TestTwoTasksRunSideBySide(t *testing.T) {
	b := newBoard(t, map[string]any{
		"implement": []any{map[string]any{
			"write": map[string]string{"NOTES.md": "worked\n"},
			"say":   "done",
		}},
	})

	for _, id := range []string{"LED-10", "LED-11"} {
		b.must(t, "new", "-repo", b.repo, "-id", id, "-flow", "quick", "fix the total")
	}

	var wg sync.WaitGroup

	said := make(map[string]string, 2)
	failed := make(map[string]error, 2)

	var mu sync.Mutex

	for _, id := range []string{"LED-10", "LED-11"} {
		wg.Add(1)

		go func() {
			defer wg.Done()

			out, err := b.orbit(t, "run", "-repo", b.repo, id)

			mu.Lock()
			said[id], failed[id] = out, err
			mu.Unlock()
		}()
	}

	wg.Wait()

	for id, err := range failed {
		if err != nil {
			t.Errorf("the run of %s failed beside the other: %v\n%s", id, err, said[id])
		}

		events := b.record(t, id)
		if !holds(events, "task.finished") {
			t.Errorf("%s never finished:\n%v", id, kinds(events))
		}

		if n := count(events, "task.created"); n != 1 {
			t.Errorf("the record of %s holds %d task.created — the two runs share a log", id, n)
		}
	}
}
