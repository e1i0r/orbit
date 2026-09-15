package verb

// Getting past a rule that is in the way.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/record"
)

// refusedBy puts a gate refusal in a task's record, the way a run does when a
// rule stops the work.
func refusedBy(t *testing.T, w *testWorld, id, rule, phase string) {
	t.Helper()

	d, err := w.store.Record()
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	e := record.Event{Kind: record.GateFailed, Phase: phase, Text: "coverage 74%"}
	if rule != "" {
		e.Data = map[string]string{"rule": rule, "gate": "coverage stays above 90%"}
	}

	if err := d.Append(id, e); err != nil {
		t.Fatalf("append the refusal: %v", err)
	}
}

// TestSkippingAGateIsSomethingSaidAboutTheRuleBehindIt.
//
// The first time, without counting to three: nobody skips a rule they agree
// with, and if you skipped it you have already said something without saying
// it.
func TestSkippingAGateIsSomethingSaidAboutTheRuleBehindIt(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")
	w.wrote(t, "ACME-60", r.Path, "pay the thing")

	rule := aRule("aaaa1111", "coverage stays above 90%")
	w.facts = []knowledge.Fact{rule}

	refusedBy(t, w, "ACME-60", "aaaa1111", "test")

	mustAsk(t, w, "task skip", In{Task: "ACME-60", Repo: r.Path, By: "operator"})

	turns, err := learn.History(w.store, "aaaa1111")
	if err != nil {
		t.Fatalf("read what happened to it: %v", err)
	}

	// Two turns and one story: the gate refusing is already in the record,
	// and the skip is written down beside it.
	if len(turns) != 2 || turns[0].What != learn.Failed || turns[1].What != learn.Skipped {
		t.Fatalf("skipping past it reads as %+v", turns)
	}

	// Where it happened, because the pattern is the point: "I always skip
	// this in the test phase" is one, and "I skipped it once" is not.
	if turns[1].Task != "ACME-60" || turns[1].Phase != "test" {
		t.Errorf("it says it happened at %q, %q", turns[1].Task, turns[1].Phase)
	}

	// And the rule goes to be looked at, still applying: skipping is not
	// disagreeing, so the next run is told it all the same.
	now := w.facts[0]
	if !now.Review {
		t.Error("a rule somebody skipped is not waiting for a decision")
	}

	if !now.Tells() {
		t.Error("skipping a rule stopped it applying")
	}
}

// TestSkippingSomethingThatIsNotARuleSaysNothingAboutAnyRule.
//
// A phase stopped by a gate the flow declares, or by one marked to wait, has
// no rule behind it — and writing a turn against whichever rule happened to
// be named last in the record would be inventing an opinion nobody had.
func TestSkippingSomethingThatIsNotARuleSaysNothingAboutAnyRule(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")
	w.wrote(t, "ACME-61", r.Path, "pay the thing")

	w.facts = []knowledge.Fact{aRule("aaaa1111", "coverage stays above 90%")}

	refusedBy(t, w, "ACME-61", "", "test")

	mustAsk(t, w, "task skip", In{Task: "ACME-61", Repo: r.Path, By: "operator"})

	turns, err := learn.History(w.store, "aaaa1111")
	if err != nil {
		t.Fatal(err)
	}

	if len(turns) != 0 {
		t.Errorf("skipping a gate no rule put there wrote %+v", turns)
	}

	if w.facts[0].Review {
		t.Error("a rule nobody skipped is waiting for a decision")
	}
}

// TestSkippingARuleAlreadyWaitingDoesNotAskTwice, so that the second skip
// adds to what happened without putting the same question back on the list.
func TestSkippingARuleAlreadyWaitingDoesNotAskTwice(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")
	w.wrote(t, "ACME-62", r.Path, "pay the thing")

	was := aRule("aaaa1111", "coverage stays above 90%")
	was.Review = true
	w.facts = []knowledge.Fact{was}

	refusedBy(t, w, "ACME-62", "aaaa1111", "test")

	mustAsk(t, w, "task skip", In{Task: "ACME-62", Repo: r.Path, By: "operator"})

	// The skip is still written down — twice skipped is twice, and that is
	// the number somebody reads later — beside the refusal that caused it.
	turns, err := learn.History(w.store, "aaaa1111")
	if err != nil {
		t.Fatal(err)
	}

	if len(turns) != 2 {
		t.Errorf("the second skip reads as %+v", turns)
	}
}
