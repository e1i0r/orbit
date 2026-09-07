package mcp

// What inspect says about the timeline: how much of it comes back, that the
// count is of the whole and not of the tail, and what a task that got stuck
// reports as the last thing that went wrong.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestTheTimelineIsTailedAndSaysSo: the count is of the whole record, so a
// caller can tell a short task from a long one whose beginning was cut.
func TestTheTimelineIsTailedAndTheCountIsNot(t *testing.T) {
	s, sn, r := oneRepo(t)

	events := []record.Event{{At: at(1), Kind: record.TaskCreated, Text: "written"}}
	for i := range timelineTail + 10 {
		events = append(events, record.Event{At: at(2 + i), Kind: record.PhaseThought, Phase: "implement", Text: "thinking"})
	}

	addTask(t, s, r, "PAY-4", events...)

	got := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-4"})

	timeline, ok := got["timeline"].([]any)
	if !ok || len(timeline) != timelineTail {
		t.Fatalf("the timeline carries %d entries, want the last %d", len(timeline), timelineTail)
	}

	if got["events_total"] != float64(len(events)) {
		t.Errorf("events_total = %v, want %d — the total is of the record and not of the tail", got["events_total"], len(events))
	}
}

// TestALongLogIsCutFromTheFrontAndSaysThatItWas. An engine that failed says
// why in its last lines, so a head-truncated log is the half that does not
// contain the answer — and a log the model believes it read in full is worse
// than one it knows it did not.
func TestALongLogIsCutFromTheFrontAndSaysThatItWas(t *testing.T) {
	s, sn, r := oneRepo(t)
	long := strings.Repeat("noise\n", outputChars) + "THE ACTUAL ERROR"
	addTask(t, s, r, "PAY-5",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.PhaseStarted, Phase: "implement"},
		record.Event{At: at(3), Kind: record.PhaseFailed, Phase: "implement", Text: long})

	out, ok := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-5"})["last_output"].(map[string]any)
	if !ok {
		t.Fatal("a phase that printed a log reports no last_output")
	}

	body := str(t, out["text"])
	if !strings.HasSuffix(body, "THE ACTUAL ERROR") {
		t.Error("the log was cut from the end, which is where the error is")
	}

	if len(body) > outputChars+120 {
		t.Errorf("the log came back at %d characters, want about %d", len(body), outputChars)
	}

	if out["complete"] != false {
		t.Error("a log that was cut reports itself as complete; a model that believes it read the whole thing will conclude from the half it got")
	}
}

func TestAShortLogComesBackWholeAndSaysSo(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-6",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.PhaseStarted, Phase: "implement"},
		record.Event{At: at(3), Kind: record.PhaseFinished, Phase: "implement", Text: "ok"})

	out, ok := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-6"})["last_output"].(map[string]any)
	if !ok || out["text"] != "ok" || out["complete"] != true {
		t.Errorf("a short log came back as %v, want it whole and marked complete", out)
	}
}

// TestInspectCarriesTheTaskItself: the text of task.md is what the engines
// were told, and a supervisor judging a run without it is judging the
// outcome of an instruction it never read.
func TestInspectCarriesTheTaskItself(t *testing.T) {
	_, sn, _ := oneRepo(t)
	created := call(t, sn, "orbit_create_task", map[string]any{"title": "make the webhook idempotent"})

	got := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": created["id"]})
	if body := str(t, got["text"]); !strings.Contains(body, "make the webhook idempotent") {
		t.Errorf("the inspection carries the task as %q, want what was written", body)
	}
}

// TestInspectSaysAnAttemptWasRetriedRatherThanLeavingItRunning. A phase
// whose gate refused it is run again, so one phase name can open twice in
// one run. Without a state for the seam, the first of them reads as a phase
// that started and never ended — which is the one thing a supervisor acts
// on.
func TestInspectSaysAnAttemptWasRetriedRatherThanLeavingItRunning(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-2",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "make the webhook idempotent"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{
			At:    at(3),
			Kind:  record.PhaseStarted,
			Phase: "implement",
			Data:  map[string]string{"engine": "claude"},
		},
		record.Event{
			At:    at(4),
			Kind:  record.GateFailed,
			Phase: "implement",
			Text:  "go test ./... — 1 failure",
		},
		record.Event{
			At:    at(5),
			Kind:  record.PhaseRetried,
			Phase: "implement",
			Data:  map[string]string{"gate": "tests", "attempt": "1", "attempts": "3"},
		},
		record.Event{
			At:    at(6),
			Kind:  record.PhaseStarted,
			Phase: "implement",
			Data:  map[string]string{"engine": "claude"},
		},
		record.Event{At: at(7), Kind: record.PhaseFinished, Phase: "implement"},
		record.Event{At: at(8), Kind: record.TaskFinished},
	)

	phases := list(t, call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-2"})["phases"])
	if len(phases) != 2 {
		t.Fatalf("the inspection reports %d phases, want the two attempts", len(phases))
	}

	if got := obj(t, phases[0])["state"]; got != "retried" {
		t.Errorf("the first attempt reads as %v, want retried", got)
	}

	if got := obj(t, phases[1])["state"]; got != "finished" {
		t.Errorf("the attempt that stood reads as %v, want finished", got)
	}
}

// TestInspectReportsAStuckTaskAsTheLastThingThatWentWrong. A task out of
// attempts writes phase.failed and then task.stuck, and the second is the
// one carrying the summary of what was tried. A supervisor handed the first
// gets the gate's exit code and none of the three attempts behind it.
func TestInspectReportsAStuckTaskAsTheLastThingThatWentWrong(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-3",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "make the webhook idempotent"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{
			At:    at(3),
			Kind:  record.PhaseStarted,
			Phase: "implement",
			Data:  map[string]string{"engine": "claude"},
		},
		record.Event{
			At:    at(4),
			Kind:  record.PhaseFailed,
			Phase: "implement",
			Data:  map[string]string{"error": "gate \"tests\" failed (exit 1)"},
		},
		record.Event{
			At:   at(5),
			Kind: record.TaskStuck,
			Text: "3 attempts at phase \"implement\", and the gate \"tests\" refused every one of them.",
			Data: map[string]string{"attempts": "3", "phase": "implement", "gate": "tests"},
		},
	)

	got := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-3"})

	last, ok := got["last_error"].(map[string]any)
	if !ok {
		t.Fatalf("the inspection carries no last_error: %v", got["last_error"])
	}

	if last["kind"] != record.TaskStuck {
		t.Errorf("last_error is a %v, want the task.stuck that ended the run", last["kind"])
	}

	text, ok := last["text"].(string)
	if !ok {
		t.Fatalf("last_error carries no text: %v", last["text"])
	}

	if !strings.Contains(text, "3 attempts") {
		t.Errorf("last_error carries %q, want the summary of the attempts", text)
	}
}
