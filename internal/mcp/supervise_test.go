package mcp

// The tools that write: a note, a correction, a retry, a pause. What they
// must leave in the record, and what they must not claim to have done.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestANoteWrittenThroughMCPComesBackFromTheRecord is the round trip the
// dialogue log is: orbit_add_note writes into the same events.jsonl the
// cockpit's notes tab reads, so a diagnosis survives the conversation that
// produced it.
func TestANoteWrittenThroughMCPComesBackFromTheRecord(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	if res := sn.Call("orbit_add_note", map[string]any{"task_id": "PAY-1", "text": "the failure is in the retry loop"}); res.IsError {
		t.Fatalf("orbit_add_note refused: %s", text(t, res))
	}

	notes, ok := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-1"})["notes"].([]any)
	if !ok || len(notes) != 1 {
		t.Fatalf("a note written through the tool is not in the record: %v", notes)
	}

	wrote := str(t, obj(t, notes[0])["text"])
	if !strings.Contains(wrote, "the failure is in the retry loop") {
		t.Errorf("the note reads %q, want the text it was given", wrote)
	}
	// The record carries no author, so a directive that reads as though the
	// person at the keyboard wrote it is evidence about the wrong party.
	if !strings.HasPrefix(wrote, "[supervisor] ") {
		t.Errorf("the note reads %q, want it marked as the supervisor's", wrote)
	}
}

func TestAddNoteNeedsSomethingToSay(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	if said := refused(t, sn, "orbit_add_note", map[string]any{"task_id": "PAY-1", "text": "   "}); !strings.Contains(said, "text") {
		t.Errorf("the refusal does not say what is missing: %s", said)
	}
}

// TestRetryRefusesATaskThatIsAlreadyRunning. Two engines in one worktree is
// two processes writing the same files, and the record would carry both as
// one attempt.
func TestRetryRefusesATaskThatIsAlreadyRunning(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-7",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{At: at(3), Kind: record.PhaseStarted, Phase: "implement"})
	holdTask(t, s, r, "PAY-7")

	said := refused(t, sn, "orbit_retry_task", map[string]any{"task_id": "PAY-7"})
	if !strings.Contains(said, "PAY-7") || !strings.Contains(said, "orbit_cancel_task") {
		t.Errorf("the refusal is %q, want it to name the task and what to do instead", said)
	}
}

// TestTheCorrectionIsInTheRecordBeforeTheRunStarts is why retry writes the
// note first: a reader asking why this attempt differed from the last one
// finds the instruction in the task's own history rather than in a chat log
// nobody kept.
func TestTheCorrectionIsInTheRecord(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-8",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{At: at(3), Kind: record.TaskFailed, Text: "broke", Data: map[string]string{"error": "broke"}})

	// The run itself is not started here — that spawns an engine — so the
	// tool's own refusal is allowed. What must be true either way is that
	// the correction reached the record.
	sn.Call("orbit_retry_task", map[string]any{"task_id": "PAY-8", "corrective_prompt": "start from the failing test"}) //nolint:errcheck // the result is not what this test is about

	notes := list(t, call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-8"})["notes"])
	if len(notes) != 1 {
		t.Fatalf("the correction is not in the record: %v", notes)
	}

	got := str(t, obj(t, notes[0])["text"])
	if !strings.Contains(got, "start from the failing test") || !strings.HasPrefix(got, "[supervisor] ") {
		t.Errorf("the correction reads %q, want the supervisor's text", got)
	}
}

// TestPauseSaysWhatItHasAndHasNotDone. The word is on disk and the run reads
// it at its next phase boundary, which has not happened yet; reporting the
// task as paused would be a claim about something this process cannot see.
func TestPauseSaysWhatItHasAndHasNotDone(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-9",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.TaskStarted})

	res := sn.Call("orbit_pause_task", map[string]any{"task_id": "PAY-9"})
	if res.IsError {
		t.Fatalf("orbit_pause_task refused: %s", text(t, res))
	}

	said := text(t, res)
	if !strings.Contains(said, "will be paused") || !strings.Contains(said, "PAY-9") {
		t.Errorf("pause answered %q, want it to say the word was left rather than that the run has stopped", said)
	}
}

// TestRetryRefusesAMarkerNobodyCouldRead. The refusal above is about a task
// something is holding; this one is about a task nobody can say anything
// about, and they are different sentences because they ask for different
// things — a stop, and a look at a file.
//
// `live` carries the same answer into the listing. It is a word rather than a
// flag for this state alone: a model told false starts the task, and a model
// cannot open the marker to check.
func TestRetryRefusesAMarkerNobodyCouldRead(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-9",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{At: at(3), Kind: record.PhaseStarted, Phase: "implement"})
	damageMarker(t, s, r, "PAY-9")

	// The sentence and not merely the refusal: task.Start reads the same
	// marker and stops too, but what it says is strconv.Atoi's syntax error,
	// and a model handed that has nothing to pass on to whoever can open the
	// file.
	said := refused(t, sn, "orbit_retry_task", map[string]any{"task_id": "PAY-9"})
	if !strings.Contains(said, "PAY-9") || !strings.Contains(said, "cannot tell whether a phase is running") {
		t.Errorf("the refusal is %q, want it to name the task and what nobody knows about it", said)
	}

	rows := list(t, call(t, sn, "orbit_list_tasks", nil)["tasks"])
	if len(rows) != 1 {
		t.Fatalf("%d tasks listed, want 1", len(rows))
	}

	if got := str(t, obj(t, rows[0])["live"]); got != "unknown" {
		t.Errorf("the row reports live=%q, want %q", got, "unknown")
	}
}
