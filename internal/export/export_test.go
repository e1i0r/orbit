package export

// The record written back out as the files it used to be, and the four ways
// that is refused or survived. The proof that what comes out goes back in —
// export, then the migration, then the same board — is a command-level test,
// in internal/cli, because it is the only package allowed to import both
// halves of the round trip.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/db"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// root is a state root of this test's own.
func root(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.New(filepath.Join(t.TempDir(), ".orbit"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	return s
}

// filled is a record with two tasks, a repository between them and a turn of
// the supervisor thread — one of everything an export has to carry.
func filled(t *testing.T) (*store.Store, *db.DB) {
	t.Helper()

	s := root(t)

	d, err := db.Open(s.DBPath())
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("close the record: %v", err)
		}
	})

	at := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	tick := 0
	next := func() time.Time {
		tick++

		return at.Add(time.Duration(tick) * time.Second)
	}

	for _, id := range []string{"T-1", "T-2"} {
		if err := d.Append(id, record.Event{At: next(), Kind: record.TaskCreated, Text: "the task called " + id}); err != nil {
			t.Fatalf("write down %s: %v", id, err)
		}

		if err := d.Append(id, record.Event{At: next(), Kind: record.PhaseFinished, Phase: "implement", Text: "done"}); err != nil {
			t.Fatalf("finish a phase of %s: %v", id, err)
		}
	}

	if err := d.Join("T-1", "/repos/payments", "payments", next()); err != nil {
		t.Fatalf("join a repository: %v", err)
	}

	if err := d.AppendMessage(record.Event{At: next(), Kind: record.SupervisorMessage, Text: "retry T-1"}); err != nil {
		t.Fatalf("say something to the supervisor: %v", err)
	}

	return s, d
}

// TestTheRecordComesBackOutAsFiles. One events.jsonl per task, the thread
// beside them, and the marker saying where a task was worked — the tree the
// database replaced, generated on demand.
func TestTheRecordComesBackOutAsFiles(t *testing.T) {
	_, from := filled(t)

	to := root(t)

	out, err := Records(from, to, "")
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	// Two events each: the task was written down, and a phase finished. The
	// repository T-1 joined is not an event — the record keeps that link in
	// a row of its own, which is why the marker below is exported at all.
	if out.Tasks != 2 || out.Events != 4 || out.Messages != 1 {
		t.Errorf("the export wrote %+v, want 2 tasks, 4 events and 1 turn", out)
	}

	path, err := to.EventsPath("T-1")
	if err != nil {
		t.Fatalf("events path: %v", err)
	}

	events, err := record.Read(path)
	if err != nil {
		t.Fatalf("read the exported log: %v", err)
	}

	if len(events) != 2 || events[0].Kind != record.TaskCreated || events[0].Text != "the task called T-1" {
		t.Errorf("T-1 came out as %d events beginning %+v", len(events), events[0])
	}

	repos, err := to.TaskRepos("T-1")
	if err != nil {
		t.Fatalf("read the marker: %v", err)
	}

	if len(repos) != 1 || repos[0] != "/repos/payments" {
		t.Errorf("T-1 came out worked in %q, want the repository it joined", repos)
	}

	thread, err := record.Read(to.SupervisorLogPath())
	if err != nil {
		t.Fatalf("read the exported thread: %v", err)
	}

	if len(thread) != 1 || thread[0].Text != "retry T-1" {
		t.Errorf("the thread came out as %+v", thread)
	}
}

// TestOneTaskIsThatTaskAlone. The thread belongs to no task, so a directory
// named after one must not end up holding the whole of it.
func TestOneTaskIsThatTaskAlone(t *testing.T) {
	_, from := filled(t)

	to := root(t)

	out, err := Records(from, to, "T-2")
	if err != nil {
		t.Fatalf("export one task: %v", err)
	}

	if out.Tasks != 1 || out.Messages != 0 {
		t.Errorf("the export wrote %+v, want the one task and no thread", out)
	}

	if _, err := os.Stat(to.SupervisorLogPath()); err == nil {
		t.Error("a one-task export left a supervisor thread behind it")
	}

	other, err := to.EventsPath("T-1")
	if err != nil {
		t.Fatalf("events path: %v", err)
	}

	if _, err := os.Stat(other); err == nil {
		t.Error("a one-task export wrote the other task out as well")
	}
}

// TestATaskTheRecordNeverHeardOfIsRefused. A mistyped id answered with an
// empty directory would be a backup of nothing that looks like a backup.
func TestATaskTheRecordNeverHeardOfIsRefused(t *testing.T) {
	_, from := filled(t)

	out, err := Records(from, root(t), "T-9")
	if err == nil {
		t.Fatalf("exporting a task that is not there wrote %+v", out)
	}

	if !strings.Contains(err.Error(), "T-9") {
		t.Errorf("the refusal is %q, want the id that was asked for in it", err)
	}
}

// TestADestinationThatHoldsSomethingIsRefused. Half of yesterday's record
// under half of today's is a directory nobody can reason about, and a state
// root is worse: the migration would read the export back in as a log an
// older Orbit wrote.
func TestADestinationThatHoldsSomethingIsRefused(t *testing.T) {
	s, _ := filled(t)

	dir := t.TempDir() // t.TempDir makes it, which is what makes it not empty

	if err := record.Write(filepath.Join(dir, "supervisor.jsonl"), nil); err != nil {
		t.Fatalf("put something in the way: %v", err)
	}

	if _, err := Run(s, dir, ""); err == nil {
		t.Fatal("the export landed on top of what was already there")
	}
}

// TestADirectoryThatIsNotThereYetIsMade. The ordinary way to run this names
// a backup that does not exist, and the refusal above must not extend to it.
func TestADirectoryThatIsNotThereYetIsMade(t *testing.T) {
	s, _ := filled(t)

	dir := filepath.Join(t.TempDir(), "backup")

	out, err := Run(s, dir, "")
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if out.Tasks != 2 {
		t.Errorf("the export wrote %+v, want both tasks", out)
	}
}

// TestATaskThatWillNotComeOutDoesNotStopTheRest. The reason to run an export
// is often that the record is already doubted, and a run that gave up on the
// first task would leave every readable one after it in the file nobody can
// read. It still fails: what did not come out is named, so a partial backup
// cannot pass for a whole one.
func TestATaskThatWillNotComeOutDoesNotStopTheRest(t *testing.T) {
	_, from := filled(t)

	to := root(t)

	// A file where T-1's directory has to go. Nothing can be written under
	// it, which is this test's stand-in for a page of the record that will
	// not read: the loop meets one task it cannot finish and two it can.
	blocked, err := to.TaskDir("T-1")
	if err != nil {
		t.Fatalf("task dir: %v", err)
	}

	if err := record.Write(blocked, nil); err != nil {
		t.Fatalf("put a file where a directory has to go: %v", err)
	}

	out, err := Records(from, to, "")
	if err == nil {
		t.Fatal("an export that could not write a task reported success")
	}

	if !strings.Contains(err.Error(), "T-1") {
		t.Errorf("the failure is %q, want the task that stayed behind in it", err)
	}

	if out.Tasks != 1 || out.Messages != 1 {
		t.Errorf("the export wrote %+v, want the other task and the thread", out)
	}
}
