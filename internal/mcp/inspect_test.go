package mcp

// What orbit_inspect_task must actually carry. Its description promises the
// log, the thinking, the last error and the notes, and the first version of
// it answered with twelve board columns — a supervisor reading that has no
// evidence at all, and no way to tell that it has none.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// failedRun is a task that ran, thought, failed a gate and then failed, so
// that one inspection has something of every kind to carry.
func failedRun(t *testing.T) (Session, string) {
	t.Helper()
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-1",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "make the webhook idempotent"},
		record.Event{At: at(2), Kind: record.TaskNoted, Text: "the retry storm started on friday"},
		record.Event{At: at(3), Kind: record.TaskStarted},
		record.Event{At: at(4), Kind: record.PhaseStarted, Phase: "implement", Data: map[string]string{"engine": "claude", "model": "opus"}},
		record.Event{At: at(5), Kind: record.PhaseThought, Phase: "implement", Text: "the handler keys on the request id"},
		record.Event{At: at(6), Kind: record.GateFailed, Phase: "implement", Text: "go test ./... — 1 failure"},
		record.Event{At: at(7), Kind: record.PhaseFailed, Phase: "implement", Text: "--- FAIL: TestIdempotent\n", Data: map[string]string{"error": "the gate said no"}},
		record.Event{At: at(8), Kind: record.TaskFailed, Text: "the implement phase failed", Data: map[string]string{"error": "the gate said no"}},
		record.Event{At: at(9), Kind: record.TaskNoted, Text: "[supervisor] start from the failing test"},
	)

	return sn, "PAY-1"
}

func TestInspectCarriesTheEvidenceAndNotJustTheRow(t *testing.T) {
	sn, id := failedRun(t)
	got := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": id})

	for _, key := range []string{"timeline", "notes", "thinking", "phases", "gates", "last_error", "last_output", "text", "events_total"} {
		if _, ok := got[key]; !ok {
			t.Errorf("the inspection carries no %q, and its description promises it", key)
		}
	}

	if got["id"] != id || got["band"] != "needs_you" {
		t.Errorf("the inspection describes %v in band %v, want %s in needs_you", got["id"], got["band"], id)
	}

	if got["events_total"] != float64(9) {
		t.Errorf("events_total = %v, want 9 — the count is of the record, not of the tail that was carried", got["events_total"])
	}
}

// TestInspectCarriesEveryNoteOldestFirst. Notes are what a human or a
// supervisor chose to write down, and the first one is usually the
// instruction the rest are answers to, so this field is never tailed.
func TestInspectCarriesEveryNoteOldestFirst(t *testing.T) {
	sn, id := failedRun(t)

	notes, ok := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": id})["notes"].([]any)
	if !ok || len(notes) != 2 {
		t.Fatalf("the inspection carries %v, want both notes", notes)
	}

	first := obj(t, notes[0])

	second := obj(t, notes[1])
	if !strings.Contains(str(t, first["text"]), "retry storm") {
		t.Errorf("the first note is %v, want the oldest one", first["text"])
	}

	if !strings.Contains(str(t, second["text"]), "start from the failing test") {
		t.Errorf("the second note is %v", second["text"])
	}
}

func TestInspectCarriesTheLastThingThatWentWrong(t *testing.T) {
	sn, id := failedRun(t)
	got := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": id})

	last, ok := got["last_error"].(map[string]any)
	if !ok {
		t.Fatalf("a task that failed reports last_error as %v", got["last_error"])
	}

	if last["kind"] != record.TaskFailed {
		t.Errorf("last_error is the %v event, want the most recent failure", last["kind"])
	}

	if last["reason"] != "the gate said no" {
		t.Errorf("last_error reason = %v, want the one in the record's data", last["reason"])
	}
}

// TestNothingWentWrongIsNullAndNotEmpty: a supervisor branching on "did this
// fail" must not have to tell "" from absent.
func TestNothingWentWrongIsNullAndNotEmpty(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-2",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "went fine"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{At: at(3), Kind: record.TaskFinished})

	got := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-2"})
	if got["last_error"] != nil {
		t.Errorf("a task that never failed reports last_error as %#v, want null", got["last_error"])
	}
}

