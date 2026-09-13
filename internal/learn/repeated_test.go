package learn

// What somebody keeps telling runs, read back out of the record.

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// aTask writes one task down and returns the events it starts with.
func aTask(repo string) []record.Event {
	return []record.Event{
		{Kind: record.TaskCreated, At: when(0), Data: map[string]string{"path": repo}},
		{Kind: record.PhaseStarted, At: when(1), Phase: "test"},
	}
}

// when is a moment, far enough apart that no two share one.
func when(n int) time.Time {
	return time.Date(2026, 9, 11, 9, n, 0, 0, time.UTC)
}

// toldTo is one directive, from whoever said it.
func toldTo(n int, by, text string) record.Event {
	return record.Event{
		Kind: record.TaskDialogue, At: when(n), Text: text,
		Data: map[string]string{"by": by},
	}
}

// TestTheSameInstructionGivenThreeTimesIsAHabit.
//
// This is the whole of the third source: what nobody would think to write
// down, because they do not know they do it. "add fuzz testing" read on its
// own is a correction to one task, and the shape test throws it away rightly
// — what makes it a rule is that it keeps happening.
func TestTheSameInstructionGivenThreeTimesIsAHabit(t *testing.T) {
	events := aTask("/w/acme")
	events = append(events,
		toldTo(2, Operator, "add fuzz testing to this"),
		toldTo(3, Operator, "put some fuzz tests on the parser"),
		toldTo(4, Operator, "this needs fuzz tests too"),
	)

	got := habits(toldIn("ACME-1", events))
	if len(got) != 1 {
		t.Fatalf("what was said three times came back as %d habits: %+v", len(got), got)
	}

	one := got[0]
	if one.Times() != 3 {
		t.Errorf("it was said %d times", one.Times())
	}

	if one.Repo != "/w/acme" || one.Phase != "test" {
		t.Errorf("the habit is filed under %q, %q", one.Repo, one.Phase)
	}

	// The words they share are the handle: it is what a reader reads first,
	// and what something writing the rule would be given with the
	// sentences.
	if len(one.Words) < enoughWords {
		t.Errorf("the habit is held together by %v", one.Words)
	}
}

// TestSaidTwiceIsStillACoincidence. Twice happens often enough that counting
// it would fill this with noise, and noise here is a person scrolling past
// the one thing that mattered.
func TestSaidTwiceIsStillACoincidence(t *testing.T) {
	events := aTask("/w/acme")
	events = append(events,
		toldTo(2, Operator, "add fuzz testing to this"),
		toldTo(3, Operator, "put some fuzz tests on the parser"),
	)

	if got := habits(toldIn("ACME-1", events)); len(got) != 0 {
		t.Errorf("something said twice came back as a habit: %+v", got)
	}
}

// TestTwoDifferentInstructionsAreNotOne, however often each is said. The
// whole of what groups them is the words they have in common, and sentences
// with nothing in common have nothing in common.
func TestTwoDifferentInstructionsAreNotOne(t *testing.T) {
	events := aTask("/w/acme")
	events = append(events,
		toldTo(2, Operator, "add fuzz testing to the parser"),
		toldTo(3, Operator, "rename that variable, it reads badly"),
		toldTo(4, Operator, "the commit message wants rewriting"),
	)

	if got := habits(toldIn("ACME-1", events)); len(got) != 0 {
		t.Errorf("three unrelated corrections came back as %+v", got)
	}
}

// TestTheSameWordsInTwoPhasesAreTwoRules.
//
// "more coverage" in the plan phase is somebody asking for a plan that
// covers more; in the test phase it is somebody asking for tests. Grouped
// together they would become one rule that is wrong in both places.
func TestTheSameWordsInTwoPhasesAreTwoRules(t *testing.T) {
	events := []record.Event{
		{Kind: record.TaskCreated, At: when(0), Data: map[string]string{"path": "/w/acme"}},
		{Kind: record.PhaseStarted, At: when(1), Phase: "plan"},
		toldTo(2, Operator, "more coverage than that, please"),
		toldTo(3, Operator, "much more coverage in this one"),
		toldTo(4, Operator, "coverage, more of it"),
		{Kind: record.PhaseStarted, At: when(5), Phase: "test"},
		toldTo(6, Operator, "more coverage than that, please"),
		toldTo(7, Operator, "much more coverage in this one"),
		toldTo(8, Operator, "coverage, more of it"),
	}

	got := habits(toldIn("ACME-1", events))
	if len(got) != 2 {
		t.Fatalf("the same words in two phases came back as %d habits", len(got))
	}

	if got[0].Phase == got[1].Phase {
		t.Errorf("both habits are filed under %q", got[0].Phase)
	}
}

// TestASentenceAlreadyNoticedAsARuleDoesNotComeBackHere, because it is
// already in the tray waiting for an answer. Counted here as well, it would
// be asked about twice.
func TestASentenceAlreadyNoticedAsARuleDoesNotComeBackHere(t *testing.T) {
	events := aTask("/w/acme")
	events = append(events,
		toldTo(2, Operator, "always add fuzz testing to the parser"),
		toldTo(3, Operator, "always add fuzz testing to the reader"),
		toldTo(4, Operator, "always add fuzz testing to the writer"),
	)

	if got := habits(toldIn("ACME-1", events)); len(got) != 0 {
		t.Errorf("sentences already in the tray came back as a habit: %+v", got)
	}
}

// TestWhatAModelRepeatsIsNotAHabitOfYours. This is about what a person does
// without noticing, and a model saying the same thing three times is a model
// saying the same thing three times.
func TestWhatAModelRepeatsIsNotAHabitOfYours(t *testing.T) {
	events := aTask("/w/acme")
	events = append(events,
		toldTo(2, "mcp", "add fuzz testing to this"),
		toldTo(3, "mcp", "put some fuzz tests on the parser"),
		toldTo(4, "supervisor", "this needs fuzz tests too"),
	)

	if got := habits(toldIn("ACME-1", events)); len(got) != 0 {
		t.Errorf("what Orbit and a model said came back as a habit of yours: %+v", got)
	}
}

// TestADirectiveWithNobodysNameOnItCountsAsAPerson, which is the way round
// that fails safe: a row somebody looks at, rather than one quietly dropped.
func TestADirectiveWithNobodysNameOnItCountsAsAPerson(t *testing.T) {
	events := aTask("/w/acme")
	for n, text := range []string{
		"add fuzz testing to this",
		"put some fuzz tests on the parser",
		"this needs fuzz tests too",
	} {
		events = append(events, record.Event{
			Kind: record.TaskDialogue, At: when(n + 2), Text: text,
		})
	}

	if got := habits(toldIn("ACME-1", events)); len(got) != 1 {
		t.Errorf("directives with no name on them came back as %d habits", len(got))
	}
}

// TestReadingTheRecordWithNothingInItSaysSo, rather than failing: a
// workspace where nobody has told a run anything is the first day of using
// Orbit, not an error.
func TestReadingTheRecordWithNothingInItSaysSo(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() })

	got, err := Repeated(s)
	if err != nil {
		t.Fatalf("reading an empty record: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("an empty record holds %d habits", len(got))
	}
}
