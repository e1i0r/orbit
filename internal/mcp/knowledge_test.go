package mcp

// The two tools that read and write what Orbit knows.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// TestAnAgentCanWriteDownWhatItLearned.
//
// This is the source that makes Knowledge grow without anybody typing: an
// agent that hit a wall mid-task writes down what it found, and the next run
// against that code is told before it starts.
func TestAnAgentCanWriteDownWhatItLearned(t *testing.T) {
	s, work, r := workingOn(t, "PAY-1")
	sn := Session{Root: work, Version: "test"}

	got := sn.Call("orbit_learn", map[string]any{
		"phrase": "the fuzz tests hang when the seed is fixed", "task_id": "PAY-1",
	})
	if got.IsError {
		t.Fatalf("writing a fact down was refused: %s", text(t, got))
	}

	facts, err := knowledge.NewStore(s.Root()).Load(r.Path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("the repository holds %d facts, want the one just written", len(facts))
	}

	// From the record and not from a person: this door is an agent saying
	// what it ran into, and the screen that lists facts says so.
	if facts[0].Source != knowledge.FromRecord {
		t.Errorf("the fact came from %v, want the record", facts[0].Source)
	}

	// And the task it came out of travels with it. A sentence nobody can
	// trace back to the run that produced it is one the model may as well
	// have made up.
	if facts[0].Ref != "PAY-1" {
		t.Errorf("the fact says it came from %q", facts[0].Ref)
	}
}

// TestWhatAnAgentWritesStaysInItsOwnRepository.
//
// It writes straight through rather than waiting to be agreed with, so this
// is the whole of what holds it: a fact about everywhere would be in the
// prompt of every phase of every run of every project on this machine, and
// nobody agreed to that by asking an agent to fix a bug.
func TestWhatAnAgentWritesStaysInItsOwnRepository(t *testing.T) {
	s, work, r := workingOn(t, "PAY-1")
	other := gitRepo(t, work, "ledger")
	sn := Session{Root: work, Version: "test"}

	if got := sn.Call("orbit_learn", map[string]any{
		"phrase": "amounts are cents", "task_id": "PAY-1",
		// Named the way the old tool took them, and neither is read any
		// more: the scope is the task's, and a caller that could name its
		// own could name everywhere.
		"repo": other.Path, "lang": "go",
	}); got.IsError {
		t.Fatalf("refused: %s", text(t, got))
	}

	ks := knowledge.NewStore(s.Root())

	mine, err := ks.Load(r.Path)
	if err != nil {
		t.Fatal(err)
	}

	if len(mine) != 1 || mine[0].Scope.Kind != knowledge.Repo || mine[0].Scope.Repo != r.Path {
		t.Fatalf("the task's own repository holds %v", mine)
	}

	theirs, err := ks.Load(other.Path)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range theirs {
		if f.Scope.Repo == other.Path {
			t.Errorf("a fact was written about the repository the task is not worked in: %q", f.Phrase)
		}

		if f.Scope.Kind == knowledge.General || f.Scope.Kind == knowledge.Language {
			t.Errorf("a fact was written that reaches past one repository: %q", f.Phrase)
		}
	}
}

// TestAFactNeedsATaskToBeAbout, because the task is what says which
// repository it is about. Without one there is nowhere to put it, and
// nowhere in particular is everywhere.
func TestAFactNeedsATaskToBeAbout(t *testing.T) {
	_, work := newRoot(t)
	gitRepo(t, work, "payments")
	sn := Session{Root: work, Version: "test"}

	for _, named := range []string{"", "NOPE-1"} {
		got := sn.Call("orbit_learn", map[string]any{
			"phrase": "amounts are cents", "task_id": named,
		})
		if !got.IsError {
			t.Errorf("a fact about task %q was written down anyway", named)
		}
	}
}

// workingOn is a state root with one repository and one task in it.
func workingOn(t *testing.T, id string) (*store.Store, string, repo.Repo) {
	t.Helper()

	s, work := newRoot(t)
	r := gitRepo(t, work, "payments")

	addTask(t, s, r, id)

	return s, work, r
}

// TestARuleWrittenByMCPNeedsItsCheckToStop, the same honesty the supervisor's
// line keeps: a sentence cannot refuse work on its own.
func TestARuleWrittenByMCPNeedsItsCheckToStop(t *testing.T) {
	s, work, r := workingOn(t, "PAY-1")
	sn := Session{Root: work, Version: "test"}

	if got := sn.Call("orbit_learn", map[string]any{
		"phrase": "coverage stays above 90%", "task_id": "PAY-1", "stops": true,
	}); got.IsError {
		t.Fatalf("refused: %s", text(t, got))
	}

	facts, err := knowledge.NewStore(s.Root()).Load(r.Path)
	if err != nil {
		t.Fatal(err)
	}

	if len(facts) != 1 || facts[0].Action() == knowledge.Stops {
		t.Error("a rule with no check claims to stop the work")
	}
}

// TestAnAgentCanAskWhatIsKnownBeforeItPlans.
func TestAnAgentCanAskWhatIsKnownBeforeItPlans(t *testing.T) {
	_, work, r := workingOn(t, "PAY-1")
	sn := Session{Root: work, Version: "test"}

	if got := sn.Call("orbit_learn", map[string]any{
		"phrase": "reconcile marks, it does not correct", "task_id": "PAY-1",
	}); got.IsError {
		t.Fatalf("refused: %s", text(t, got))
	}

	got := sn.Call("orbit_knowledge", map[string]any{"repo": r.Path})
	if got.IsError {
		t.Fatalf("asking what is known was refused: %s", text(t, got))
	}

	if !strings.Contains(text(t, got), "reconcile marks") {
		t.Errorf("what came back does not hold the fact:\n%s", text(t, got))
	}
}

// TestAFactWithNoSentenceIsRefused, at this door as at every other.
func TestAFactWithNoSentenceIsRefused(t *testing.T) {
	_, work := newRoot(t)
	sn := Session{Root: work, Version: "test"}

	if got := sn.Call("orbit_learn", map[string]any{"phrase": "   "}); !got.IsError {
		t.Error("a fact with nothing in it was written down")
	}
}

// TestAFactSaysHowFarItReachesInTheWordsTheToolAnswersWith.
func TestAFactSaysHowFarItReachesInTheWordsTheToolAnswersWith(t *testing.T) {
	for _, c := range []struct {
		in   knowledge.Scope
		want string
	}{
		{knowledge.Scope{Kind: knowledge.General}, "everywhere"},
		{knowledge.Scope{Kind: knowledge.Language, Lang: "go"}, "in go"},
		{knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"}, "in this repository"},
		{knowledge.Scope{Kind: knowledge.Symbol, Path: "run.go", Symbol: "Start"}, "in run.go#Start"},
		{knowledge.Scope{Kind: knowledge.File, Path: "run.go"}, "in run.go"},
	} {
		if got := factWhere(c.in); got != c.want {
			t.Errorf("factWhere(%+v) = %q, want %q", c.in, got, c.want)
		}
	}
}
