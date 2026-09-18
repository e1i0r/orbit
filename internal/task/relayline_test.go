package task

// The line a run leaves behind when every engine has run out.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// TestAWaitIsSaidTheWayAPersonSaysOne.
//
// time.Duration prints "1h35m0s", which is a machine talking. The seconds
// are noise in a sentence about waiting an hour and a half — and a minute is
// where a number starts being worth printing, so a wait of exactly one is a
// number and not "less than a minute", which would tell a reader to come
// back at no particular time.
func TestAWaitIsSaidTheWayAPersonSaysOne(t *testing.T) {
	for _, one := range []struct {
		wait time.Duration
		want string
	}{
		{0, "less than a minute"},
		{30 * time.Second, "less than a minute"},
		{59 * time.Second, "less than a minute"},
		{time.Minute, "1m"},
		{90 * time.Second, "2m"},
		{35 * time.Minute, "35m"},
		{time.Hour, "1h"},
		{time.Hour + 35*time.Minute, "1h35m"},
	} {
		if got := spellOut(one.wait); got != one.want {
			t.Errorf("a wait of %s is said as %q, want %q", one.wait, got, one.want)
		}
	}
}

// TestARunWithNowhereToGoSaysUntilWhen.
//
// Until when is the point of it. Allowances come back, so an hour is an
// answer and "abandoned" is not — and when nothing on this machine can say
// what the hour is, it has to say that rather than name one. A wait of none
// is not a wait: " for another less than a minute" tells a reader to try
// again now, which is what has just failed.
func TestARunWithNowhereToGoSaysUntilWhen(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-70", "a task with nowhere to run", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	p := flow.Phase{Name: "implement", Engine: "claude"}

	soon := noEngine(s, tk, p, "claude", 90*time.Minute)
	if soon == nil {
		t.Fatal("a run with nowhere to go did not fail")
	}

	told := lastNoEngine(t, s, tk)
	if !strings.Contains(told.Text, "for another 1h30m") {
		t.Errorf("a run that can go again in an hour and a half reads %q", told.Text)
	}

	if told.Data["back"] != (90 * time.Minute).String() {
		t.Errorf("the field says %q, want the exact duration for whatever reads it next", told.Data["back"])
	}

	// And with nothing able to say when, it says that instead.
	blind := noEngine(s, tk, p, "claude", 0)
	if blind == nil {
		t.Fatal("a run with nowhere to go did not fail")
	}

	told = lastNoEngine(t, s, tk)
	if !strings.Contains(told.Text, "nothing here can say when one comes back") {
		t.Errorf("a run nobody can time reads %q", told.Text)
	}

	if strings.Contains(told.Text, "for another") {
		t.Errorf("a run nobody can time was given a time: %q", told.Text)
	}

	if got, there := told.Data["back"]; there {
		t.Errorf("the field says it comes back in %q, which nothing measured", got)
	}
}

// lastNoEngine is the newest event saying a run had nowhere to go.
func lastNoEngine(t *testing.T, s *store.Store, tk Task) record.Event {
	t.Helper()

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Kind == record.TaskNoEngine {
			return events[i]
		}
	}

	t.Fatal("nothing in the record says the run had nowhere to go")

	return record.Event{}
}

// TestTheWaitReportedIsTheFirstAllowanceBack.
//
// A run that has nowhere to go tells the reader when to come back, and the
// answer is the soonest of them: told the last one instead, they wait two
// hours for a window that opened in ten minutes, and the task sits on the
// board the whole time.
func TestTheWaitReportedIsTheFirstAllowanceBack(t *testing.T) {
	s, _ := fixture(t)

	engines := map[string]engine.Engine{
		"claude":   engine.NewFake("done"),
		"codex":    engine.NewFake("done"),
		"opencode": engine.NewFake("done"),
	}

	soonest := 10 * time.Minute
	back := map[string]time.Duration{
		"claude":   2 * time.Hour,
		"codex":    soonest,
		"opencode": time.Hour,
	}

	free, wait := whoIsFree(s, engines, nil, func(name string) Spare {
		return Spare{Back: back[name]}
	})

	if len(free) != 0 {
		t.Fatalf("every engine had run out and %v read as free", free)
	}

	if wait != soonest {
		t.Errorf("the run says to come back in %s, want the %s the first window takes", wait, soonest)
	}

	// An engine nothing can time says nothing about when, and must not read
	// as the soonest of them.
	_, blind := whoIsFree(s, engines, nil, func(name string) Spare {
		if name == "claude" {
			return Spare{}
		}

		return Spare{Back: back[name]}
	})

	if blind != soonest {
		t.Errorf("an engine nobody can time changed the answer to %s", blind)
	}
}
