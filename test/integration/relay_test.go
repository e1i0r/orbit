//go:build integration

package integration

// An engine that runs out, and the task carrying on under another one.
//
// It is here and not beside the unit tests because the one place this can go
// wrong is the one a unit test cannot reach. Whether a run is read as having
// run out is decided by matching what the provider printed against a list of
// phrases written by hand — and between that list and a real refusal stands
// exec, a pipe, a stream parser and the adapter that decides whether the
// words went to stdout or into the error. A fake engine object inside the
// program skips all four.
//
// So the stand-in here is a real binary, installed under the engine's own
// name, that prints what a provider prints and leaves non-zero. Everything
// after that is Orbit: the command line resolving the engine, spawning it,
// reading its stream, writing the ending down, choosing who takes the task,
// and running the same phase again under the new one.
//
// One thing it does not prove, and it is worth saying so rather than letting
// somebody read more into a green test than is there: the stand-in speaks
// claude's stream whatever name it is installed under, so what the engine
// that takes over *answers* is not parsed the way a real codex's would be.
// What is proved is the half this is about — the refusal read through a real
// pipe, the hand-over, and the new engine really running and really writing
// the file.

import (
	"fmt"
	"strings"
	"testing"
)

// saidByAnthropic is what claude prints when the allowance is gone. The words
// are the provider's; internal/engine/ranout.go is the list they are matched
// against, and this is the scenario that proves the match survives the pipe.
const saidByAnthropic = "Claude AI usage limit reached, resets at 3pm"

// ranOutOn is a script where one engine is refused by its provider and every
// phase otherwise does real work.
func ranOutOn(engine string) map[string]any {
	return map[string]any{
		"engines": map[string]any{
			engine: []any{map[string]any{"ranOut": saidByAnthropic}},
		},
		"phases": map[string]any{
			"implement": []any{map[string]any{
				"write": map[string]string{
					"ledger.go": "package ledger\n\n// Total adds a charge and a refund.\nfunc Total(charge, refund int) int {\n\treturn charge + refund\n}\n",
				},
				"say": "the refund was being subtracted twice",
			}},
		},
	}
}

// TestAnEngineRefusedByItsProviderHandsTheTaskOn is the whole project, end to
// end, through the real binary.
func TestAnEngineRefusedByItsProviderHandsTheTaskOn(t *testing.T) {
	b := newBoardOn(t, ranOutOn("claude"))

	// Autopilot is what decides whether a relay happens without asking, and
	// the engine setting is the one preference Orbit has been told: it is
	// what a relay reaches for first.
	b.must(t, "settings", "set", "autopilot", "on")
	b.must(t, "settings", "set", "engine", "codex")

	b.must(t, "board", "new", "-repo", b.repo, "-id", "LED-9", "-flow", "quick", "fix the total")
	b.must(t, "task", "start", "-repo", b.repo, "-engine", "claude", "LED-9")

	events := b.record(t, "LED-9")

	for _, want := range []string{"phase.ran_out", "task.relayed", "task.finished"} {
		if !holds(events, want) {
			t.Fatalf("the record has no %s:\n%v", want, kinds(events))
		}
	}

	// It ran out; it did not break. The two send a reader to do opposite
	// things, and everything below depends on which of them this was.
	if holds(events, "phase.failed") {
		t.Errorf("a refused allowance was written down as a failure:\n%v", kinds(events))
	}

	relay := lastOf(events, "task.relayed")
	if relay.Data["from"] != "claude" || relay.Data["to"] != "codex" {
		t.Errorf("the relay reads %q → %q, want claude → codex", relay.Data["from"], relay.Data["to"])
	}

	if relay.Data["why"] != "ran out" {
		t.Errorf("the relay says it happened because %q", relay.Data["why"])
	}

	// The phase that did the work says which engine did it. A record still
	// naming the engine the flow named would credit claude with codex's work.
	if !ranUnder(events, "codex") {
		t.Errorf("no phase was run by codex:\n%v", kinds(events))
	}

	// And the work is on disk. A relay that wrote a tidy record and left the
	// worktree untouched would be a relay that carried nothing on.
	if found := worktreeFile(t, b, "ledger.go"); !strings.Contains(found, "charge + refund") {
		t.Errorf("the phase the new engine ran left nothing behind:\n%s", found)
	}

	// Run with -v, this is the relay as a person would read it. It is the
	// other half of what this test is for: the assertions say it works, and
	// this says what working looks like, without waiting for an allowance to
	// run out on a real afternoon.
	t.Log("\n" + story(events))
}

// story is the record as a few lines a person reads.
func story(events []event) string {
	var b strings.Builder

	for _, e := range events {
		switch e.Kind {
		case "phase.thought", "phase.tool_call", "phase.asked":
			continue
		}

		// The error when there is one, because that is where a refusal
		// lands: the provider printed it on stderr and the adapter put it
		// in what it returned, not in what it captured.
		said := e.Data["error"]
		if said == "" {
			said = e.Text
		}

		fmt.Fprintf(&b, "%-16s %-12s %-8s %s\n", e.Kind, e.Phase,
			e.Data["engine"], strings.SplitN(strings.TrimSpace(said), "\n", 2)[0])
	}

	return b.String()
}

// TestWithoutAutopilotItStopsAndNamesWhoCouldTakeIt. The same decision,
// answered the other way — and the reason there is no second switch for it.
func TestWithoutAutopilotItStopsAndNamesWhoCouldTakeIt(t *testing.T) {
	b := newBoardOn(t, ranOutOn("claude"))

	b.must(t, "settings", "set", "autopilot", "off")
	b.must(t, "board", "new", "-repo", b.repo, "-id", "LED-10", "-flow", "quick", "fix the total")

	// The run ends unhappily on purpose: it ran out and nobody said to carry
	// on without asking, so orbit leaves non-zero and says why.
	said, err := b.orbit(t, "task", "start", "-repo", b.repo, "-engine", "claude", "LED-10")
	if err == nil {
		t.Fatalf("orbit task start: want an error when the engine ran out:\n%s", said)
	}

	events := b.record(t, "LED-10")

	needs := lastOf(events, "task.needs_engine")
	if needs.Kind == "" {
		t.Fatalf("the run stopped without saying it needs an engine:\n%v", kinds(events))
	}

	if !strings.Contains(needs.Data["engines"], "codex") {
		t.Errorf("it stopped without naming who could take it: %q", needs.Data["engines"])
	}

	if holds(events, "task.relayed") {
		t.Errorf("the task changed engine with autopilot off:\n%v", kinds(events))
	}
}

// ranUnder is whether any phase of the run was started under that engine.
func ranUnder(events []event, engine string) bool {
	for _, e := range events {
		if e.Kind == "phase.started" && e.Data["engine"] == engine {
			return true
		}
	}

	return false
}
