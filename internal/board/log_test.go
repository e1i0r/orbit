package board

// The two questions the task view asks: one task's whole record, and where
// its worktree is. Both are read here rather than through the window's own
// fake, because a port asserted only against a fake is a port nobody has
// run.

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestTheLogIsTheWholeRecordInOrder is the log tab's source, read the slow
// way: a task the poller has never seen.
func TestTheLogIsTheWholeRecordInOrder(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent(), failedEvent())

	entries, err := NewReader(s, work).Log(repoPath, "ACME-1")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("the record folded to %d entries, want 3", len(entries))
	}
	if entries[0].Text != "Retry the webhook on 5xx" {
		t.Errorf("the first entry says %q, want the title it was written down with", entries[0].Text)
	}
	// The attempt is the seam the log tab draws, and it has to be a fact
	// about the entry rather than a string the window matches on.
	if entries[0].Attempt != 0 || entries[1].Attempt != 1 || entries[2].Attempt != 1 {
		t.Errorf("the attempts are %d, %d, %d; want 0, 1, 1",
			entries[0].Attempt, entries[1].Attempt, entries[2].Attempt)
	}
}

// TestTheLogIsAnsweredFromWhatThePollerAlreadyRead is the fast path, and it
// is the reason this method is on the Reader at all: the log tab of a
// running task is redrawn twice a second.
func TestTheLogIsAnsweredFromWhatThePollerAlreadyRead(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())
	r := NewReader(s, work)
	refresh(t, r)

	appendTo(t, s, repoPath, "ACME-1", failedEvent())
	before, err := r.Log(repoPath, "ACME-1")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(before) != 2 {
		t.Fatalf("the cached record folded to %d entries, want the 2 the poll had read", len(before))
	}
	refresh(t, r)
	after, err := r.Log(repoPath, "ACME-1")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(after) != 3 {
		t.Errorf("after another poll the record folded to %d entries, want 3", len(after))
	}
}

// TestALogNobodyCanReadSaysWhichTask covers the failure the window has to be
// able to put in the pane. A wrapped error naming the id is what tells a
// reader which of the rows they were looking at is the unreadable one.
//
// The damage is a line longer than the reader's buffer, because that is the
// one kind the record refuses outright: a line of broken JSON is folded into
// an unreadable event and the rest of the log still reads, which is the
// behaviour a log tab wants and not a failure at all.
func TestALogNobodyCanReadSaysWhichTask(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-404", created("Retry the webhook on 5xx"))
	tooLongLine(t, eventsPath(t, s, repoPath, "ACME-404"))

	if _, err := NewReader(s, work).Log(repoPath, "ACME-404"); err == nil {
		t.Fatal("a record nobody can read came back as a record")
	} else if !strings.Contains(err.Error(), "ACME-404") {
		t.Errorf("the failure says %q, want it to name the task", err)
	}
}

// TestATaskNobodyHasWrittenDownIsAnEmptyLog is the other half, and it is not
// an error: a task directory with no record yet is a task nothing has
// happened to, and the pane says so in words rather than in a failure.
func TestATaskNobodyHasWrittenDownIsAnEmptyLog(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	entries, err := NewReader(s, work).Log(repoPath, "ACME-404")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("a task nothing has happened to folded to %d entries", len(entries))
	}
}

// TestTheWorktreeIsTheStoresAnswerAndNotTheRepository is defect 2 of this
// task, one layer down. The window may not import internal/store, so this is
// the only place the two paths can be compared at all — and they must not be
// the same path.
func TestTheWorktreeIsTheStoresAnswerAndNotTheRepository(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	got, err := NewReader(s, work).Worktree(repoPath, "ACME-1")
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	want, err := s.WorktreeDir(repoPath, "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}
	if got != want {
		t.Errorf("Worktree = %q, want the store's own answer %q", got, want)
	}
	if got == repoPath || strings.HasPrefix(got, repoPath+string(filepath.Separator)) {
		t.Errorf("Worktree = %q, which is inside the repository at %q", got, repoPath)
	}
	if !strings.HasSuffix(got, filepath.Join("worktrees", filepath.Base(filepath.Dir(got)), "ACME-1")) {
		t.Errorf("Worktree = %q, want it under the state root's worktrees", got)
	}
}

// TestAWorktreeForATaskThatCannotBeNamedIsRefused: the id becomes a
// directory name, so a store that took any string would take one that walks
// out of the state root.
func TestAWorktreeForATaskThatCannotBeNamedIsRefused(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	if path, err := NewReader(s, work).Worktree(repoPath, "../../etc"); err == nil {
		t.Errorf("a task id that walks out of the root was answered with %q", path)
	}
}
