package verb

// What one rule has been through, and the two doors that read it.
//
// `rules history` is the table and `rules review` is the argument made out
// of it. Neither was tested: both read zero, and between them they are the
// whole of what somebody sees before deciding to switch a rule off.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/words"
)

// went writes one turn of a rule's life into the store.
func went(t *testing.T, w *testWorld, turn learn.Turn) {
	t.Helper()

	if err := learn.Happened(w.store, turn); err != nil {
		t.Fatalf("write down %s: %v", turn.What, err)
	}
}

// day is a time on the seventeenth, n seconds in, so a test can say which
// turn came first without sleeping.
func day(n int) time.Time {
	return time.Date(2026, 9, 17, 9, 0, n, 0, time.UTC)
}

// TestTheHistoryOfARuleSaysWhereEachTurnHappened. One column and not two,
// because they never both apply: a gate refusing belongs to a run and to
// nobody, and a pause typed at a terminal belongs to a person and to no run.
func TestTheHistoryOfARuleSaysWhereEachTurnHappened(t *testing.T) {
	w := worldOf(t)

	went(t, w, learn.Turn{Rule: "card-number", At: day(1), What: learn.Written, By: "operator"})
	went(t, w, learn.Turn{
		Rule: "card-number", At: day(2), What: learn.Skipped, Task: "ACME-1", Phase: "test",
	})
	went(t, w, learn.Turn{Rule: "card-number", At: day(3), What: learn.Paused, By: "elio", Was: "too noisy"})

	out := mustAsk(t, w, "rules history", In{Args: map[string]string{"rule": "card-number"}})

	lines := strings.Split(out.Said, "\n")
	if len(lines) != 3 {
		t.Fatalf("the history is %d lines, want one per turn:\n%s", len(lines), out.Said)
	}

	// A turn inside a run is placed by the task and the phase.
	if !strings.Contains(lines[1], "ACME-1 · test") {
		t.Errorf("the skipped turn reads %q, want the task and the phase", lines[1])
	}

	// One taken from a terminal is placed by whoever took it, and carries
	// the reason they gave.
	if !strings.Contains(lines[2], "elio") || !strings.Contains(lines[2], "too noisy") {
		t.Errorf("the paused turn reads %q, want who paused it and why", lines[2])
	}

	// And a turn that belongs to a run with no phase behind it is placed by
	// the task alone rather than by an empty separator.
	if strings.Contains(lines[0], "·") {
		t.Errorf("a turn with no phase carries a separator: %q", lines[0])
	}
}

// TestARuleNothingHappenedToSaysSo. An empty table is a reader wondering
// whether the rule is quiet or the command is broken.
func TestARuleNothingHappenedToSaysSo(t *testing.T) {
	w := worldOf(t)

	out := mustAsk(t, w, "rules history", In{Args: map[string]string{"rule": "card-number"}})

	if !strings.Contains(out.Said, "card-number") {
		t.Errorf("it said %q, want the rule named", out.Said)
	}

	if strings.Contains(out.Said, "·") {
		t.Errorf("it drew a row for a rule with no history: %q", out.Said)
	}
}

// TestAReviewIsAnArgumentAndNotATable. What a reader is doing here is
// deciding whether to keep something, and "it refused work twice in the test
// phase and you got past it both times" is an argument where a row of two is
// a number they have to make one out of.
func TestAReviewIsAnArgumentAndNotATable(t *testing.T) {
	w := worldOf(t)
	w.facts = []knowledge.Rule{{
		ID: "card-number", Phrase: "never log a card number", Why: "it is somebody's money",
	}}

	went(t, w, learn.Turn{Rule: "card-number", At: day(1), What: learn.Written, By: "operator"})

	// The refusals are read off the record rather than out of a second
	// table: a gate that refuses work already writes gate.failed against
	// the task with the rule's name on it.
	for range 2 {
		refusedBy(t, w, "ACME-1", "card-number", "test")

		went(t, w, learn.Turn{
			Rule: "card-number", At: day(2), What: learn.Skipped, Task: "ACME-1", Phase: "test",
		})
	}

	out := mustAsk(t, w, "rules review", In{Args: map[string]string{"rule": "card-number"}})

	for _, want := range []string{"card-number", "never log a card number", "it is somebody's money"} {
		if !strings.Contains(out.Said, want) {
			t.Errorf("the review does not say %q:\n%s", want, out.Said)
		}
	}

	if !strings.Contains(out.Said, "kept it") {
		t.Errorf("the review does not say when it was kept:\n%s", out.Said)
	}

	if !strings.Contains(out.Said, "test") {
		t.Errorf("the review does not say which phase it cost something in:\n%s", out.Said)
	}
}

