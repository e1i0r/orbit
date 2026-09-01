package ui

// What can be done to the thing under the cursor, band by band. The key bar
// hides a refused verb and the menu shows it with the reason, so a refusal
// without a reason is a hole in the menu and a reason the catalogues cannot
// say is a hole in every language but English. Both are failures here.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// printers builds the two Printers this file compares. It redirects
// $ORBIT_HOME first: words.For overlays $ORBIT_HOME/lang/*.json on the
// embedded catalogue, and a test that reads the machine it runs on is a test
// whose result depends on the machine it runs on.
func printers(t *testing.T) (english, spanish *words.Printer) {
	t.Helper()
	t.Setenv("ORBIT_HOME", t.TempDir())

	return words.For("en"), words.For("es")
}

type affordanceCase struct {
	name     string
	task     view.Task
	settings Conditions
	offered  []string
}

// everyCase is one task per shape the board can hold. The offered list is
// the verbs whose key the bar may show; everything else in the menu is
// greyed with a reason.
func everyCase() []affordanceCase {
	can := Conditions{CanResume: true}

	return []affordanceCase{{
		name:     "a task written down and never started",
		task:     view.Task{ID: "ACME-1", Band: view.ToDo},
		settings: can,
		offered:  []string{"enter", "D"},
	}, {
		name:     "a run working in a phase",
		task:     view.Task{ID: "ACME-2", Band: view.Running, Phase: "implement", Live: view.LiveHeld, Attempt: 1, Engine: "claude"},
		settings: can,
		offered:  []string{"enter", "p", "x", "b"},
	}, {
		// A run marker nobody could read. Every verb below turns on whether
		// a process holds this task, and nothing here can say, so the only
		// thing left is to open it — which is what its reason asks for.
		name:     "a run whose marker could not be read",
		task:     view.Task{ID: "ACME-11", Band: view.Running, Live: view.LiveUnknown, Attempt: 1, Engine: "claude"},
		settings: can,
		offered:  []string{"enter"},
	}, {
		// h is not on these three lists, and that is the fix this plan
		// made: handing the keyboard back is offered where a keyboard was
		// taken, not wherever one could have been. A run the reader merely
		// paused wants r.
		name:     "a run the reader has paused",
		task:     view.Task{ID: "ACME-3", Band: view.Running, Live: view.LiveHeld, Attempt: 1, Engine: "claude", Reason: view.Reason{Key: view.ReasonHeld}},
		settings: can,
		offered:  []string{"enter", "r", "x", "b", "t"},
	}, {
		name:     "a phase waiting at the gate its flow asked for",
		task:     view.Task{ID: "ACME-4", Band: view.NeedsYou, Live: view.LiveHeld, Attempt: 1, Engine: "claude", Reason: view.Reason{Key: view.ReasonGate}},
		settings: can,
		offered:  []string{"enter", "r", "x", "b", "t"},
	}, {
		name:     "the same gate with autopilot on",
		task:     view.Task{ID: "ACME-5", Band: view.NeedsYou, Live: view.LiveHeld, Attempt: 1, Engine: "claude", Reason: view.Reason{Key: view.ReasonGate}},
		settings: Conditions{Autopilot: true, CanResume: true},
		offered:  []string{"enter", "r", "x", "b", "t"},
	}, {
		// The same paused run, with the keyboard actually taken. This is
		// the only shape on the board where h means anything, and it is the
		// one Conditions.Taken exists to tell apart from the one above.
		name:     "a paused run whose keyboard this reader took",
		task:     view.Task{ID: "ACME-10", Band: view.Running, Live: view.LiveHeld, Attempt: 1, Engine: "claude", Reason: view.Reason{Key: view.ReasonHeld}},
		settings: Conditions{CanResume: true, Taken: true},
		offered:  []string{"enter", "r", "x", "b", "t", "h"},
	}, {
		name:     "a run that failed and whose process is gone",
		task:     view.Task{ID: "ACME-6", Band: view.NeedsYou, Attempt: 1, Engine: "claude", Reason: view.Reason{Key: view.ReasonFailed}},
		settings: can,
		offered:  []string{"enter", "b", "t", "D"},
	}, {
		name:     "a finished task nobody has read",
		task:     view.Task{ID: "ACME-7", Band: view.Done, Attempt: 1, Engine: "claude"},
		settings: can,
		offered:  []string{"enter", "b", "t", "d", "D"},
	}, {
		name:     "a finished task already read",
		task:     view.Task{ID: "ACME-8", Band: view.Done, Attempt: 1, Engine: "claude", Read: true},
		settings: can,
		offered:  []string{"enter", "b", "t", "D"},
	}, {
		name:     "a paused run on an engine that cannot resume a session",
		task:     view.Task{ID: "ACME-9", Band: view.Running, Live: view.LiveHeld, Attempt: 1, Engine: "codex", Reason: view.Reason{Key: view.ReasonHeld}},
		settings: Conditions{},
		offered:  []string{"enter", "r", "x", "b"},
	}}
}

