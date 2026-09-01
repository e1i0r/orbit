package db

// Which tasks belong to which repository — the one question about tasks that
// no fold answers, because it starts from the repository and crosses all of
// them.

import (
	"slices"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// wrote is a task written down against a repository, the way task.Create
// writes one: the repository is named in the event itself.
func wrote(t *testing.T, d *DB, id, name, abs string) {
	t.Helper()

	e := record.Event{
		Kind: record.TaskCreated,
		Text: id + " was written by a test",
		Data: map[string]string{"repo": name, "path": abs},
	}

	if err := d.Append(id, e); err != nil {
		t.Fatalf("write %s down: %v", id, err)
	}
}

// TestARepositoryListsTheTasksWrittenAgainstIt. The listing is by id and not
// by when the task was written, because that is the order a board draws and
// the order the directory listing this replaced gave.
func TestARepositoryListsTheTasksWrittenAgainstIt(t *testing.T) {
	d := open(t)

	wrote(t, d, "ACME-2", "app", "/w/app")
	wrote(t, d, "ACME-1", "app", "/w/app")
	wrote(t, d, "LED-1", "ledger", "/w/ledger")

	got, err := d.TasksOfRepo("/w/app")
	if err != nil {
		t.Fatalf("the tasks of /w/app: %v", err)
	}

	if !slices.Equal(got, []string{"ACME-1", "ACME-2"}) {
		t.Errorf("/w/app lists %v, want [ACME-1 ACME-2] in order", got)
	}
}

// TestADeletedTaskLeavesTheListingAndKeepsItsEvents. The two halves of the
// soft delete meet here: the enumeration every caller reads leaves the task
// out, and the record the enumeration read it out of is untouched.
func TestADeletedTaskLeavesTheListingAndKeepsItsEvents(t *testing.T) {
	d := open(t)

	wrote(t, d, "ACME-1", "app", "/w/app")
	wrote(t, d, "ACME-2", "app", "/w/app")

	if err := d.Append("ACME-1", record.Event{Kind: record.TaskDeleted}); err != nil {
		t.Fatalf("delete ACME-1: %v", err)
	}

	got, err := d.TasksOfRepo("/w/app")
	if err != nil {
		t.Fatalf("the tasks of /w/app: %v", err)
	}

	if !slices.Equal(got, []string{"ACME-2"}) {
		t.Errorf("/w/app lists %v, want [ACME-2] — the deleted task is off the listing", got)
	}

	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("read the record of the deleted task: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("the deleted task has %d events, want the two that were written: what it did is the part a delete does not take away", len(events))
	}
}

// TestARepositoryNobodyHasWrittenAgainstListsNothing. Nothing rather than a
// failure: a checkout Orbit knows and has never been given a task is the
// ordinary state of a repository somebody has just added.
func TestARepositoryNobodyHasWrittenAgainstListsNothing(t *testing.T) {
	d := open(t)

	wrote(t, d, "ACME-1", "app", "/w/app")

	got, err := d.TasksOfRepo("/w/nothing")
	if err != nil {
		t.Fatalf("the tasks of a repository with none: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("a repository nobody has written against lists %v", got)
	}
}

// TestATaskWrittenByAnOlderOrbitBelongsToNothing. Every task.created before
// this version carries a flow and nothing else, and a link cannot be guessed
// from one: the migration reads it out of the file the older Orbit kept it
// in. What must not happen is the task joining a repository named "".
func TestATaskWrittenByAnOlderOrbitBelongsToNothing(t *testing.T) {
	d := open(t)

	if err := d.Append("ACME-1", record.Event{Kind: record.TaskCreated, Text: "written before the record had rows"}); err != nil {
		t.Fatalf("append: %v", err)
	}

	got, err := d.ReposOfTask("ACME-1")
	if err != nil {
		t.Fatalf("the repositories of ACME-1: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("a task written with no repository belongs to %v", got)
	}
}

// TestATaskReachesIntoASecondCheckout. One task, two repositories: the one
// it was written against and one a worktree was opened in. Both are the
// task's, oldest join first, and neither replaces the other.
func TestATaskReachesIntoASecondCheckout(t *testing.T) {
	d := open(t)

	wrote(t, d, "ACME-1", "app", "/w/app")

	joining := record.Event{Kind: record.RepoJoined, Data: map[string]string{"repo": "api", "path": "/w/api"}}
	if err := d.Append("ACME-1", joining); err != nil {
		t.Fatalf("join the second checkout: %v", err)
	}

	got, err := d.ReposOfTask("ACME-1")
	if err != nil {
		t.Fatalf("the repositories of ACME-1: %v", err)
	}

	if !slices.Equal(got, []string{"/w/app", "/w/api"}) {
		t.Errorf("ACME-1 belongs to %v, want both in the order they joined", got)
	}

	for _, abs := range []string{"/w/app", "/w/api"} {
		ids, err := d.TasksOfRepo(abs)
		if err != nil {
			t.Fatalf("the tasks of %s: %v", abs, err)
		}

		if !slices.Equal(ids, []string{"ACME-1"}) {
			t.Errorf("%s lists %v, want ACME-1", abs, ids)
		}
	}
}

// TestJoiningTwiceIsJoiningOnce. repo.joined is appended every time a
// worktree is opened, which is every retry, and Join is called by a
// migration that runs before every command. Both have to be harmless the
// second time or a task would belong to one repository several times over.
func TestJoiningTwiceIsJoiningOnce(t *testing.T) {
	d := open(t)

	at := time.Date(2026, 8, 20, 11, 0, 0, 0, time.UTC)

	for range 3 {
		if err := d.Join("ACME-1", "/w/app", "app", at); err != nil {
			t.Fatalf("join: %v", err)
		}

		joining := record.Event{Kind: record.RepoJoined, Data: map[string]string{"repo": "app", "path": "/w/app"}}
		if err := d.Append("ACME-1", joining); err != nil {
			t.Fatalf("append a join: %v", err)
		}
	}

	got, err := d.ReposOfTask("ACME-1")
	if err != nil {
		t.Fatalf("the repositories of ACME-1: %v", err)
	}

	if !slices.Equal(got, []string{"/w/app"}) {
		t.Errorf("after joining the same repository six times the task belongs to %v", got)
	}
}

// TestJoinMakesTheTaskItNamesRatherThanRefusing. Join is what the migration
// writes for a state root whose link was a line in a file, and such a state
// root can hold a task directory whose log is empty — a create killed
// between the directory and the event. The link is still a fact about that
// task, and dropping it would lose the only thing anybody knows about it.
func TestJoinMakesTheTaskItNamesRatherThanRefusing(t *testing.T) {
	d := open(t)

	at := time.Date(2026, 8, 20, 11, 0, 0, 0, time.UTC)
	if err := d.Join("ACME-1", "/w/app", "app", at); err != nil {
		t.Fatalf("join a task nothing has been recorded about: %v", err)
	}

	ids, err := d.TasksOfRepo("/w/app")
	if err != nil {
		t.Fatalf("the tasks of /w/app: %v", err)
	}

	if !slices.Equal(ids, []string{"ACME-1"}) {
		t.Errorf("/w/app lists %v, want ACME-1", ids)
	}

	// And it left no event behind. The link was never something that
	// happened at a time in the old state root, so nothing may appear in the
	// task's history saying it was.
	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("the events of ACME-1: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("carrying a link across put %d event(s) in the task's history", len(events))
	}
}

// TestTheLinkOfAClosedRecordFails, for the same reason every other read of
// one does: a command that asks after the close has a bug in it, and an
// empty answer reads as a repository with no tasks.
func TestTheLinkOfAClosedRecordFails(t *testing.T) {
	d, err := Open(t.TempDir() + "/orbit.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	wrote(t, d, "ACME-1", "app", "/w/app")

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := d.TasksOfRepo("/w/app"); err == nil {
		t.Error("reading the tasks of a repository from a closed record answered cleanly")
	}

	if _, err := d.ReposOfTask("ACME-1"); err == nil {
		t.Error("reading the repositories of a task from a closed record answered cleanly")
	}

	if err := d.Join("ACME-1", "/w/app", "app", time.Now()); err == nil {
		t.Error("joining a repository in a closed record answered cleanly")
	}
}
