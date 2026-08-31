package index

// The index read from the side that matters: it says what the logs say, it
// can be thrown away, and folding twice is folding once.

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// logs is a Source made of files in one directory, which is the shape
// internal/store answers with.
type logs struct {
	dir string
	ids []string
}

func (l logs) TaskIDs() ([]string, error) { return l.ids, nil }

func (l logs) EventsPath(taskID string) (string, error) {
	return filepath.Join(l.dir, taskID+".jsonl"), nil
}

// written is a source holding one task, with the events already appended.
func written(t *testing.T, taskID string, events ...record.Event) logs {
	t.Helper()

	src := logs{dir: t.TempDir(), ids: []string{taskID}}
	appendTo(t, src, taskID, events...)

	return src
}

func appendTo(t *testing.T, src logs, taskID string, events ...record.Event) {
	t.Helper()

	path, err := src.EventsPath(taskID)
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}

	for _, e := range events {
		if err := record.Append(path, e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
}

func opened(t *testing.T) *Index {
	t.Helper()

	x, err := Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	t.Cleanup(func() {
		if err := x.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	return x
}

func at(m int) time.Time {
	return time.Date(2026, 8, 28, 9, m, 0, 0, time.UTC)
}

func TestTheIndexSaysWhatTheLogSays(t *testing.T) {
	src := written(t, "ACME-1",
		record.Event{At: at(0), Kind: record.TaskCreated, Text: "retry the webhook on 5xx", Data: map[string]string{"flow": "task"}},
		record.Event{At: at(1), Kind: record.RepoJoined, Data: map[string]string{"repo": "app", "path": "/w/app"}},
		record.Event{At: at(2), Kind: record.RepoJoined, Data: map[string]string{"repo": "api", "path": "/w/api"}},
	)

	x := opened(t)

	folded, err := Build(x, src)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if folded != 3 {
		t.Errorf("the build folded %d events, want 3", folded)
	}

	// The row the whole index exists for: one task, two repositories, asked
	// from either end.
	repos, err := x.ReposOfTask("ACME-1")
	if err != nil {
		t.Fatalf("ReposOfTask: %v", err)
	}

	if len(repos) != 2 || repos[0] != "/w/app" || repos[1] != "/w/api" {
		t.Errorf("the task was worked in %v, want both repositories in the order they joined", repos)
	}

	for _, repo := range []string{"/w/app", "/w/api"} {
		ids, err := x.TaskIDsOfRepo(repo)
		if err != nil {
			t.Fatalf("TaskIDsOfRepo %s: %v", repo, err)
		}

		if len(ids) != 1 || ids[0] != "ACME-1" {
			t.Errorf("%s lists %v, want ACME-1", repo, ids)
		}
	}
}

// Building is also keeping it current: the second pass reads the tail and
// nothing else, and a pass over a log nobody appended to folds nothing.
func TestASecondBuildFoldsOnlyWhatWasAppended(t *testing.T) {
	src := written(t, "ACME-1", record.Event{At: at(0), Kind: record.TaskCreated, Text: "one"})
	x := opened(t)

	if _, err := Build(x, src); err != nil {
		t.Fatalf("Build: %v", err)
	}

	folded, err := Build(x, src)
	if err != nil {
		t.Fatalf("Build again: %v", err)
	}

	if folded != 0 {
		t.Errorf("the second build folded %d events of a log nobody appended to", folded)
	}

	appendTo(t, src, "ACME-1", record.Event{At: at(1), Kind: record.RepoJoined, Data: map[string]string{"path": "/w/app"}})

	folded, err = Build(x, src)
	if err != nil {
		t.Fatalf("Build after the append: %v", err)
	}

	if folded != 1 {
		t.Errorf("the build folded %d events, want the one that was appended", folded)
	}

	// And the event is in there once, not twice.
	n, err := x.CountOfKind(record.TaskCreated)
	if err != nil {
		t.Fatalf("CountOfKind: %v", err)
	}

	if n != 1 {
		t.Errorf("task.created is in the index %d times, want once", n)
	}
}

// The index holds nothing the record does not, and this is what that buys:
// delete it, ask again, get the same answer.
func TestAnIndexThrownAwayComesBackTheSame(t *testing.T) {
	src := written(t, "ACME-1",
		record.Event{At: at(0), Kind: record.TaskCreated, Text: "one", Data: map[string]string{"flow": "task"}},
		record.Event{At: at(1), Kind: record.RepoJoined, Data: map[string]string{"path": "/w/app"}},
	)

	x := opened(t)

	if _, err := Build(x, src); err != nil {
		t.Fatalf("Build: %v", err)
	}

	folded, err := Rebuild(x, src)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	if folded != 2 {
		t.Errorf("the rebuild folded %d events, want the whole log", folded)
	}

	repos, err := x.ReposOfTask("ACME-1")
	if err != nil {
		t.Fatalf("ReposOfTask: %v", err)
	}

	if len(repos) != 1 || repos[0] != "/w/app" {
		t.Errorf("after the rebuild the task was worked in %v, want /w/app", repos)
	}
}

// An index folded by another version of Orbit is emptied rather than read.
// There is nothing in it worth keeping and no migration to get wrong.
func TestAnIndexFromAnotherVersionIsEmptied(t *testing.T) {
	src := written(t, "ACME-1", record.Event{At: at(0), Kind: record.TaskCreated, Text: "one"})
	path := filepath.Join(t.TempDir(), "index.db")

	x, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if _, err := Build(x, src); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if _, err := x.db.Exec(`PRAGMA user_version = 999`); err != nil {
		t.Fatalf("stamp another version: %v", err)
	}

	if err := x.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	again, err := Open(path)
	if err != nil {
		t.Fatalf("Open again: %v", err)
	}

	defer func() {
		if err := again.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	n, err := again.CountOfKind(record.TaskCreated)
	if err != nil {
		t.Fatalf("CountOfKind: %v", err)
	}

	if n != 0 {
		t.Errorf("an index from another version came back holding %d events", n)
	}

	// And it is not damage: the next build fills it from the logs again.
	if _, err := Build(again, src); err != nil {
		t.Fatalf("Build over the emptied index: %v", err)
	}

	if n, err = again.CountOfKind(record.TaskCreated); err != nil || n != 1 {
		t.Errorf("after the rebuild task.created is in the index %d times (%v), want once", n, err)
	}
}

// A log replaced rather than appended to — which is what a restored backup
// or a hand-edited file looks like from here — leaves no rows behind from
// the log that used to be there.
func TestALogThatShrankIsFoldedFromTheTop(t *testing.T) {
	src := written(t, "ACME-1",
		record.Event{At: at(0), Kind: record.TaskCreated, Text: "the long one"},
		record.Event{At: at(1), Kind: record.RepoJoined, Data: map[string]string{"path": "/w/app"}},
		record.Event{At: at(2), Kind: record.TaskStuck, Data: map[string]string{"attempts": "3"}},
	)

	x := opened(t)

	if _, err := Build(x, src); err != nil {
		t.Fatalf("Build: %v", err)
	}

	path, err := src.EventsPath("ACME-1")
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}

	if err := os.WriteFile(path, []byte(`{"at":"2026-08-28T09:00:00Z","kind":"task.created","text":"short"}`+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Build(x, src); err != nil {
		t.Fatalf("Build over the replaced log: %v", err)
	}

	if n, err := x.CountOfKind(record.TaskStuck); err != nil || n != 0 {
		t.Errorf("the index still holds %d task.stuck (%v) from a log that no longer says it", n, err)
	}

	repos, err := x.ReposOfTask("ACME-1")
	if err != nil {
		t.Fatalf("ReposOfTask: %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("the task is still worked in %v, and the log that said so is gone", repos)
	}
}