// verbsOffered lists the first key of every affordance that is offered, in
// the order Affordances returned them.
func verbsOffered(as []Affordance) []string {
	var offered []string

	for _, a := range as {
		if a.OK {
			offered = append(offered, a.Key.Keys()[0])
		}
	}

	return offered
}

func TestEachBandOffersTheVerbsThatMeanSomethingForIt(t *testing.T) {
	english, _ := printers(t)
	keys := NewKeys(english)

	for _, c := range everyCase() {
		t.Run(c.name, func(t *testing.T) {
			got := verbsOffered(keys.Affordances(c.task, c.settings))
			if strings.Join(got, " ") != strings.Join(c.offered, " ") {
				t.Errorf("offers %v, want %v", got, c.offered)
			}
		})
	}
}

// TestTheMenuIsTheSameListForEveryTask is what makes the menu readable: the
// verbs do not move about between tasks, only their answers change. A menu
// whose entries reorder is one the reader has to read rather than reach for.
func TestTheMenuIsTheSameListForEveryTask(t *testing.T) {
	english, _ := printers(t)
	keys := NewKeys(english)

	var want []string

	for _, c := range everyCase() {
		var got []string
		for _, a := range keys.Affordances(c.task, c.settings) {
			got = append(got, a.Key.Keys()[0])
		}

		if want == nil {
			want = got
			continue
		}

		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Fatalf("%s lists %v, want %v", c.name, got, want)
		}
	}
}

// TestEveryRefusalCarriesAReasonBothCataloguesCanSay is the check that keeps
// a reason key from being invented and never translated. English falls back
// to the sentence at the call site whatever happens, so a Spanish sentence
// identical to it is a key es.json never gained.
func TestEveryRefusalCarriesAReasonBothCataloguesCanSay(t *testing.T) {
	english, spanish := printers(t)

	keys := NewKeys(english)
	for _, c := range everyCase() {
		for _, a := range keys.Affordances(c.task, c.settings) {
			verb := a.Key.Keys()[0]
			if a.OK {
				if a.WhyNot != (words.Arg{}) {
					t.Errorf("%s: %q is offered and still carries %+v", c.name, verb, a.WhyNot)
				}

				if said := a.Why(english); said != "" {
					t.Errorf("%s: %q is offered and says %q", c.name, verb, said)
				}

				continue
			}

			if a.WhyNot.Name == "" {
				t.Errorf("%s: %q is refused with no reason — the menu would grey it and say nothing", c.name, verb)
				continue
			}

			said, dicho := a.Why(english), a.Why(spanish)
			if said == "" {
				t.Errorf("%s: %q is refused with reason %q and no sentence", c.name, verb, a.WhyNot.Name)
			}

			if dicho == said {
				t.Errorf("%s: %q is refused with reason %q, which es.json does not carry — a Spanish reader gets %q", c.name, verb, a.WhyNot.Name, dicho)
			}
		}
	}
}

// find returns the affordance for one key, so a test can ask about a verb by
// name rather than by position.
func find(as []Affordance, verb string) Affordance {
	for _, a := range as {
		if a.Key.Keys()[0] == verb {
			return a
		}
	}

	return Affordance{}
}

