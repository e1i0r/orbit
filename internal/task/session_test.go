package task

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

func TestLastSessionReturnsEmptyWhenNilOrNotResumable(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "SESS-1", "session test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if sess := lastSession(nil, tk, "fake", &engine.Fake{}); sess != "" {
		t.Errorf("lastSession on nil store = %q, want empty", sess)
	}

	if sess := lastSession(s, tk, "fake", nil); sess != "" {
		t.Errorf("lastSession on nil engine = %q, want empty", sess)
	}

	if sess := lastSession(s, tk, "fake", &engine.Fake{Resumable: false}); sess != "" {
		t.Errorf("lastSession on non-resumable fake = %q, want empty", sess)
	}

	if sess := lastSession(s, tk, "", &engine.Fake{Resumable: true}); sess != "" {
		t.Errorf("lastSession with no engine named = %q, want empty", sess)
	}
}

func TestLastSessionFindsMostRecentSessionID(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "SESS-2", "session test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := &engine.Fake{Resumable: true}

	// No session events yet
	if sess := lastSession(s, tk, "fake", fake); sess != "" {
		t.Errorf("lastSession before runs = %q, want empty", sess)
	}

	// A phase, as a run writes one: started, naming the engine, then ended,
	// carrying the session. The pair is what the walk reads.
	phase(t, s, tk, "plan", "fake", record.PhaseFinished, "sess-alpha")

	if sess := lastSession(s, tk, "fake", fake); sess != "sess-alpha" {
		t.Errorf("lastSession = %q, want sess-alpha", sess)
	}

	phase(t, s, tk, "impl", "fake", record.PhaseCancelled, "sess-beta")

	if sess := lastSession(s, tk, "fake", fake); sess != "sess-beta" {
		t.Errorf("lastSession after cancel = %q, want sess-beta", sess)
	}
}

// TestLastSessionIgnoresAnotherEnginesSession is the fix. A flow may name a
// different engine on each phase, and a session id belongs to the tool that
// issued it: handing codex a claude session is either an error the user has
// to decipher or a silent fresh start reported as a resume.
func TestLastSessionIgnoresAnotherEnginesSession(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "SESS-4", "two engines, one task", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := &engine.Fake{Resumable: true}

	phase(t, s, tk, "plan", "claude", record.PhaseFinished, "claude-sess")

	if sess := lastSession(s, tk, "codex", fake); sess != "" {
		t.Errorf("lastSession for codex = %q, want empty — that session is claude's", sess)
	}

	if sess := lastSession(s, tk, "claude", fake); sess != "claude-sess" {
		t.Errorf("lastSession for claude = %q, want claude-sess", sess)
	}

	// Now codex runs a phase of its own. Each engine keeps its own thread,
	// and the later one does not shadow the earlier.
	phase(t, s, tk, "check", "codex", record.PhaseFinished, "codex-sess")

	if sess := lastSession(s, tk, "codex", fake); sess != "codex-sess" {
		t.Errorf("lastSession for codex = %q, want codex-sess", sess)
	}

	if sess := lastSession(s, tk, "claude", fake); sess != "claude-sess" {
		t.Errorf("lastSession for claude = %q, want claude-sess — codex's phase is not claude's", sess)
	}
}

// phase writes the two events a run writes around one phase: the start that
// names the engine, and the ending that carries the session.
func phase(t *testing.T, s *store.Store, tk Task, name, eng, ending, session string) {
	t.Helper()

	for _, e := range []record.Event{
		{Kind: record.PhaseStarted, Phase: name, Data: map[string]string{"engine": eng, "n": "1"}},
		{Kind: ending, Phase: name, Data: map[string]string{"session": session}},
	} {
		if err := emit(s, tk, e); err != nil {
			t.Fatalf("emit %s: %v", e.Kind, err)
		}
	}
}

func TestRunPassesLastSessionToResumableEngine(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "SESS-3", "run session test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// An earlier phase of this task, run by the same engine the flow below
	// names, so its session is one this run may pick up.
	phase(t, s, tk, "setup", "fake", record.PhaseFinished, "sess-existing")

	fake := &engine.Fake{
		Output:    "all done",
		SessionID: "sess-next",
		Resumable: true,
	}
	engines := map[string]engine.Engine{"fake": fake}
	f := flow.Flow{
		Name: "single",
		Phases: []flow.Phase{
			{Name: "step1", Engine: "fake"},
		},
	}

	if err := Run(context.Background(), s, tk, f, engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("fake.Calls length = %d, want 1", len(fake.Calls))
	}

	if fake.Calls[0].Resume != "sess-existing" {
		t.Errorf("fake.Calls[0].Resume = %q, want sess-existing", fake.Calls[0].Resume)
	}
}

// TestAPhaseWhoseRecordWillNotReadOpensAFreshSession.
//
// Resuming is best-effort on purpose: an engine that has forgotten the last
// phase still works, and failing the whole run over a log the reader could
// not parse would cost more than it saves. What the read error must not do
// is disappear -- a phase that quietly loses everything the phase before it
// knew is a run nobody can explain afterwards, which is why this one goes to
// the log on its way past.
func TestAPhaseWhoseRecordWillNotReadOpensAFreshSession(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "SESS-UNREADABLE-1", "resume over a log that will not read", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	path, err := s.EventsPath(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	// One line over record.MaxLine, which trips the scanner rather than
	// yielding a record.unreadable event.
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 5<<20)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := lastSession(s, tk, "fake", &engine.Fake{Resumable: true}); got != "" {
		t.Errorf("lastSession over an unreadable log = %q, want the empty id that starts a new session", got)
	}
}
