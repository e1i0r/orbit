package task

// Changing engine in the middle of a task.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// under is an engine answering to a name of its own, so that a test can have
// two of them. Every engine this package ships answers "fake".
type under struct {
	engine.Engine
	as string
}

func (u under) Name() string { return u.as }

// twoEngines is one that has nothing left and one that works.
func twoEngines(said string) map[string]engine.Engine {
	return map[string]engine.Engine{
		"claude": under{spentEngine{said: said}, "claude"},
		"codex":  under{engine.NewFake("wrote it"), "codex"},
	}
}

// oneEngineFlow is a flow of one phase, put to the engine named.
func oneEngineFlow(name string) flow.Flow {
	return flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "implement", Engine: name}}}
}

// onlyFree says one engine has something left and the rest have not, with
// their windows coming back in the time given.
func onlyFree(name string, back time.Duration) Allowance {
	return func(asked string) Spare {
		if asked == name {
			return Spare{Free: true}
		}

		return Spare{Back: back}
	}
}

// autopilotOn writes the switch every relay that happens without asking
// depends on.
func autopilotOn(t *testing.T, s *store.Store, on bool) {
	t.Helper()

	if err := s.SaveSettings(store.Settings{Autopilot: on}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
}

// relayIn is the first hand-over in the record, and whether there was one.
func relayIn(t *testing.T, s *store.Store, tk Task) (record.Event, bool) {
	t.Helper()

	for _, e := range eventsOf(t, s, tk) {
		if e.Kind == record.TaskRelayed {
			return e, true
		}
	}

	return record.Event{}, false
}

const ranDryOut = "Claude AI usage limit reached"

// TestAnEngineThatRanOutHandsTheTaskOn is the whole project in one test: the
// engine holding a task runs out, another one takes it, and the task
// finishes.
func TestAnEngineThatRanOutHandsTheTaskOn(t *testing.T) {
	s, tk := nowhere(t, "ACME-1", "do the thing")
	autopilotOn(t, s, true)

	engines := twoEngines(ranDryOut)

	err := Run(context.Background(), s, tk, oneEngineFlow("claude"), engines, nil,
		onlyFree("codex", 0))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got, relayed := relayIn(t, s, tk)
	if !relayed {
		t.Fatalf("the task changed engine and the record does not say so:\n%v", kindsFor(t, s, tk))
	}

	if got.Data["from"] != "claude" || got.Data["to"] != "codex" {
		t.Errorf("the relay says %q handed to %q, want claude handed to codex",
			got.Data["from"], got.Data["to"])
	}

	if got.Data["why"] != whyRanOut {
		t.Errorf("the relay says it happened because %q, want %q", got.Data["why"], whyRanOut)
	}

	kinds := kindsFor(t, s, tk)
	if count(kinds, record.TaskFinished) != 1 {
		t.Errorf("the task did not finish after the relay:\n%v", kinds)
	}

	// The phase that ran under the new engine says so. A record that kept
	// naming the engine the flow named would credit claude with codex's
	// work, which is the one thing this cannot afford.
	if !ranUnder(t, s, tk, "codex") {
		t.Errorf("no phase in the record was run by codex:\n%v", kinds)
	}
}

// ranUnder is whether any phase of the task was started under that engine.
func ranUnder(t *testing.T, s *store.Store, tk Task, name string) bool {
	t.Helper()

	for _, e := range eventsOf(t, s, tk) {
		if e.Kind == record.PhaseStarted && e.Data["engine"] == name {
			return true
		}
	}

	return false
}

// TestWithoutAutopilotItStopsAndSaysWhoCouldTakeIt is the other half of the
// one decision: autopilot already means whether a run walks its flow without
// stopping for a person, and a relay is the same question.
func TestWithoutAutopilotItStopsAndSaysWhoCouldTakeIt(t *testing.T) {
	s, tk := nowhere(t, "ACME-2", "do the thing")
	autopilotOn(t, s, false)

	engines := twoEngines(ranDryOut)

	err := Run(context.Background(), s, tk, oneEngineFlow("claude"), engines, nil,
		onlyFree("codex", 0))
	if err == nil {
		t.Fatal("Run: want an error when the engine ran out and nobody said to carry on")
	}

	if _, relayed := relayIn(t, s, tk); relayed {
		t.Error("the task changed engine with autopilot off")
	}

	last := lastOfKind(t, s, tk, record.TaskNeedsEngine)
	if last.Kind == "" {
		t.Fatalf("the run stopped without saying it needs an engine:\n%v", kindsFor(t, s, tk))
	}

	if !strings.Contains(last.Data["engines"], "codex") {
		t.Errorf("it stopped without naming who could take it: %q", last.Data["engines"])
	}
}

// TestWithNothingLeftItSaysUntilWhen is the difference between a task that is
// waiting and a task that is over. Allowances come back.
func TestWithNothingLeftItSaysUntilWhen(t *testing.T) {
	s, tk := nowhere(t, "ACME-3", "do the thing")
	autopilotOn(t, s, true)

	engines := twoEngines(ranDryOut)

	err := Run(context.Background(), s, tk, oneEngineFlow("claude"), engines, nil,
		onlyFree("nobody", 90*time.Minute))
	if err == nil {
		t.Fatal("Run: want an error when no engine has anything left")
	}

	last := lastOfKind(t, s, tk, record.TaskNoEngine)
	if last.Kind == "" {
		t.Fatalf("the run stopped without saying there is no engine:\n%v", kindsFor(t, s, tk))
	}

	if last.Data["back"] != (90 * time.Minute).String() {
		t.Errorf("it says the allowance comes back in %q, want %q",
			last.Data["back"], (90 * time.Minute).String())
	}

	if !strings.Contains(last.Text, "1h30m") {
		t.Errorf("the line a person reads does not say until when: %q", last.Text)
	}
}

// TestAnEngineThatBrokeIsNotRelayed. A compile error is a compile error
// whoever is typing, and spending another allowance on it buys nothing.
func TestAnEngineThatBrokeIsNotRelayed(t *testing.T) {
	s, tk := nowhere(t, "ACME-4", "do the thing")
	autopilotOn(t, s, true)

	engines := twoEngines("./main.go:12:2: declared and not used")

	err := Run(context.Background(), s, tk, oneEngineFlow("claude"), engines, nil,
		onlyFree("codex", 0))
	if err == nil {
		t.Fatal("Run: want an error when the engine broke")
	}

	if _, relayed := relayIn(t, s, tk); relayed {
		t.Error("a phase that broke was handed to another engine")
	}

	kinds := kindsFor(t, s, tk)
	if count(kinds, record.TaskFailed) != 1 {
		t.Errorf("a run that broke did not end as a failure:\n%v", kinds)
	}
}

// TestAnEngineIsNeverHandedTheSamePhaseTwice is the loop guard. An engine
// that ran out a minute ago can still read as free — a proxy caches, a
// rollout file is as fresh as the last run — so who has been asked is
// remembered rather than re-read.
func TestAnEngineIsNeverHandedTheSamePhaseTwice(t *testing.T) {
	s, tk := nowhere(t, "ACME-5", "do the thing")
	autopilotOn(t, s, true)

	// Both of them run out, and the port says both are free. Without the
	// guard this hands the phase back and forth for ever.
	engines := map[string]engine.Engine{
		"claude": under{spentEngine{said: ranDryOut}, "claude"},
		"codex":  under{spentEngine{said: ranDryOut}, "codex"},
	}

	free := func(string) Spare { return Spare{Free: true} }

	err := Run(context.Background(), s, tk, oneEngineFlow("claude"), engines, nil, free)
	if err == nil {
		t.Fatal("Run: want an error when every engine has run out")
	}

	if n := count(kindsFor(t, s, tk), record.TaskRelayed); n != 1 {
		t.Errorf("the phase was handed on %d times, want 1 — there are two engines", n)
	}
}

// TestChangingEngineByHandIsWrittenDownAsARelay. `orbit task start -engine`
// could already do this and nothing said so: the run that followed read as an
// ordinary run that happened to be on codex.
func TestChangingEngineByHandIsWrittenDownAsARelay(t *testing.T) {
	s, tk := nowhere(t, "ACME-6", "do the thing")

	// Nothing has run yet, so there is nothing to relay from: a first run is
	// a start and not a hand-over.
	if err := HandedTo(s, tk, "codex"); err != nil {
		t.Fatalf("HandedTo: %v", err)
	}

	if _, relayed := relayIn(t, s, tk); relayed {
		t.Fatal("the first run of a task was written down as a relay")
	}

	autopilotOn(t, s, false)

	engines := map[string]engine.Engine{
		"claude": under{engine.NewFake("wrote it"), "claude"},
		"codex":  under{engine.NewFake("wrote it"), "codex"},
	}

	if err := Run(context.Background(), s, tk, oneEngineFlow("claude"), engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if err := HandedTo(s, tk, "codex"); err != nil {
		t.Fatalf("HandedTo: %v", err)
	}

	got, relayed := relayIn(t, s, tk)
	if !relayed {
		t.Fatalf("a person changed the engine and the record does not say so:\n%v", kindsFor(t, s, tk))
	}

	if got.Data["from"] != "claude" || got.Data["why"] != whyChosen {
		t.Errorf("the relay reads %q → %q because %q, want claude → codex because %q",
			got.Data["from"], got.Data["to"], got.Data["why"], whyChosen)
	}

	// The same engine twice is not a hand-over.
	if err := HandedTo(s, tk, "claude"); err != nil {
		t.Fatalf("HandedTo: %v", err)
	}

	if n := count(kindsFor(t, s, tk), record.TaskRelayed); n != 1 {
		t.Errorf("the record holds %d relays, want 1", n)
	}
}

// lastOfKind is the last event of that kind, and a zero event when the
// record holds none.
func lastOfKind(t *testing.T, s *store.Store, tk Task, kind string) record.Event {
	t.Helper()

	var found record.Event

	for _, e := range eventsOf(t, s, tk) {
		if e.Kind == kind {
			found = e
		}
	}

	return found
}

// TestTheEngineThatTakesOverIsToldWhyItHasTo is the three pieces of this
// project meeting: the ending has a name, the name is in the record, and the
// engine that arrives reads it.
//
// Without it the relay hands a fresh engine a worktree of half-finished
// changes and no account of them — which is the same forgetting the relay
// was built to stop, one step further along.
func TestTheEngineThatTakesOverIsToldWhyItHasTo(t *testing.T) {
	s, tk := nowhere(t, "ACME-7", "do the thing")
	autopilotOn(t, s, true)

	took := engine.NewFake("wrote it")
	engines := map[string]engine.Engine{
		"claude": under{spentEngine{said: ranDryOut}, "claude"},
		"codex":  under{took, "codex"},
	}

	if err := Run(context.Background(), s, tk, oneEngineFlow("claude"), engines, nil,
		onlyFree("codex", 0)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(took.Calls) != 1 {
		t.Fatalf("the engine that took over ran %d times, want 1", len(took.Calls))
	}

	said := took.Calls[0].Prompt
	for _, want := range []string{"## The attempt before you", "ran out of tokens"} {
		if !strings.Contains(said, want) {
			t.Errorf("the engine that took over was not told %q:\n%s", want, said)
		}
	}

	// And it was not handed the session of the engine that ran out. A
	// session id claude wrote means nothing to codex, and a run that passed
	// one along would read as resumed while the engine started from nothing.
	if took.Calls[0].Resume != "" {
		t.Errorf("codex was handed a session to resume: %q", took.Calls[0].Resume)
	}
}
