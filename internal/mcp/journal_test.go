package mcp

// What a call leaves behind in the record of the task it acted on, and what
// it must not leave behind anywhere.

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
)

// kindsOn is one task's record, by kind, in the order it was written.
func kindsOn(t *testing.T, s *store.Store, r repo.Repo, id string) []record.Event {
	t.Helper()
	path, err := s.EventsPath(r.Path, id)
	if err != nil {
		t.Fatalf("events path of task %s: %v", id, err)
	}
	events, err := record.Read(path)
	if err != nil {
		t.Fatalf("read the record of task %s: %v", id, err)
	}
	return events
}

// only answers the events of one kind, and fails saying what it found when
// there is not exactly one.
func only(t *testing.T, events []record.Event, kind string) record.Event {
	t.Helper()
	var found []record.Event
	for _, e := range events {
		if e.Kind == kind {
			found = append(found, e)
		}
	}
	if len(found) != 1 {
		t.Fatalf("the record has %d %s events, want exactly 1: %v", len(found), kind, events)
	}
	return found[0]
}

// A pause asked for over MCP is in the task's history, said as the thing it
// was: a model acted on this, at this time.
func TestAPauseAskedForOverMCPIsInTheTasksHistory(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-9",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.TaskStarted})

	if res := sn.Call("orbit_pause_task", map[string]any{"task_id": "PAY-9"}); res.IsError {
		t.Fatalf("orbit_pause_task refused: %s", text(t, res))
	}
	events := kindsOn(t, s, r, "PAY-9")
	wrote := only(t, events, record.TaskDialogue)
	if wrote.Data["by"] != "mcp" {
		t.Errorf("by = %q, want the server that acted", wrote.Data["by"])
	}
	if !strings.Contains(wrote.Text, "pause") {
		t.Errorf("the record says %q, want it to say what was asked for", wrote.Text)
	}
	// The one thing it must not be. A note is handed to the next phase that
	// starts, so "a model asked this task to pause" written as a note is an
	// engine being told to stop by nobody.
	for _, e := range events {
		if e.Kind == record.TaskNoted {
			t.Errorf("a pause left a note the next run will be handed: %q", e.Text)
		}
	}
}

// heldTask plants a run marker naming a process that is really there and is
// really willing to be signalled — a sleep of its own, in a group of its
// own. holdTask writes this process's pid, which is right for a tool that
// only reads the marker and fatal for the one that sends it SIGTERM.
func heldTask(t *testing.T, s *store.Store, r repo.Repo, id string) {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", "sleep 30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start a process to signal: %v", err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck // best effort: the test is over
		_ = cmd.Wait()                                      //nolint:errcheck // best effort: the test is over
	})
	path, err := s.RunPath(r.Path, id)
	if err != nil {
		t.Fatalf("run marker path of task %s: %v", id, err)
	}
	body := fmt.Sprintf("pid: %d\nstarted: %s\n", cmd.Process.Pid, time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("plant the run marker of task %s: %v", id, err)
	}
}

func TestACancelIsInTheTasksHistory(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-10",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.TaskStarted})
	heldTask(t, s, r, "PAY-10")

	if res := sn.Call("orbit_cancel_task", map[string]any{"task_id": "PAY-10"}); res.IsError {
		t.Fatalf("orbit_cancel_task refused: %s", text(t, res))
	}
	if got := only(t, kindsOn(t, s, r, "PAY-10"), record.TaskDialogue); !strings.Contains(got.Text, "cancelled") {
		t.Errorf("the record says %q, want it to say the task was cancelled", got.Text)
	}
}

// A restart leaves two lines of two different kinds, and which is which is
// the whole distinction: the correction is a note, because the next phase is
// meant to read it, and the restart beside it is not, because it is about
// the task rather than to it.
func TestARestartIsInTheTasksHistoryBesideItsCorrection(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-12",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{At: at(3), Kind: record.TaskFailed, Text: "broke", Data: map[string]string{"error": "broke"}})

	res := sn.Call("orbit_retry_task", map[string]any{"task_id": "PAY-12", "corrective_prompt": "start from the failing test"})
	if res.IsError {
		t.Fatalf("orbit_retry_task refused: %s", text(t, res))
	}
	events := kindsOn(t, s, r, "PAY-12")
	note := only(t, events, record.TaskNoted)
	if !strings.Contains(note.Text, "start from the failing test") {
		t.Errorf("the note reads %q, want the correction and nothing else", note.Text)
	}
	restart := only(t, events, record.TaskDialogue)
	if !strings.Contains(restart.Text, "started this task again") || !strings.Contains(restart.Text, "correction") {
		t.Errorf("the record says %q, want the restart and that it carried an instruction", restart.Text)
	}
	if strings.Contains(restart.Text, "start from the failing test") {
		t.Errorf("the record says %q, want the correction kept to the one line that is it", restart.Text)
	}
}

