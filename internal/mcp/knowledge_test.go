package mcp

// The two tools that read and write what Orbit knows.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// TestAnAgentCanWriteDownWhatItLearned.
//
// This is the source that makes Knowledge grow without anybody typing: an
// agent that hit a wall mid-task writes down what it found, and the next run
// against that code is told before it starts.
func TestAnAgentCanWriteDownWhatItLearned(t *testing.T) {
	s, work := newRoot(t)
	repo := gitRepo(t, work, "payments")
	sn := Session{Root: work, Version: "test"}

	got := sn.Call("orbit_learn", map[string]any{
		"phrase": "the fuzz tests hang when the seed is fixed",
		"repo":   repo.Path,
	})
	if got.IsError {
		t.Fatalf("writing a fact down was refused: %s", text(t, got))
	}

	facts, err := knowledge.NewStore(s.Root()).Load(repo.Path)
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
}

// TestARuleWrittenByMCPNeedsItsCheckToStop, the same honesty the supervisor's
// line keeps: a sentence cannot refuse work on its own.
func TestARuleWrittenByMCPNeedsItsCheckToStop(t *testing.T) {
	s, work := newRoot(t)
	repo := gitRepo(t, work, "payments")
	sn := Session{Root: work, Version: "test"}

	if got := sn.Call("orbit_learn", map[string]any{
		"phrase": "coverage stays above 90%", "repo": repo.Path, "stops": true,
	}); got.IsError {
		t.Fatalf("refused: %s", text(t, got))
	}

	facts, err := knowledge.NewStore(s.Root()).Load(repo.Path)
	if err != nil {
		t.Fatal(err)
	}

	if len(facts) != 1 || facts[0].Action() == knowledge.Stops {
		t.Error("a rule with no check claims to stop the work")
	}
}

// TestAnAgentCanAskWhatIsKnownBeforeItPlans.
func TestAnAgentCanAskWhatIsKnownBeforeItPlans(t *testing.T) {
	_, work := newRoot(t)
	repo := gitRepo(t, work, "payments")
	sn := Session{Root: work, Version: "test"}

	if got := sn.Call("orbit_learn", map[string]any{
		"phrase": "reconcile marks, it does not correct", "repo": repo.Path,
	}); got.IsError {
		t.Fatalf("refused: %s", text(t, got))
	}

	got := sn.Call("orbit_knowledge", map[string]any{"repo": repo.Path})
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

// TestARepositoryIsNamedByPathAndNotGuessed. A supervising model works
// across several, and a fact filed against the wrong one is a rule applied
// where nobody put it.
func TestARepositoryIsNamedByPathAndNotGuessed(t *testing.T) {
	for _, c := range []struct {
		what string
		args map[string]any
		want knowledge.Scope
	}{
		{"nothing named", map[string]any{}, knowledge.Scope{Kind: knowledge.General}},
		{"a language", map[string]any{"lang": "go"}, knowledge.Scope{Kind: knowledge.Language, Lang: "go"}},
		{"a checkout", map[string]any{"repo": "/w/orbit"}, knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"}},
	} {
		got, err := factScope(c.args)
		if err != nil {
			t.Errorf("%s: %v", c.what, err)
			continue
		}

		if got != c.want {
			t.Errorf("%s: %+v, want %+v", c.what, got, c.want)
		}
	}

	// Both at once is not a scope: a fact is about a language or about a
	// repository, and a tool that guessed which would file it somewhere
	// nobody asked for.
	if _, err := factScope(map[string]any{"lang": "go", "repo": "/w/orbit"}); err == nil {
		t.Error("a fact named both a language and a repository and was filed anyway")
	}
}
