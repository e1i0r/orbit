package ui

// Three readings that disagreed with what was on the screen beside them.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/verb"
)

// TestTheMapIsReadAgainWhenItDisagreesWithTheDiff.
//
// The tree is read once, when the task's screen opens, and a task still
// running had written nothing then. It said "this task has changed nothing
// in this checkout" for the rest of the run, beside a diff pane listing
// four files.
func TestTheMapIsReadAgainWhenItDisagreesWithTheDiff(t *testing.T) {
	m := openOn(t, "PAY-1")

	// Read, and it saw nothing changed.
	m.shape.known, m.shape.tree = true, verb.Cell{}

	// No diff either: the two agree, and nothing is asked again.
	m.diff = ""
	if m.staleTree() {
		t.Error("an unchanged tree beside an empty diff reads as stale")
	}

	// A diff arrives. Now they disagree.
	m.diff = "diff --git a/internal/task/run.go b/internal/task/run.go\n+one line\n"
	if !m.staleTree() {
		t.Fatal("a tree saying nothing changed beside a diff of one file is not stale")
	}

	// Forgotten, so the next sync asks again — and marked, so a checkout
	// that genuinely reads as unchanged is not asked about for ever.
	next := m.forgetTreeReading()
	if next.shape.known || next.shape.asking {
		t.Error("the reading was not dropped")
	}

	if !next.shape.reread {
		t.Error("the re-read was not remembered, so every diff will ask again")
	}

	if next.staleTree() {
		t.Error("a reading already taken again still reads as stale")
	}
}

// TestARuleThatStopsTheWorkIsACheck.
//
// A run's gates are its phase's own and then the standing rules. This
// counted only the first kind, so `careful` — which declares no checks —
// said "there is nothing to run on either side" while `make check` ran and
// passed at three gates.
func TestARuleThatStopsTheWorkIsACheck(t *testing.T) {
	m := openOn(t, "PAY-1")

	// No port: nothing, and no panic.
	m.opts.Knows = nil
	if got := m.ruleChecks(); len(got) != 0 {
		t.Errorf("a window with no knowledge port offered %d checks", len(got))
	}

	m.opts.Knows = func() []knowledge.Rule {
		return []knowledge.Rule{
			{Phrase: "make check has to be green", Check: "make check", Stops: true},
			// A rule with nothing to run is not a check.
			{Phrase: "prefer table tests", Stops: true},
		}
	}

	got := m.ruleChecks()
	if len(got) != 1 {
		t.Fatalf("ruleChecks offered %d, want only the one that can be run: %+v", len(got), got)
	}

	if got[0].Command != "make check" {
		t.Errorf("the check runs %q, want make check", got[0].Command)
	}

	if !strings.Contains(got[0].Name, "make check has to be green") {
		t.Errorf("the check is named %q, want the rule's own sentence", got[0].Name)
	}
}
