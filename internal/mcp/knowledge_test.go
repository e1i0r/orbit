package mcp

// The two tools an agent reads what Orbit knows with, and offers it more.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// TestWhatAnAgentFindsWaitsInTheTray.
//
// This is the source that makes what Orbit knows grow without anybody
// typing: an agent that hit a wall mid-task offers what it found. It is an
// offer and not a fact — a fact is in the prompt of every phase of every run
// against that code, and nothing gets there without somebody having read it.
func TestWhatAnAgentFindsWaitsInTheTray(t *testing.T) {
	s, work, r := workingOn(t, "PAY-1")
	sn := Session{Root: work, Version: "test"}

	got := sn.Call("orbit_learn", map[string]any{
		"phrase": "the fuzz tests hang when the seed is fixed", "task_id": "PAY-1",
	})
	if got.IsError {
		t.Fatalf("offering a rule was refused: %s", text(t, got))
	}

	facts, err := knowledge.NewStore(s.Root()).Load(r.Path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(facts) != 0 {
		t.Fatalf("the repository holds %d facts, and nobody has agreed to any", len(facts))
	}

	waiting, err := learn.Waiting(s)
	if err != nil {
		t.Fatalf("read the tray: %v", err)
	}

	if len(waiting) != 1 || !strings.Contains(waiting[0].Text, "fuzz tests hang") {
		t.Fatalf("the tray holds %v", waiting)
	}

	one := waiting[0]

	// It says a model found it, and which task it came out of. Both are
	// what a reader needs before agreeing: a sentence nobody can trace back
	// to the run that produced it is one the model may as well have made up.
	if one.By != learn.AModel || one.About != "PAY-1" {
		t.Errorf("the row says it came from %q, about %q", one.By, one.About)
	}

	if one.Repo != r.Path {
		t.Errorf("the row is about the checkout %q", one.Repo)
	}

	// And the answer says what did not happen. An agent told "written down"
	// carries on believing the next run is already warned.
	if said := text(t, got); !strings.Contains(said, "waiting") {
		t.Errorf("the agent was told %q, which does not say it is not in effect", said)
	}
}

// TestWhatAnAgentOffersStaysInItsOwnRepository.
//
// A rule about everywhere would be in the prompt of every phase of every run
// of every project on this machine, and nobody agreed to that by asking an
// agent to fix a bug. The tray is a person's queue, not a wider door.
func TestWhatAnAgentOffersStaysInItsOwnRepository(t *testing.T) {
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

	waiting, err := learn.Waiting(s)
	if err != nil {
		t.Fatal(err)
	}

	if len(waiting) != 1 || waiting[0].Repo != r.Path {
		t.Fatalf("the tray holds %v, and the task is worked in %q", waiting, r.Path)
	}

	if waiting[0].Repo == other.Path {
		t.Error("a rule was offered about the repository the task is not worked in")
	}
}

// TestAnAgentCannotAskForAGate.
//
// A rule with a check is a gate: every phase of every future run in that
// repository executes that command and is sent back when it fails. A check
// that is wrong, or slow, is an hour of a task spent on something nobody
// agreed to — so the command belongs to whoever keeps the rule, and the
// tool has nothing to ask for it with.
func TestAnAgentCannotAskForAGate(t *testing.T) {
	s, work, r := workingOn(t, "PAY-1")
	sn := Session{Root: work, Version: "test"}

	if got := sn.Call("orbit_learn", map[string]any{
		"phrase": "coverage stays above 90%", "task_id": "PAY-1",
		"stops": true, "check": "make coverage",
	}); got.IsError {
		t.Fatalf("refused: %s", text(t, got))
	}

	// Kept without one, the way a person keeps it when they are not asked
	// for a command either.
	waiting, err := learn.Waiting(s)
	if err != nil {
		t.Fatal(err)
	}

	if len(waiting) != 1 {
		t.Fatalf("the tray holds %d rows", len(waiting))
	}

	if err := learn.Keep(s, waiting[0].At, waiting[0].Text, "", learn.Place{}); err != nil {
		t.Fatalf("keep: %v", err)
	}

	kept := onlyFact(t, s, r.Path)
	if kept.Action() == knowledge.Stops {
		t.Error("a rule the agent asked to stop work stops work")
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
			t.Errorf("a rule about task %q was offered anyway", named)
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

// onlyFact is the single fact the checkout holds, and a failure when it
// holds any other number of them.
func onlyFact(t *testing.T, s *store.Store, repoPath string) knowledge.Fact {
	t.Helper()

	facts, err := knowledge.NewStore(s.Root()).Load(repoPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("what Orbit knows holds %d facts, want one", len(facts))
	}

	return facts[0]
}

// TestAnAgentCanAskWhatIsKnownBeforeItPlans, which is the other half of one
// flow: what it offers waits, and what somebody already agreed to is there
// to be read.
func TestAnAgentCanAskWhatIsKnownBeforeItPlans(t *testing.T) {
	s, work, r := workingOn(t, "PAY-1")
	sn := Session{Root: work, Version: "test"}

	at := time.Now().UTC()
	if err := learn.Propose(s, learn.Said{
		At: at, Text: "reconcile marks, it does not correct", By: learn.AModel,
		About: "PAY-1", Repo: r.Path,
	}); err != nil {
		t.Fatalf("propose: %v", err)
	}

	if err := learn.Keep(s, at, "reconcile marks, it does not correct", "",
		learn.Place{Repo: r.Path}); err != nil {
		t.Fatalf("keep: %v", err)
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
		t.Error("a rule with nothing in it was offered")
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