// TestARefusalNamesWhatToDoNext walks the three reasons that differ from one
// another only by what the reader should press next. They are separate keys
// rather than one sentence with a placeholder because the placeholder would
// be a state word, and a state word substituted into a translated sentence
// is one English word in the middle of a Spanish one.
func TestARefusalNamesWhatToDoNext(t *testing.T) {
	english, _ := printers(t)
	keys := NewKeys(english)
	gate := view.Task{Band: view.NeedsYou, Live: view.LiveHeld, Attempt: 1, Engine: "claude", Reason: view.Reason{Key: view.ReasonGate}}

	held := find(keys.Affordances(view.Task{Band: view.Running, Live: view.LiveHeld, Attempt: 1, Reason: view.Reason{Key: view.ReasonHeld}}, Conditions{}), "p")
	if held.WhyNot.Name != whyPauseAlreadyHeld {
		t.Errorf("pausing a paused run says %q, want %q", held.WhyNot.Name, whyPauseAlreadyHeld)
	}

	waiting := find(keys.Affordances(gate, Conditions{}), "p")
	if waiting.WhyNot.Name != whyPauseAlreadyWaiting {
		t.Errorf("pausing a run waiting at a gate says %q, want %q", waiting.WhyNot.Name, whyPauseAlreadyWaiting)
	}

	lifting := find(keys.Affordances(gate, Conditions{Autopilot: true}), "p")
	if lifting.WhyNot.Name != whyPauseAutopilotIsLifting {
		t.Errorf("pausing a gate autopilot is lifting says %q, want %q — the switch is what the reader has to reach for", lifting.WhyNot.Name, whyPauseAutopilotIsLifting)
	}
}

// TestTheEngineThatCannotResumeIsNamed is the rule internal/engine states in
// its own package doc: an engine that cannot resume a session greys out the
// button and says why. Saying why means saying which engine.
func TestTheEngineThatCannotResumeIsNamed(t *testing.T) {
	english, spanish := printers(t)
	keys := NewKeys(english)

	paused := view.Task{Band: view.Running, Live: view.LiveHeld, Attempt: 1, Engine: "codex", Reason: view.Reason{Key: view.ReasonHeld}}
	for _, verb := range []string{"t", "h"} {
		a := find(keys.Affordances(paused, Conditions{}), verb)
		if a.OK {
			t.Fatalf("%q is offered on an engine that cannot resume a session", verb)
		}

		if a.WhyNot.Value != "codex" {
			t.Errorf("%q refuses with value %q, want the engine's name", verb, a.WhyNot.Value)
		}

		for _, p := range []*words.Printer{english, spanish} {
			if said := a.Why(p); !strings.Contains(said, "codex") {
				t.Errorf("%q says %q, which does not name the engine", verb, said)
			}
		}
	}
}

// TestTheEngineIsAskedAboutOneTaskAtATime is the other half of the sentence
// above being true. Naming the engine is only honest if the answer is about
// that engine. A standing bool — an AND over every engine configured — makes
// a build with two engines, one of which cannot resume, refuse t on every
// task and tell each of them that its own engine is the one at fault.
func TestTheEngineIsAskedAboutOneTaskAtATime(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m.opts.CanResume = func(engine string) bool { return engine == "claude" }
	for engine, want := range map[string]bool{"claude": true, "codex": false, "": false} {
		if got := m.conditions(view.Task{ID: "ACME-1", Engine: engine}).CanResume; got != want {
			t.Errorf("a task on %q is told CanResume=%v, want %v", engine, got, want)
		}
	}

	m.opts.CanResume = nil
	if m.conditions(view.Task{ID: "ACME-1", Engine: "claude"}).CanResume {
		t.Error("a window with no way to ask about engines answered yes")
	}
}

// TestAskIsListedAndRefused is the tool being honest about its own gap in
// the same voice it uses about an engine's. The verb exists in the menu, it
// is refused, and the reason says what to do instead.
func TestAskIsListedAndRefused(t *testing.T) {
	english, _ := printers(t)

	keys := NewKeys(english)
	for _, c := range everyCase() {
		ask := find(keys.Affordances(c.task, c.settings), "a")
		if ask.OK {
			t.Fatalf("%s: ask is offered, and nothing implements it", c.name)
		}

		if ask.WhyNot.Name != whyAskNotBuilt {
			t.Fatalf("%s: ask refuses with %q, want %q", c.name, ask.WhyNot.Name, whyAskNotBuilt)
		}
	}
}
