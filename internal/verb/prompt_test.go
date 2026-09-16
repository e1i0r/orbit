package verb

// Reading back the prompt a phase was given.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// onARecord is a world with one task written down and whatever its record
// has been told. The reading reaches the store and the task and no further.
type onARecord struct {
	World

	store *store.Store
	task  task.Task
}

func (o onARecord) Store() *store.Store   { return o.store }
func (o onARecord) Words() *words.Printer { return words.For("") }

func (o onARecord) Find(string, string) (task.Task, repo.Repo, error) {
	return o.task, repo.Repo{}, nil
}

// ranWith is a world whose task has been asked each of these prompts, one
// phase after another.
func ranWith(t *testing.T, asked ...record.Event) onARecord {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() })

	one, err := task.Create(s, repo.Repo{}, "ACME-1", "do the thing", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	d, err := s.Record()
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	at := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	for i, e := range asked {
		e.At = at.Add(time.Duration(i) * time.Minute)
		if err := d.Append(one.ID, e); err != nil {
			t.Fatalf("append %s: %v", e.Kind, err)
		}
	}

	return onARecord{store: s, task: one}
}

// askedIn is one phase.asked as the record holds it.
func askedIn(phase, engine, text string) record.Event {
	return record.Event{
		Kind: record.PhaseAsked, Phase: phase, Text: text,
		Data: map[string]string{"engine": engine},
	}
}

// TestTheLastPromptIsTheOneItAnswersWith. A phase that ran three times ran
// three times for a reason, and the reader is almost always asking about the
// run they just watched.
func TestTheLastPromptIsTheOneItAnswersWith(t *testing.T) {
	w := ranWith(t,
		askedIn("plan", "claude", "# ACME-1\n\nplan it"),
		record.Event{Kind: record.PhaseStarted, Phase: "implement"},
		askedIn("implement", "codex", "# ACME-1\n\nwrite it"),
	)

	out, err := prompted(w, In{Task: "ACME-1"})
	if err != nil {
		t.Fatalf("prompted: %v", err)
	}

	if out.Said != "# ACME-1\n\nwrite it" {
		t.Errorf("it read back %q, want the last prompt", out.Said)
	}

	seen, ok := out.Saw.(promptSeen)
	if !ok {
		t.Fatalf("it saw %T, want a promptSeen", out.Saw)
	}

	// The engine that was given it, which after a relay is not the one the
	// flow named.
	if seen.Phase != "implement" || seen.Engine != "codex" {
		t.Errorf("it read phase %q on %q, want implement on codex", seen.Phase, seen.Engine)
	}
}

// TestOnePhaseCanBeAskedForByName.
func TestOnePhaseCanBeAskedForByName(t *testing.T) {
	w := ranWith(t,
		askedIn("plan", "claude", "plan it"),
		askedIn("implement", "codex", "write it"),
	)

	out, err := prompted(w, In{Task: "ACME-1", Args: map[string]string{"phase": "plan"}})
	if err != nil {
		t.Fatalf("prompted: %v", err)
	}

	if out.Said != "plan it" {
		t.Errorf("it read back %q, want the plan phase's prompt", out.Said)
	}
}

// TestTheTwoNothingsAreDifferentSentences.
//
// A task that has never run and a task whose phases all ran before Orbit
// wrote prompts down look identical from here, and a reader told only
// "nothing" goes looking for a bug.
func TestTheTwoNothingsAreDifferentSentences(t *testing.T) {
	never, err := prompted(ranWith(t), In{Task: "ACME-1"})
	if err != nil {
		t.Fatalf("prompted: %v", err)
	}

	if !strings.Contains(never.Said, "has not run yet") {
		t.Errorf("a task that never ran says %q", never.Said)
	}

	before, err := prompted(ranWith(t,
		record.Event{Kind: record.PhaseStarted, Phase: "implement"},
	), In{Task: "ACME-1"})
	if err != nil {
		t.Fatalf("prompted: %v", err)
	}

	if !strings.Contains(before.Said, "before Orbit kept the prompts") {
		t.Errorf("a task that ran before prompts were kept says %q", before.Said)
	}

	// And a phase nobody ran is named, so the reader can see they mistyped
	// it rather than wondering whether the phase went unrecorded.
	missing, err := prompted(ranWith(t, askedIn("plan", "claude", "plan it")),
		In{Task: "ACME-1", Args: map[string]string{"phase": "reviw"}})
	if err != nil {
		t.Fatalf("prompted: %v", err)
	}

	if !strings.Contains(missing.Said, `"reviw"`) {
		t.Errorf("a phase that is not there says %q", missing.Said)
	}
}