func TestInspectCarriesTheGatesAndHowTheyAnswered(t *testing.T) {
	sn, id := failedRun(t)

	gates, ok := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": id})["gates"].([]any)
	if !ok || len(gates) != 1 {
		t.Fatalf("the inspection carries %v, want the one gate that ran", gates)
	}

	gate := obj(t, gates[0])
	if gate["passed"] != false {
		t.Errorf("the gate is reported as passed = %v, and it failed", gate["passed"])
	}

	if !strings.Contains(str(t, gate["text"]), "go test") {
		t.Errorf("the gate carries %v, want what it ran", gate["text"])
	}
}

// TestInspectSaysWhatEachPhaseDid folds the started/finished pair into one
// entry, because two entries per phase is the raw record and this field is
// the summary of it.
func TestInspectSaysWhatEachPhaseDid(t *testing.T) {
	sn, id := failedRun(t)

	phases, ok := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": id})["phases"].([]any)
	if !ok || len(phases) != 1 {
		t.Fatalf("the inspection carries %v, want one entry for the one phase that ran", phases)
	}

	got := obj(t, phases[0])
	if got["phase"] != "implement" || got["state"] != "failed" {
		t.Errorf("the phase is reported as %v/%v, want implement/failed", got["phase"], got["state"])
	}

	if got["engine"] != "claude" || got["model"] != "opus" {
		t.Errorf("the phase ran on %v/%v, want the engine and model the record names", got["engine"], got["model"])
	}

	if got["error"] != "the gate said no" {
		t.Errorf("the phase carries error %v, want the one in its data", got["error"])
	}
}

// TestTheThinkingIsTheLastFewBlocksAndNotAllOfThem. Every block of a long
// run would be most of a context window, and the reason a task went wrong is
// stated near the end.
func TestTheThinkingIsTheLastFewBlocks(t *testing.T) {
	s, sn, r := oneRepo(t)

	events := []record.Event{{At: at(1), Kind: record.TaskCreated, Text: "written"}}
	for i := range 12 {
		events = append(events, record.Event{At: at(2 + i), Kind: record.PhaseThought, Phase: "implement", Text: thoughtNumber(i)})
	}

	addTask(t, s, r, "PAY-3", events...)

	thinking, ok := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-3"})["thinking"].([]any)
	if !ok || len(thinking) != thoughtTail {
		t.Fatalf("the inspection carries %d thinking blocks, want the last %d", len(thinking), thoughtTail)
	}

	last := obj(t, thinking[len(thinking)-1])
	if last["text"] != thoughtNumber(11) {
		t.Errorf("the last block carried is %v, want the last one written", last["text"])
	}

	first := obj(t, thinking[0])
	if first["text"] != thoughtNumber(7) {
		t.Errorf("the tail starts at %v, want the fifth from the end", first["text"])
	}
}

func thoughtNumber(i int) string {
	return "thought " + string(rune('a'+i))
}

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
		record.Event{At: at(3), Kind: record.PhaseStarted, Phase: "implement", Data: map[string]string{"engine": "claude"}},
		record.Event{At: at(4), Kind: record.GateFailed, Phase: "implement", Text: "go test ./... — 1 failure"},
		record.Event{At: at(5), Kind: record.PhaseRetried, Phase: "implement", Data: map[string]string{"gate": "tests", "attempt": "1", "attempts": "3"}},
		record.Event{At: at(6), Kind: record.PhaseStarted, Phase: "implement", Data: map[string]string{"engine": "claude"}},
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
		record.Event{At: at(3), Kind: record.PhaseStarted, Phase: "implement", Data: map[string]string{"engine": "claude"}},
		record.Event{At: at(4), Kind: record.PhaseFailed, Phase: "implement", Data: map[string]string{"error": "gate \"tests\" failed (exit 1)"}},
		record.Event{At: at(5), Kind: record.TaskStuck, Text: "3 attempts at phase \"implement\", and the gate \"tests\" refused every one of them.", Data: map[string]string{"attempts": "3", "phase": "implement", "gate": "tests"}},
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
