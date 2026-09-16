//go:build integration

package integration

// A phase refused what it needed, through the whole of Orbit.
//
// It is here and not only beside the unit tests because this is how it got
// out: the engine handles its own denial, answers in prose that it could
// not, and exits zero. Every layer between the refusal and the verdict has
// to carry it — the stream parser, the adapter, the phase, the run — and a
// fake engine object inside the program is none of them.
//
// It cost a real task. NOTIF-1 asked for a CONTRIBUTING.md, was denied the
// write, said so, and was written down as finished.

import (
	"strings"
	"testing"
)

// TestAPhaseDeniedAndEmptyIsNotWrittenDownAsFinished.
func TestAPhaseDeniedAndEmptyIsNotWrittenDownAsFinished(t *testing.T) {
	b := newBoard(t, map[string]any{
		"implement": []any{map[string]any{"refuse": "Write"}},
	})

	b.must(t, "board", "new", "-repo", b.repo, "-id", "LED-11", "-flow", "quick", "write a CONTRIBUTING.md")

	said, err := b.orbit(t, "task", "start", "-repo", b.repo, "LED-11")
	if err == nil {
		t.Fatalf("a run that was denied and wrote nothing ended happily:\n%s", said)
	}

	if !strings.Contains(said, "what the phase is allowed") {
		t.Errorf("it does not say where to look:\n%s", said)
	}

	events := b.record(t, "LED-11")

	if !holds(events, "phase.denied") {
		t.Errorf("the record does not say the phase was denied:\n%v", kinds(events))
	}

	for _, wrong := range []string{"phase.finished", "task.finished"} {
		if holds(events, wrong) {
			t.Errorf("a phase that did nothing was written down as %s:\n%v", wrong, kinds(events))
		}
	}

	// Which tool, because what a reader does about this is look at what the
	// phase was allowed — and that question starts with what it asked for.
	if got := lastOf(events, "phase.denied"); got.Data["tool"] != "Write" {
		t.Errorf("it says it was denied %q, want Write", got.Data["tool"])
	}
}

// TestAPhaseRefusedThatWentAnotherWayIsFine. A refusal on its own is
// ordinary: an engine turned down once that found another way did the work,
// and the work is on disk to prove it.
func TestAPhaseRefusedThatWentAnotherWayIsFine(t *testing.T) {
	b := newBoard(t, map[string]any{
		"implement": []any{map[string]any{
			"write": map[string]string{"NOTES.md": "it went another way\n"},
			"say":   "the first way was refused",
		}},
	})

	b.must(t, "board", "new", "-repo", b.repo, "-id", "LED-12", "-flow", "quick", "write a note")
	b.must(t, "task", "start", "-repo", b.repo, "LED-12")

	events := b.record(t, "LED-12")
	if holds(events, "phase.denied") {
		t.Errorf("a phase that did the work was written down as denied:\n%v", kinds(events))
	}

	if !holds(events, "task.finished") {
		t.Errorf("it did not finish:\n%v", kinds(events))
	}
}
