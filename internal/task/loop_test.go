package task

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// tddFlow is one phase and then a loop that goes round until the check
// passes, which is the shape the issue draws.
func tddFlow(check string, max int) flow.Flow {
	return flow.Flow{Name: "tdd", Phases: []flow.Phase{
		{Name: "implement", Engine: "fake"},
		{Name: "green", Loop: &flow.Loop{
			Phases: []flow.Phase{{Name: "fix", Engine: "fake"}},
			Until:  []flow.Gate{{Name: "unit", Command: check}},
			Max:    max,
		}},
	}}
}

// TestALoopGoesRoundUntilTheCheckPasses. The one thing that says the work is
// done is an exit code: a model asked whether its own work passes is being
// asked to mark its own paper.
func TestALoopGoesRoundUntilTheCheckPasses(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-29", "make the tests green", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("wrote a fix")

	if err := Run(context.Background(), s, tk, tddFlow(countingGate("3"), 5), fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// One implement, then the loop's phase once per turn: the check fails
	// twice and passes on the third.
	if len(fake.Calls) != 4 {
		t.Errorf("the engine ran %d times, want 4 — implement, then three turns of the loop", len(fake.Calls))
	}

	kinds := kindsOf(mustEvents(t, s, tk))
	if got := count(kinds, record.LoopChecked); got != 3 {
		t.Errorf("the record holds %d checks, want one per turn: %v", got, kinds)
	}
}

// TestALoopStopsAtItsCap, and the task is stuck rather than failed: the
// history of the turns is what the reader picks it up with.
func TestALoopStopsAtItsCap(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-30", "a check nothing will satisfy", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("tried")

	err = Run(context.Background(), s, tk, tddFlow("echo 'FAIL: TestIdempotent'; exit 1", 2), fakes(fake), nil)
	if err == nil {
		t.Fatal("Run: want an error when the loop runs out of turns")
	}

	if len(fake.Calls) != 3 {
		t.Errorf("the engine ran %d times, want 3 — implement and the two turns the loop allows", len(fake.Calls))
	}

	events := mustEvents(t, s, tk)

	last := events[len(events)-1]
	if last.Kind != record.TaskStuck {
		t.Fatalf("the record ends in %q, want task.stuck: %v", last.Kind, kindsOf(events))
	}

	if !strings.Contains(last.Text, "FAIL: TestIdempotent") {
		t.Errorf("the task is stuck without the history of what failed:\n%s", last.Text)
	}

	// Every turn is there and numbered from the first, because the reader
	// is following what changed between them: counted from anywhere else,
	// the account opens with a turn nobody can see the start of.
	for _, want := range []string{"Turn 1 — ", "Turn 2 — "} {
		if !strings.Contains(last.Text, want) {
			t.Errorf("the account of the stuck loop does not carry %q:\n%s", want, last.Text)
		}
	}

	if strings.Contains(last.Text, "Turn 0") {
		t.Errorf("the turns are numbered from zero:\n%s", last.Text)
	}

	if last.Data["attempts"] != "2" {
		t.Errorf("it says %q attempts, want the two turns the loop allowed", last.Data["attempts"])
	}
}

// TestEachTurnIsToldWhatTheCheckSaid. Retrying without the error is
// repeating blind, which is 3.1's rule applied to the outer loop.
func TestEachTurnIsToldWhatTheCheckSaid(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-31", "learn from the failure", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("tried")
	if err := Run(context.Background(), s, tk, tddFlow(countingGate("2"), 3), fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) < 3 {
		t.Fatalf("the engine ran %d times, want implement and two turns", len(fake.Calls))
	}

	second := fake.Calls[2].Prompt
	if !strings.Contains(second, "undefined: Retry") || !strings.Contains(second, "unit") {
		t.Errorf("the second turn was not told what the check said:\n%s", second)
	}
}

// TestAPhaseWithNoLoopIsUntouched.
func TestAPhaseWithNoLoopIsUntouched(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-32", "a straight line", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("done")
	if err := Run(context.Background(), s, tk, twoPhases(), fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 2 {
		t.Errorf("the engine ran %d times, want the two phases", len(fake.Calls))
	}
}

// TestALoopPhaseIsAskedAsItself. The place a phase holds inside a loop says
// nothing about the flow around it: reading the flow's phase at that index
// asked the wrong phase its instructions, and panicked outright when the loop
// held more phases than the flow did.
func TestALoopPhaseIsAskedAsItself(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-31", "the loop's phases speak for themselves", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "loop-only", Phases: []flow.Phase{
		{Name: "green", Loop: &flow.Loop{
			Phases: []flow.Phase{
				{Name: "write-test", Engine: "fake", Prompt: "write the failing test first"},
				{Name: "make-it-pass", Engine: "fake", Prompt: "now make it pass"},
			},
			Until: []flow.Gate{{Name: "unit", Command: "true"}},
			Max:   2,
		}},
	}}

	fake := engine.NewFake("done")

	if err := Run(context.Background(), s, tk, f, fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 2 {
		t.Fatalf("the engine ran %d times, want one per phase of the single turn", len(fake.Calls))
	}

	for i, want := range []string{"write the failing test first", "now make it pass"} {
		if !strings.Contains(fake.Calls[i].Prompt, want) {
			t.Errorf("phase %d was not asked its own instructions:\n%s", i+1, fake.Calls[i].Prompt)
		}
	}
}

// TestWhatAPersonSaidGoesToTheFirstPhaseOfTheFirstTurnAndNowhereElse.
//
// It was taken before the loop began, and the phase.started every inner
// phase emits is what marks it consumed — so a loop that never took it
// swallowed it, and the phase after the loop was told nothing either.
// Repeated into every turn, a note becomes an instruction the model answers
// three times.
func TestWhatAPersonSaidGoesToTheFirstPhaseOfTheFirstTurnAndNowhereElse(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-35", "make the tests green", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	const note = "hold off on the backoff"
	if err := Note(s, tk, note); err != nil {
		t.Fatalf("Note: %v", err)
	}

	// A loop of two phases that goes round twice, so there are four inner
	// prompts and only the first of them may carry it.
	twice := flow.Flow{Name: "tdd", Phases: []flow.Phase{
		{Name: "green", Loop: &flow.Loop{
			Phases: []flow.Phase{{Name: "fix", Engine: "fake"}, {Name: "recheck", Engine: "fake"}},
			Until:  []flow.Gate{{Name: "unit", Command: countingGate("2")}},
			Max:    3,
		}},
	}}

	fake := engine.NewFake("tried")
	if err := Run(context.Background(), s, tk, twice, fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 4 {
		t.Fatalf("the engine ran %d times, want two phases over two turns", len(fake.Calls))
	}

	carried := 0

	for i, c := range fake.Calls {
		if !strings.Contains(c.Prompt, note) {
			continue
		}

		carried++

		if i != 0 {
			t.Errorf("call %d was told what a person said, and it is not the first", i)
		}
	}

	if carried != 1 {
		t.Errorf("what a person said reached %d of the four prompts, want the first alone", carried)
	}
}

// fedLoopFlow is a phase, then a loop whose inner phase asked to be fed
// what the phase before it said — which is the shape of the coverage flow
// Orbit ships: implement, then fix until the checks pass, reading what the
// last try left behind.
func fedLoopFlow(check string, max int) flow.Flow {
	return flow.Flow{Name: "fed", Phases: []flow.Phase{
		{Name: "implement", Engine: "fake"},
		{Name: "green", Loop: &flow.Loop{
			Phases: []flow.Phase{{Name: "fix", Engine: "fake", FeedOutput: true}},
			Until:  []flow.Gate{{Name: "unit", Command: check}},
			Max:    max,
		}},
	}}
}

// TestAPhaseInsideALoopIsFedWhatItAskedFor. feed_output was read for every
// phase of a flow except the ones inside a loop, where nothing set it at
// all: the coverage flow Orbit ships says feed_output on the phase that
// fixes what the checks caught, and that phase was handed nothing. On the
// first turn what it is owed is the output of the phase before the loop,
// and on every turn after that the last thing the loop itself said.
func TestAPhaseInsideALoopIsFedWhatItAskedFor(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-31", "fix it until it passes", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("what the phase said")

	f := fedLoopFlow(countingGate("3"), 5)
	if err := Run(context.Background(), s, tk, f, fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The first call is implement, which asked for nothing; the three
	// after it are the loop's turns.
	if len(fake.Calls) != 4 {
		t.Fatalf("the engine ran %d times, want 4 — implement and three turns", len(fake.Calls))
	}

	for i, call := range fake.Calls[1:] {
		if !strings.Contains(call.Prompt, "## Previous phase output") {
			t.Errorf("turn %d was fed nothing, and its phase asked to be fed:\n%s", i+1, call.Prompt)

			continue
		}

		if !strings.Contains(call.Prompt, "what the phase said") {
			t.Errorf("turn %d was fed something other than the last output:\n%s", i+1, call.Prompt)
		}
	}
}
