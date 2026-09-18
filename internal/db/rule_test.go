package db

// What happened to a rule.

import (
	"strings"
	"testing"
	"time"
)

// TestATurnComesBackWithWhereItHappened. A turn taken inside a run carries
// the task and the phase, because what makes a rule worth reconsidering is
// the pattern — "I always skip this in the test phase" is a pattern where "I
// skipped it once" is not.
func TestATurnComesBackWithWhereItHappened(t *testing.T) {
	d := open(t)

	turn := RuleTurn{
		Rule:  "R-1",
		At:    time.Date(2026, 9, 17, 9, 0, 1, 0, time.UTC),
		What:  RuleSkipped,
		By:    "operator",
		Was:   "make check",
		Task:  "ACME-1",
		Phase: "test",
	}

	if err := d.Happened(turn); err != nil {
		t.Fatalf("write down the turn: %v", err)
	}

	history, err := d.RuleHistory("R-1")
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("the rule has %d turns, want one", len(history))
	}

	got := history[0]
	if !got.At.Equal(turn.At) {
		t.Errorf("the turn is stamped %s, want %s", got.At, turn.At)
	}

	got.At = turn.At

	if got != turn {
		t.Errorf("the turn came back as %+v, want %+v", got, turn)
	}
}

// TestARuleStoryIsInTheOrderItWasWritten. Ordered by the row's own id and
// never by the timestamp: two processes' clocks have no order between them,
// and a story that reorders itself is not a story.
func TestARuleStoryIsInTheOrderItWasWritten(t *testing.T) {
	d := open(t)

	// The stamps run backwards on purpose. A reader ordering by them would
	// answer forgotten, reworded, kept — the rule's life in reverse.
	at := func(second int) time.Time {
		return time.Date(2026, 9, 17, 9, 0, second, 0, time.UTC)
	}

	turns := []RuleTurn{
		{Rule: "R-1", At: at(3), What: RuleKept},
		{Rule: "R-1", At: at(2), What: RuleReworded, Was: "the old sentence"},
		{Rule: "R-1", At: at(1), What: RuleForgotten},
	}

	for _, turn := range turns {
		if err := d.Happened(turn); err != nil {
			t.Fatalf("write down %s: %v", turn.What, err)
		}
	}

	history, err := d.RuleHistory("R-1")
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	var story []string
	for _, turn := range history {
		story = append(story, turn.What)
	}

	want := RuleKept + "," + RuleReworded + "," + RuleForgotten
	if strings.Join(story, ",") != want {
		t.Errorf("the story reads %v, want %s", story, want)
	}
}

// TestOneRuleStoryIsNotAnother. The name is what a story is read by, and two
// rules living in one table must not answer each other's history.
func TestOneRuleStoryIsNotAnother(t *testing.T) {
	d := open(t)

	at := time.Date(2026, 9, 17, 9, 0, 1, 0, time.UTC)

	for _, rule := range []string{"R-1", "R-2"} {
		if err := d.Happened(RuleTurn{Rule: rule, At: at, What: RuleKept}); err != nil {
			t.Fatalf("write down %s: %v", rule, err)
		}
	}

	if err := d.Happened(RuleTurn{Rule: "R-2", At: at, What: RuleOff}); err != nil {
		t.Fatalf("switch R-2 off: %v", err)
	}

	history, err := d.RuleHistory("R-1")
	if err != nil {
		t.Fatalf("read the history of R-1: %v", err)
	}

	if len(history) != 1 || history[0].What != RuleKept {
		t.Errorf("R-1 has the history %+v, want the one turn that was its own", history)
	}
}

// TestARuleWithNoNameIsWrittenDownNowhere. A rule with no name is one
// somebody wrote by hand and Orbit has not written since, which is allowed
// and is not a reason to refuse the edit that was actually asked for.
func TestARuleWithNoNameIsWrittenDownNowhere(t *testing.T) {
	d := open(t)

	if err := d.Happened(RuleTurn{At: time.Now(), What: RuleKept}); err != nil {
		t.Fatalf("a turn about a rule with no name was refused: %v", err)
	}

	history, err := d.RuleHistory("")
	if err != nil {
		t.Fatalf("read the history of no rule: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("a rule with no name has %d turns, want none", len(history))
	}
}

// TestATurnWithNoClockIsStampedNow. Most callers fill At in nowhere: the
// turn is written as the thing is done, so now is the answer — and a row
// stamped with the zero time would sort before everything for ever.
func TestATurnWithNoClockIsStampedNow(t *testing.T) {
	d := open(t)

	before := time.Now().UTC().Add(-time.Second)

	if err := d.Happened(RuleTurn{Rule: "R-1", What: RuleKept}); err != nil {
		t.Fatalf("write down the turn: %v", err)
	}

	history, err := d.RuleHistory("R-1")
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("the rule has %d turns, want one", len(history))
	}

	if history[0].At.Before(before) {
		t.Errorf("the turn is stamped %s, want a time no older than %s", history[0].At, before)
	}
}

// TestARuleNothingHappenedToHasNoStory. No rows is an empty reading, not a
// refusal: a rule somebody wrote by hand has never been through anything.
func TestARuleNothingHappenedToHasNoStory(t *testing.T) {
	d := open(t)

	history, err := d.RuleHistory("R-404")
	if err != nil {
		t.Fatalf("read the history of a rule nothing happened to: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("it has %d turns, want none", len(history))
	}
}
