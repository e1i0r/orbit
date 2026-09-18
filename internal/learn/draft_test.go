package learn

// The one place a model decides anything: reading what somebody keeps
// saying and writing it down as a rule.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/db"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// answering is an Ask that says the same thing every time, and remembers
// what it was asked.
func answering(answer string, asked *string) Ask {
	return func(_ context.Context, question string) (string, error) {
		if asked != nil {
			*asked = question
		}

		return answer, nil
	}
}

// aHabit is one habit, said three times, written into a real record so that
// reading it back is what the test is about.
func aHabit(t *testing.T) (*store.Store, Habit) {
	t.Helper()

	s := root(t)

	d, err := s.Record()
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	for _, e := range []record.Event{
		{Kind: record.TaskCreated, At: when(0), Data: map[string]string{"path": "/w/acme"}},
		{Kind: record.PhaseStarted, At: when(1), Phase: "test"},
		toldTo(2, Operator, "add fuzz testing to this"),
		toldTo(3, Operator, "put some fuzz tests on the parser"),
		toldTo(4, Operator, "this needs fuzz tests too"),
	} {
		if err := d.Append("ACME-1", e); err != nil {
			t.Fatalf("append %s: %v", e.Kind, err)
		}
	}

	got, err := Repeated(s)
	if err != nil {
		t.Fatalf("read what was repeated: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("the fixture reads back as %d habits", len(got))
	}

	return s, got[0]
}

// TestTheModelWritesTheRuleAndItWaitsLikeAnyOther.
//
// It is written by a model and it is still only an offer: every fact reaches
// every phase's prompt, and one nobody agreed to would be a rule a person
// never saw put there.
func TestTheModelWritesTheRuleAndItWaitsLikeAnyOther(t *testing.T) {
	s, h := aHabit(t)

	got, err := drafted(context.Background(), s,
		answering("testing | anything that parses input gets fuzz tests", nil), h)
	if err != nil {
		t.Fatalf("drafting: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("the model's answer became %d rules", len(got))
	}

	waiting, err := Waiting(s)
	if err != nil {
		t.Fatal(err)
	}

	if len(waiting) != 1 {
		t.Fatalf("the tray holds %d rows", len(waiting))
	}

	one := waiting[0]
	if one.By != FromAHabit || one.Topic != "testing" {
		t.Errorf("the row says it came from %q, about %q", one.By, one.Topic)
	}

	if one.Habit != h.Handle() {
		t.Errorf("the row was drawn from %q, and the habit is %q", one.Habit, h.Handle())
	}

	if !strings.Contains(one.Text, "fuzz") {
		t.Errorf("the rule reads %q", one.Text)
	}
}

// TestTheModelIsShownTheSentencesAndNothingElse. It is a classification and
// not an analysis, and that is the whole of what keeps it cheap.
func TestTheModelIsShownTheSentencesAndNothingElse(t *testing.T) {
	_, h := aHabit(t)

	asked := question(h)

	for _, want := range []string{"add fuzz testing to this", "this needs fuzz tests too"} {
		if !strings.Contains(asked, want) {
			t.Errorf("the model was not shown %q", want)
		}
	}

	// And the list it must choose from, so that two readings of the same
	// person add up rather than inventing a third name for one thing.
	for _, topic := range topics {
		if !strings.Contains(asked, topic) {
			t.Errorf("the model was not given the topic %q to choose from", topic)
		}
	}
}

// TestARuleAlreadyAnsweredIsNotOfferedAgain, whatever the answer was.
// Dropping one is an answer, and asking again is asking something settled.
func TestARuleAlreadyAnsweredIsNotOfferedAgain(t *testing.T) {
	s, h := aHabit(t)

	first, err := Draft(context.Background(), s,
		answering("testing | anything that parses input gets fuzz tests", nil))
	if err != nil {
		t.Fatalf("drafting: %v", err)
	}

	if len(first) != 1 {
		t.Fatalf("the first reading offered %d rules", len(first))
	}

	// Dropped, which is the answer that makes this worth checking: a kept
	// rule is out of the tray anyway, and a dropped one is still a row.
	if err := Drop(s, first[0].At); err != nil {
		t.Fatalf("drop: %v", err)
	}

	again, err := Draft(context.Background(), s,
		answering("testing | anything that parses input gets fuzz tests", nil))
	if err != nil {
		t.Fatalf("drafting again: %v", err)
	}

	if len(again) != 0 {
		t.Errorf("a rule that was dropped came back: %+v", again)
	}

	if h.Handle() == "" {
		t.Error("the habit has no name, so nothing could have been remembered about it")
	}
}

// TestNothingIsAnAnswer. A habit that does not amount to a rule is the
// commonest case there is, and a model made to answer something would answer
// something.
func TestNothingIsAnAnswer(t *testing.T) {
	s, _ := aHabit(t)

	got, err := Draft(context.Background(), s, answering("", nil))
	if err != nil {
		t.Fatalf("drafting: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("an empty answer became %d rules", len(got))
	}
}

// TestWithNoEngineNothingElseBreaks. This is a path that needs a model, and
// the rest of Orbit does not: a machine with no engine installed has to be
// told so plainly rather than shown a crash.
func TestWithNoEngineNothingElseBreaks(t *testing.T) {
	s, _ := aHabit(t)

	if _, err := Draft(context.Background(), s, nil); err == nil {
		t.Error("drafting with no engine answered as though it had one")
	}

	broken := func(_ context.Context, _ string) (string, error) {
		return "", errors.New("no engine on this machine")
	}

	if _, err := Draft(context.Background(), s, broken); err == nil {
		t.Error("an engine that would not run answered as though it had")
	}
}

// TestAHabitIsKnownByTheSentenceItStartedWith, which is the one thing about
// it that does not move: the record only grows forward, so it picks up newer
// sentences and never an older one. The words it shares do move — they
// narrow every time somebody says the thing again.
func TestAHabitIsKnownByTheSentenceItStartedWith(t *testing.T) {
	_, h := aHabit(t)

	if h.Handle() != h.Said[0].At.UTC().Format(time.RFC3339Nano) {
		t.Errorf("the habit is known by %q", h.Handle())
	}

	grown := h
	grown.Words = []string{"fuzz"}
	grown.Said = append(grown.Said, Directive{At: when(9), Text: "more fuzz tests"})

	if grown.Handle() != h.Handle() {
		t.Errorf("the habit grew and changed its name from %q to %q", h.Handle(), grown.Handle())
	}

	if (Habit{}).Handle() != "" {
		t.Error("a habit made of nothing has a name")
	}
}

// TestAnUnaskedHabitIsUnanswered, so that the first reading of a record
// offers something rather than nothing.
func TestAnUnaskedHabitIsUnanswered(t *testing.T) {
	s := root(t)

	d, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}

	answered, err := d.Answered("")
	if err != nil || answered {
		t.Errorf("a habit with no name reads as answered: %v, %v", answered, err)
	}

	if err := d.Propose(db.Proposal{
		SaidAt: when(1), Said: "always run the tests", Habit: "a-habit",
	}); err != nil {
		t.Fatal(err)
	}

	if answered, err := d.Answered("a-habit"); err != nil || !answered {
		t.Errorf("a habit already asked about reads as %v, %v", answered, err)
	}
}

// sixHabits is one task in which six different things were each said often
// enough to be a habit, one in each phase the work went through.
func sixHabits(t *testing.T) *store.Store {
	t.Helper()

	s := root(t)

	d, err := s.Record()
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	events := []record.Event{
		{Kind: record.TaskCreated, At: when(0), Data: map[string]string{"path": "/w/acme"}},
	}

	n := 1

	for _, phase := range []string{"plan", "implement", "test", "review", "document", "land"} {
		events = append(events, record.Event{Kind: record.PhaseStarted, At: when(n), Phase: phase})
		n++

		for range enoughTimes {
			events = append(events, toldTo(n, Operator, "add fuzz testing to the "+phase))
			n++
		}
	}

	for _, e := range events {
		if err := d.Append("ACME-1", e); err != nil {
			t.Fatalf("append %s: %v", e.Kind, err)
		}
	}

	return s
}

// TestOnlySoManyHabitsAreDraftedInOneGo.
//
// Every habit drafted is a model call paid for, and this runs from a key. A
// record with two months in it holds more habits than anybody wants to pay
// to read at once, and the ones left over are still there the next time.
func TestOnlySoManyHabitsAreDraftedInOneGo(t *testing.T) {
	s := sixHabits(t)

	there, err := Repeated(s)
	if err != nil {
		t.Fatalf("read what was repeated: %v", err)
	}

	if len(there) != atOnce+1 {
		t.Fatalf("the fixture reads back as %d habits, want one more than %d", len(there), atOnce)
	}

	asked := 0

	got, err := Draft(context.Background(), s, func(_ context.Context, _ string) (string, error) {
		asked++

		return "testing | anything that parses input gets fuzz tests", nil
	})
	if err != nil {
		t.Fatalf("drafting: %v", err)
	}

	if asked != atOnce {
		t.Errorf("six habits cost %d calls to a model, want %d", asked, atOnce)
	}

	if len(got) != atOnce {
		t.Errorf("six habits became %d rules, want %d", len(got), atOnce)
	}
}