// A new task's record opens with the fact that nobody at a keyboard wrote
// it, which is the first thing a reader finding it on the board wants.
func TestANewTaskSaysAModelWroteIt(t *testing.T) {
	s, sn, r := oneRepo(t)
	answer := call(t, sn, "orbit_create_task", map[string]any{"title": "retry the webhook on 5xx"})
	id := str(t, answer["id"])

	wrote := only(t, kindsOn(t, s, r, id), record.TaskDialogue)
	if wrote.Data["by"] != "mcp" || !strings.Contains(wrote.Text, "over mcp") {
		t.Errorf("the record says %q by %q, want it to say a model wrote the task over mcp", wrote.Text, wrote.Data["by"])
	}
	if msg := str(t, answer["message"]); strings.Contains(msg, "not in the task's history") {
		t.Errorf("the tool says the trace could not be kept: %s", msg)
	}
}

// Reading is not something that happened to a task. A record that grew a
// line every time a supervisor looked at it would bury the four lines that
// say what it did — in the file the next run is folded from.
func TestReadingTheBoardLeavesNothingBehind(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-11", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})
	before := len(kindsOn(t, s, r, "PAY-11"))

	call(t, sn, "orbit_get_board_summary", nil)
	call(t, sn, "orbit_list_tasks", nil)
	call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-11"})
	call(t, sn, "orbit_list_repos", nil)
	call(t, sn, "orbit_list_flows", nil)

	if after := len(kindsOn(t, s, r, "PAY-11")); after != before {
		t.Errorf("reading the board wrote %d events into a task's record, want none", after-before)
	}
}

// A trace that could not be written is said and not swallowed. The act has
// already happened by then — the task really is cancelled — so the tool
// cannot answer no; what it must not do is let the model tell somebody the
// cockpit will show a call that is nowhere.
func TestATraceThatCouldNotBeWrittenIsSaid(t *testing.T) {
	s, _, r := oneRepo(t)
	got := journal(s, task.Task{ID: "has/slash", Repo: r}, "a model did something over mcp")
	if !strings.Contains(got, "not in the task's history") {
		t.Errorf("journal answered %q, want the clause a tool adds to its own sentence", got)
	}
	if said := journal(s, task.Task{ID: "PAY-1", Repo: r}, "a model did something over mcp"); said != "" {
		t.Errorf("journal answered %q on a trace it wrote, want nothing to add", said)
	}
	if correction("") != "" {
		t.Errorf("a restart with no instruction claims to have carried one: %q", correction(""))
	}
}

// A supervisor asking "has this already been tried" is asking about the
// dialogue, and orbit_inspect_task is where it asks.
func TestInspectReportsWhatWasAlreadyDoneToTheTask(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-13",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.TaskStarted})

	if res := sn.Call("orbit_pause_task", map[string]any{"task_id": "PAY-13"}); res.IsError {
		t.Fatalf("orbit_pause_task refused: %s", text(t, res))
	}
	answer := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-13"})
	said := list(t, answer["dialogue"])
	if len(said) != 1 {
		t.Fatalf("inspect reports %d things done from outside a run, want the pause: %v", len(said), said)
	}
	if by := str(t, obj(t, said[0])["by"]); by != "mcp" {
		t.Errorf("by = %q, want what acted", by)
	}
	if what := str(t, obj(t, said[0])["text"]); !strings.Contains(what, "pause") {
		t.Errorf("the entry reads %q, want what was done", what)
	}
	// And it is not folded in with the notes, which are what the next phase
	// is handed.
	if notes := answer["notes"]; notes != nil {
		t.Errorf("notes = %v, want none: nothing was written for the engine to read", notes)
	}
}
