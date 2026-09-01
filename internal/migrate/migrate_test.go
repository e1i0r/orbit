package migrate

// The files an older Orbit wrote, read into the database — and left exactly
// where they were.

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/db"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// root is a state root of this test's own, with one repository registered.
func root(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.New(filepath.Join(t.TempDir(), ".orbit"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	return s
}

// log writes one task's events.jsonl the way the older Orbit did.
func log(t *testing.T, s *store.Store, id string, events ...record.Event) {
	t.Helper()

	if _, err := s.CreateTaskDir(t.TempDir(), id); err != nil {
		t.Fatalf("create the directory of %s: %v", id, err)
	}

	path, err := s.EventsPath(id)
	if err != nil {
		t.Fatalf("events path of %s: %v", id, err)
	}

	at := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)

	for i, e := range events {
		e.At = at.Add(time.Duration(i) * time.Second)
		if err := record.Append(path, e); err != nil {
			t.Fatalf("append to the log of %s: %v", id, err)
		}
	}
}

// turns writes the supervisor thread the way the older Orbit did.
func turns(t *testing.T, s *store.Store, events ...record.Event) {
	t.Helper()

	at := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)

	for i, e := range events {
		e.At = at.Add(time.Duration(i) * time.Second)
		if err := record.Append(s.SupervisorLogPath(), e); err != nil {
			t.Fatalf("append to the supervisor thread: %v", err)
		}
	}
}

// opened is the record of a state root, closed when the test ends.
func opened(t *testing.T, s *store.Store) *db.DB {
	t.Helper()

	d, err := db.Open(s.DBPath())
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("close the record: %v", err)
		}
	})

	return d
}

// TestTheLogsBecomeRows. One task's history, read out of the file it was
// written to and into the database, with its runs and phases built on the
// way past — the migration writes through the same door every other writer
// does, so a migrated task is indistinguishable from a lived one.
func TestTheLogsBecomeRows(t *testing.T) {
	s := root(t)

	log(t, s, "ACME-1",
		record.Event{Kind: record.TaskCreated, Text: "Retry the webhook on 5xx", Data: map[string]string{"flow": "deliver"}},
		record.Event{Kind: record.TaskStarted},
		record.Event{Kind: record.PhaseStarted, Phase: "implement", Data: map[string]string{"engine": "claude", "model": "claude-opus-5"}},
		record.Event{Kind: record.PhaseFinished, Phase: "implement"},
		record.Event{Kind: record.TaskFinished},
	)

	out, err := Run(s)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if out.Tasks != 1 || out.Events != 5 {
		t.Errorf("the pass moved %s, want five events from one task", out)
	}

	d := opened(t, s)

	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("read the events: %v", err)
	}

	if len(events) != 5 || events[0].Text != "Retry the webhook on 5xx" {
		t.Fatalf("the record holds %d events, want the five that were in the file", len(events))
	}

	runs, err := d.Runs("ACME-1")
	if err != nil {
		t.Fatalf("read the runs: %v", err)
	}

	if len(runs) != 1 || runs[0].Outcome != record.TaskFinished {
		t.Fatalf("the migrated task has %d runs ending %q, want one that finished", len(runs), runs[0].Outcome)
	}

	if len(runs[0].Phases) != 1 || runs[0].Phases[0].Engine != "claude" {
		t.Errorf("the run's phases are %+v, want the one implement phase on claude", runs[0].Phases)
	}
}

// TestNothingIsDeleted. The previous binary has to keep working, and the way
// to be sure a migration did not eat a record is to still have it.
func TestNothingIsDeleted(t *testing.T) {
	s := root(t)

	log(t, s, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"})
	turns(t, s, record.Event{Kind: record.SupervisorMessage, Text: "start it"})

	path, err := s.EventsPath("ACME-1")
	if err != nil {
		t.Fatalf("events path: %v", err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the log: %v", err)
	}

	if _, err := Run(s); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the log after the migration: %v", err)
	}

	if string(before) != string(after) {
		t.Errorf("the log changed under the migration:\nwas %q\nnow %q", before, after)
	}

	if _, err := os.Stat(s.SupervisorLogPath()); err != nil {
		t.Errorf("the supervisor thread is gone after the migration: %v", err)
	}
}

// TestASecondPassCopiesNothing. This runs before every command, so the
// normal case is the one where there is nothing to do.
func TestASecondPassCopiesNothing(t *testing.T) {
	s := root(t)

	log(t, s, "ACME-1",
		record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"},
		record.Event{Kind: record.TaskStarted},
	)
	turns(t, s, record.Event{Kind: record.SupervisorMessage, Text: "start it"})

	if _, err := Run(s); err != nil {
		t.Fatalf("the first pass: %v", err)
	}

	out, err := Run(s)
	if err != nil {
		t.Fatalf("the second pass: %v", err)
	}

	if out.Moved() {
		t.Errorf("the second pass moved %s, want nothing", out)
	}

	events, err := opened(t, s).Events("ACME-1")
	if err != nil {
		t.Fatalf("read the events: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("the record holds %d events after two passes, want the two in the file", len(events))
	}
}

// TestAPassPicksUpWhereTheLastOneStopped. A root migrated by one version and
// then written to by the old binary again is not a case to refuse: the file
// is appended to and never reordered, so what is past the count is what is
// left to do.
func TestAPassPicksUpWhereTheLastOneStopped(t *testing.T) {
	s := root(t)

	log(t, s, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"})

	if _, err := Run(s); err != nil {
		t.Fatalf("the first pass: %v", err)
	}

	// The old binary carrying on, appending to a log that has already been
	// read once.
	log(t, s, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		record.Event{Kind: record.PhaseStarted, Phase: "implement"},
	)

	out, err := Run(s)
	if err != nil {
		t.Fatalf("the second pass: %v", err)
	}

	if out.Events != 2 {
		t.Errorf("the second pass moved %d events, want the two that were added", out.Events)
	}

	events, err := opened(t, s).Events("ACME-1")
	if err != nil {
		t.Fatalf("read the events: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("the record holds %d events, want the three in the file", len(events))
	}
}

// TestAnEmptyRootMigratesNothing. A machine that has never run Orbit is the
// first thing this meets, and it is not an error.
func TestAnEmptyRootMigratesNothing(t *testing.T) {
	out, err := Run(root(t))
	if err != nil {
		t.Fatalf("migrate an empty root: %v", err)
	}

	if out.Moved() {
		t.Errorf("an empty root moved %s, want nothing", out)
	}
}

// TestTheSupervisorThreadBecomesRows. It is the one conversation that
// belongs to no task, it had a file of its own, and it moves too.
func TestTheSupervisorThreadBecomesRows(t *testing.T) {
	s := root(t)

	said := record.Event{Kind: record.SupervisorMessage, Text: "cancel everything"}
	turns(t, s,
		said,
		record.Event{Kind: record.SupervisorBriefing, Text: "deliver ACME-1 tonight"},
	)

	out, err := Run(s)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if out.Messages != 2 {
		t.Errorf("the pass moved %d turns, want two", out.Messages)
	}

	thread, err := opened(t, s).Messages()
	if err != nil {
		t.Fatalf("read the thread: %v", err)
	}

	if len(thread) != 2 || thread[0].Text != said.Text {
		t.Fatalf("the record holds %d turns, want the two that were in the file, in order", len(thread))
	}

	if thread[1].Kind != record.SupervisorBriefing {
		t.Errorf("the second turn is a %q, want the briefing it was", thread[1].Kind)
	}
}
