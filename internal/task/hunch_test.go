package task

// The decision engine at a gate: what it is allowed to do, and what it is
// never allowed to do.

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/hunch"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// saidSo is a decision engine that answers what a test told it to, and
// remembers what it was shown.
type saidSo struct {
	verdict hunch.Verdict
	err     error
	asked   []hunch.Stop
}

func (s *saidSo) Decide(_ context.Context, about hunch.Stop) (hunch.Verdict, error) {
	s.asked = append(s.asked, about)

	return s.verdict, s.err
}

// sure is a verdict of one word, held with the confidence given.
func sure(choice hunch.Choice, confidence float64) hunch.Verdict {
	return hunch.Verdict{Choice: choice, Confidence: confidence, Model: "jev-test"}
}

// deciding is a run of one waiting phase, with the engine and the settings
// a test wants, and what the record held when it was over.
func deciding(
	t *testing.T, id string, cfg store.Settings, port hunch.Port,
) (*saidSo, []record.Event) {
	t.Helper()

	s, r := fixture(t)

	if err := s.SaveSettings(cfg); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	tk, err := Create(s, r, id, "make the endpoint idempotent", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	said, ok := port.(*saidSo)
	if !ok {
		t.Fatalf("this helper runs the double, and it was given a %T", port)
	}

	// A phase that asks to wait, a gate that polls fast, and a context that
	// gives up: whatever the engine does not release, the deadline does, so
	// a test never sits on a gate.
	ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()

	f := flow.Flow{Name: "careful", Phases: []flow.Phase{{Name: "review", Engine: "fake", Wait: true}}}

	// The error is the deadline in the tests where nothing releases the
	// gate, and that is the point rather than a failure.
	_ = Run(ctx, s, tk, f, fakes(engine.NewFake("did the work")), //nolint:errcheck // see above
		FileGate(s, 10*time.Millisecond, port))

	return said, mustEvents(t, s, tk)
}

// letGo is whether the record says the phase was let go, and by what.
func letGo(events []record.Event) (resumed bool, how string) {
	for _, e := range events {
		if e.Kind == record.PhaseResumed {
			return true, e.Data["how"]
		}
	}

	return false, ""
}

// decided is the decision the record kept, if any.
func decided(events []record.Event) (record.Event, bool) {
	for _, e := range events {
		if e.Kind == record.Decided {
			return e, true
		}
	}

	return record.Event{}, false
}

// TestOffAsksNothing. A machine with a key and the setting off is a machine
// where nothing about a run has changed, which is what makes the key safe
// to have in the environment at all.
func TestOffAsksNothing(t *testing.T) {
	port := &saidSo{verdict: sure(hunch.Done, 0.99)}

	said, events := deciding(t, "ACME-70", store.Settings{}, port)

	if len(said.asked) != 0 {
		t.Errorf("the decision engine was asked %d times with the setting off", len(said.asked))
	}

	if _, ok := decided(events); ok {
		t.Error("the record holds a decision that was never supposed to be made")
	}

	if resumed, _ := letGo(events); resumed {
		t.Error("the phase was let go by something that was switched off")
	}
}

// TestShadowWritesItDownAndActsOnNothing, which is the whole of what a week
// of shadow is for: the answer beside what the reader went on to do.
func TestShadowWritesItDownAndActsOnNothing(t *testing.T) {
	port := &saidSo{verdict: sure(hunch.Done, 0.99)}

	said, events := deciding(t, "ACME-71", store.Settings{Decisions: store.DecisionsShadow}, port)

	if len(said.asked) != 1 {
		t.Fatalf("the decision engine was asked %d times, want once", len(said.asked))
	}

	got, ok := decided(events)
	if !ok {
		t.Fatalf("shadow wrote nothing down: %v", kindsOf(events))
	}

	if got.Data["choice"] != "done" || got.Data["mode"] != store.DecisionsShadow {
		t.Errorf("the record says %v, want a shadow decision of done", got.Data)
	}

	if got.Data["acted"] != "false" {
		t.Errorf("shadow says it acted: %v", got.Data)
	}

	if resumed, how := letGo(events); resumed {
		t.Errorf("shadow let the phase go (how=%q)", how)
	}
}

// TestOnLetsAPhaseGoWhenItIsSureEnough, and says in the record that it was
// the one that did.
func TestOnLetsAPhaseGoWhenItIsSureEnough(t *testing.T) {
	port := &saidSo{verdict: sure(hunch.Done, 0.90)}
	cfg := store.Settings{Decisions: store.DecisionsOn, DecisionFloor: 70}

	said, events := deciding(t, "ACME-72", cfg, port)

	resumed, how := letGo(events)
	if !resumed {
		t.Fatalf("a phase nothing was unsure about is still waiting: %v", kindsOf(events))
	}

	if how != howDecided {
		t.Errorf("the phase was let go by %q, want the decision engine", how)
	}

	got, _ := decided(events)
	if got.Data["acted"] != "true" || got.Data["confidence"] != "0.90" {
		t.Errorf("the record says %v, want an acted decision at 0.90", got.Data)
	}

	// What it was shown, which is the other half of being able to argue
	// with a decision afterwards.
	if len(said.asked) != 1 || said.asked[0].Phase != "review" {
		t.Errorf("the decision engine was shown %+v", said.asked)
	}
}

// TestUnderTheFloorWaits. The floor is the whole safety of this: a verdict
// the model is not sure about is a verdict a person answers, and the
// record still holds what it thought.
func TestUnderTheFloorWaits(t *testing.T) {
	port := &saidSo{verdict: sure(hunch.Done, 0.55)}
	cfg := store.Settings{Decisions: store.DecisionsOn, DecisionFloor: 70}

	_, events := deciding(t, "ACME-73", cfg, port)

	if resumed, _ := letGo(events); resumed {
		t.Error("a phase was let go on a verdict under the floor")
	}

	got, ok := decided(events)
	if !ok || got.Data["acted"] != "false" {
		t.Errorf("the record says %v, want an unacted decision", got.Data)
	}
}

// TestOnlyDoneLetsAPhaseGo. "Again" and "human" both end in a person, and
// a supervisor that resumed a run on either would be reading its own
// verdict backwards.
func TestOnlyDoneLetsAPhaseGo(t *testing.T) {
	for _, choice := range []hunch.Choice{hunch.Again, hunch.Human, hunch.Choice("something else")} {
		port := &saidSo{verdict: sure(choice, 0.99)}
		cfg := store.Settings{Decisions: store.DecisionsOn, DecisionFloor: 70}

		_, events := deciding(t, "ACME-74-"+string(choice), cfg, port)

		if resumed, _ := letGo(events); resumed {
			t.Errorf("a verdict of %q let the phase go", choice)
		}
	}
}

// TestADecisionThatDidNotArriveChangesNothing: an outage at somebody else's
// API is not a thing that happened to the task, so it waits as it always
// did and the record says nothing about it.
func TestADecisionThatDidNotArriveChangesNothing(t *testing.T) {
	port := &saidSo{err: errors.New("the decision engine is not answering")}
	cfg := store.Settings{Decisions: store.DecisionsOn, DecisionFloor: 70}

	_, events := deciding(t, "ACME-75", cfg, port)

	if resumed, _ := letGo(events); resumed {
		t.Error("a phase was let go by a decision that never arrived")
	}

	if _, ok := decided(events); ok {
		t.Error("the record holds a decision that was never made")
	}
}

// TestAPauseAReaderPressedIsNotLifted. Autopilot keeps this rule and so
// does this: the flow's gates are Orbit's to lift, and a brake a person put
// on is theirs.
func TestAPauseAReaderPressedIsNotLifted(t *testing.T) {
	s, r := fixture(t)

	cfg := store.Settings{Decisions: store.DecisionsOn, DecisionFloor: 70, Autopilot: true}
	if err := s.SaveSettings(cfg); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	tk, err := Create(s, r, "ACME-76", "something the reader paused", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Control(s, tk, wordPause); err != nil {
		t.Fatalf("pause it: %v", err)
	}

	port := &saidSo{verdict: sure(hunch.Done, 0.99)}

	ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()

	// Autopilot is on, so nothing but the reader's own pause is holding
	// this phase.
	f := flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "implement", Engine: "fake"}}}

	//nolint:errcheck // the deadline is what ends this run, and that is the point
	_ = Run(ctx, s, tk, f, fakes(engine.NewFake("did the work")),
		FileGate(s, 10*time.Millisecond, port))

	events := mustEvents(t, s, tk)
	if resumed, how := letGo(events); resumed {
		t.Errorf("the reader's own pause was lifted by %q", how)
	}

	if len(port.asked) != 0 {
		t.Error("the decision engine was asked about a pause that was not its to lift")
	}
}

