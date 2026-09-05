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
