package learn

// What a rule did, read off the record rather than out of a second table.
//
// A gate that refuses work already writes gate.failed against the task with
// the rule's name on it. Copying those into a table of their own would be a
// write on every failing gate of every run, kept in two places, and wrong in
// one of them the first time something went half way — so the history is
// folded from both sources on the way out, and this is the half that reads
// the record.

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// refusedIn writes a gate refusal into one task's record, the way a run does
// when a rule stops the work.
func refusedIn(t *testing.T, s *store.Store, id, rule, phase string, at time.Time) {
	t.Helper()

	d, err := s.Record()
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	e := record.Event{
		Kind: record.GateFailed, At: at, Phase: phase,
		Data: map[string]string{"rule": rule, "gate": "make check"},
	}

	if err := d.Append(id, e); err != nil {
		t.Fatalf("write the refusal: %v", err)
	}
}

// aRoot is a state root of the test's own.
func aRoot(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() }) //nolint:errcheck // the test is over

	return s
}

// TestARuleRefusingWorkIsReadOffEveryTaskItHappenedIn.
//
// What makes a rule worth reconsidering is the pattern, and a pattern is
// spread over tasks: "it stops me in the test phase" is a reading of four
// runs, not of one, so the walk is over every task the record holds.
func TestARuleRefusingWorkIsReadOffEveryTaskItHappenedIn(t *testing.T) {
	s := aRoot(t)

	at := time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC)

	refusedIn(t, s, "ACME-1", "card-number", "test", at)
	refusedIn(t, s, "PAY-7", "card-number", "test", at.Add(time.Minute))
	refusedIn(t, s, "PAY-7", "other-rule", "review", at.Add(2*time.Minute))

	turns, err := History(s, "card-number")
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	if len(turns) != 2 {
		t.Fatalf("the rule refused work %d times, want the two it did: %+v", len(turns), turns)
	}

	seen := map[string]bool{}

	for _, one := range turns {
		if one.What != Failed {
			t.Errorf("a refusal was read as %q", one.What)
		}

		if one.By != "" {
			t.Errorf("a refusal names %q as having done it, and the gate ran on its own", one.By)
		}

		if one.Phase != "test" {
			t.Errorf("it happened in %q, want the phase the record said", one.Phase)
		}

		seen[one.Task] = true
	}

	for _, id := range []string{"ACME-1", "PAY-7"} {
		if !seen[id] {
			t.Errorf("the refusal in %s was not read", id)
		}
	}
}

// TestARefusalOverAnotherRuleIsNotThisRulesHistory. The rule's name is on
// the event, and a fold that ignored it would give every rule every gate
// failure on the machine.
func TestARefusalOverAnotherRuleIsNotThisRulesHistory(t *testing.T) {
	s := aRoot(t)

	at := time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC)
	refusedIn(t, s, "ACME-1", "other-rule", "test", at)

	turns, err := History(s, "card-number")
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	if len(turns) != 0 {
		t.Errorf("another rule's refusals are in this one's history: %+v", turns)
	}
}

// TestAGateThatFailedOverNoRuleAtAllIsNobodysHistory. A flow declares gates
// of its own, and one of those refusing work says nothing about any rule.
func TestAGateThatFailedOverNoRuleAtAllIsNobodysHistory(t *testing.T) {
	s := aRoot(t)

	d, err := s.Record()
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	e := record.Event{
		Kind: record.GateFailed, At: time.Now().UTC(), Phase: "test",
		Data: map[string]string{"gate": "make check"},
	}

	if err := d.Append("ACME-1", e); err != nil {
		t.Fatalf("write the refusal: %v", err)
	}

	turns, err := History(s, "card-number")
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	if len(turns) != 0 {
		t.Errorf("a gate with no rule behind it is in a rule's history: %+v", turns)
	}
}

// TestTheTwoSourcesAreOneStoryInTheOrderTheyHappened. What somebody did to
// the rule is written down when they do it; what the rule did is already in
// the record. Read as two lists, a reader would have to merge them by eye.
func TestTheTwoSourcesAreOneStoryInTheOrderTheyHappened(t *testing.T) {
	s := aRoot(t)

	at := time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC)

	if err := Happened(s, Turn{Rule: "card-number", At: at, What: Written, By: Operator}); err != nil {
		t.Fatalf("write it down: %v", err)
	}

	refusedIn(t, s, "ACME-1", "card-number", "test", at.Add(time.Minute))

	paused := Turn{Rule: "card-number", At: at.Add(2 * time.Minute), What: Paused, By: Operator}
	if err := Happened(s, paused); err != nil {
		t.Fatalf("pause it: %v", err)
	}

	turns, err := History(s, "card-number")
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	if len(turns) != 3 {
		t.Fatalf("the story is %d turns, want the three that happened: %+v", len(turns), turns)
	}

	want := []string{Written, Failed, Paused}
	for i, one := range turns {
		if one.What != want[i] {
			t.Errorf("turn %d is %q, want %q", i, one.What, want[i])
		}
	}
}

// TestARuleWithNoNameHasNoHistoryByConstruction. Names are coined when a
// rule is written, so one without a name is one somebody wrote by hand and
// Orbit has not written since — and walking every task to find nothing would
// be reading the whole record to answer a question with no subject.
func TestARuleWithNoNameHasNoHistoryByConstruction(t *testing.T) {
	s := aRoot(t)

	refusedIn(t, s, "ACME-1", "", "test", time.Now().UTC())

	turns, err := History(s, "")
	if err != nil {
		t.Fatalf("read the history of no rule: %v", err)
	}

	if len(turns) != 0 {
		t.Errorf("a rule with no name has the history %+v", turns)
	}
}