// TestAutopilotStopsForAVerdictItIsSureAbout.
//
// Autopilot lifts the flow's gates, and until there was something that could
// read the work in half a second it lifted them blind: a phase that asked
// to stop was waved through with nothing having looked at what it did. With
// a decision engine it is autopilot that hands over the decision, and the
// run stops where a person is needed.
func TestAutopilotStopsForAVerdictItIsSureAbout(t *testing.T) {
	port := &saidSo{verdict: sure(hunch.Human, 0.95)}
	cfg := store.Settings{Autopilot: true, Decisions: store.DecisionsOn, DecisionFloor: 70}

	_, events := deciding(t, "ACME-77", cfg, port)

	if !parked(events) {
		t.Errorf("autopilot waved through a run the engine says needs a person: %v", kindsOf(events))
	}

	got, ok := decided(events)
	if !ok || got.Data["acted"] != "true" || got.Data["choice"] != "human" {
		t.Errorf("the record says %v, want an acted verdict of human", got.Data)
	}
}

// TestAutopilotIsUntouchedByAVerdictItIsNotSureAbout, and by a verdict of
// done: everything but a confident "somebody has to look at this" is
// autopilot exactly as it was.
func TestAutopilotIsUntouchedByAVerdictItIsNotSureAbout(t *testing.T) {
	for _, v := range []hunch.Verdict{
		sure(hunch.Human, 0.55), // not sure enough
		sure(hunch.Again, 0.60), // not sure enough
		sure(hunch.Done, 0.99),  // sure, and there is nothing to stop for
	} {
		port := &saidSo{verdict: v}
		cfg := store.Settings{Autopilot: true, Decisions: store.DecisionsOn, DecisionFloor: 70}

		id := "ACME-78-" + string(v.Choice) + strconv.Itoa(int(v.Confidence*100))

		_, events := deciding(t, id, cfg, port)

		if parked(events) {
			t.Errorf("autopilot stopped for %q at %.2f", v.Choice, v.Confidence)
		}
	}
}

// TestShadowNeverStopsAutopilotEither. Shadow writes down what it would
// have done, on both sides of the decision.
func TestShadowNeverStopsAutopilotEither(t *testing.T) {
	port := &saidSo{verdict: sure(hunch.Human, 0.99)}
	cfg := store.Settings{Autopilot: true, Decisions: store.DecisionsShadow, DecisionFloor: 70}

	_, events := deciding(t, "ACME-79", cfg, port)

	if parked(events) {
		t.Error("shadow stopped a run")
	}

	got, ok := decided(events)
	if !ok || got.Data["acted"] != "false" {
		t.Errorf("the record says %v, want a shadow decision that acted on nothing", got.Data)
	}
}

// parked is whether the record says the run stopped for somebody.
func parked(events []record.Event) bool {
	for _, e := range events {
		if e.Kind == record.PhaseWaiting {
			return true
		}
	}

	return false
}