// TestAQuietRuleIsSaidToBeQuiet. A rule that has stopped nothing is what a
// rule that is working looks like, and a review that went silent about it
// reads as a review that failed.
func TestAQuietRuleIsSaidToBeQuiet(t *testing.T) {
	w := worldOf(t)
	w.facts = []knowledge.Rule{{ID: "card-number", Phrase: "never log a card number"}}

	went(t, w, learn.Turn{Rule: "card-number", At: day(1), What: learn.Written, By: "operator"})

	out := mustAsk(t, w, "rules review", In{Args: map[string]string{"rule": "card-number"}})

	if !strings.Contains(out.Said, "not stopped anything") {
		t.Errorf("the review of a quiet rule reads:\n%s", out.Said)
	}
}

// TestAReviewOfARuleThatIsNotThereSaysWhereTheNamesAre. "No" on its own
// leaves somebody guessing whether they mistyped it or it was forgotten.
func TestAReviewOfARuleThatIsNotThereSaysWhereTheNamesAre(t *testing.T) {
	w := worldOf(t)

	err := refuseErr(t, w, "rules review", In{Args: map[string]string{"rule": "card-number"}})
	if !strings.Contains(err.Error(), "orbit knowledge") {
		t.Errorf("the refusal is %q, want it to say where the names are", err)
	}
}

// TestASentenceIsKeptByWhenItWasSaid. The instant is the sentence's own
// name, which is what a second surface uses: the number in a listing is only
// true of the listing that printed it.
func TestASentenceIsKeptByWhenItWasSaid(t *testing.T) {
	w := worldOf(t)

	said := learn.Said{
		At: day(1), Text: "run the tests before you push", By: "operator", Topic: "testing",
	}

	if err := learn.Propose(w.store, said); err != nil {
		t.Fatalf("propose: %v", err)
	}

	at := said.At.Format(time.RFC3339Nano)

	mustAsk(t, w, "rules keep", In{Args: map[string]string{"at": at}})

	waiting, err := learn.Waiting(w.store)
	if err != nil {
		t.Fatalf("read what is waiting: %v", err)
	}

	if len(waiting) != 0 {
		t.Errorf("the sentence is still waiting: %+v", waiting)
	}

	// And the same instant twice is a refusal rather than a second answer:
	// somebody else having got there first looks exactly like this.
	if _, err := Run(ctxOf(), w, "rules keep", In{Args: map[string]string{"at": at}}); err == nil {
		t.Error("the same sentence was kept twice")
	}
}

// TestAnInstantThatIsNotOneSaysWhatItWanted. The argument is a moment, and
// "no" about a value the reader typed has to say what shape it should have
// been.
func TestAnInstantThatIsNotOneSaysWhatItWanted(t *testing.T) {
	w := worldOf(t)

	err := refuseErr(t, w, "rules keep", In{Args: map[string]string{"at": "yesterday"}})

	if !strings.Contains(err.Error(), "yesterday") {
		t.Errorf("the refusal is %q, want it to say back what was typed", err)
	}

	if !strings.Contains(err.Error(), "instant") {
		t.Errorf("the refusal is %q, want it to say a sentence is named by an instant", err)
	}
}

// TestTheWordsOfAReviewAreTheWindowsToo. Story is exported because the
// cockpit draws the same sentences: two surfaces telling it two ways would
// be two accounts of the same evidence, which is the thing somebody is about
// to decide on.
func TestTheWordsOfAReviewAreTheWindowsToo(t *testing.T) {
	f := knowledge.Rule{ID: "card-number", Phrase: "never log a card number"}

	turns := []learn.Turn{
		{Rule: f.ID, At: day(1), What: learn.Written},
		{Rule: f.ID, At: day(2), What: learn.Failed, Task: "ACME-1", Phase: "test"},
		{Rule: f.ID, At: day(3), What: learn.Paused, Was: "too noisy"},
	}

	told := Story(words.For("en"), f, turns)
	if len(told) < 3 {
		t.Fatalf("the story is %v, want the keeping, the friction and the pause", told)
	}

	if !strings.Contains(told[len(told)-1], "too noisy") {
		t.Errorf("the story ends %q, want the reason it was paused", told[len(told)-1])
	}
}
