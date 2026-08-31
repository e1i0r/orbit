package ui

// The one table of engines, seen from the screens that draw dials off it.
//
// Every test here hands the window an engine no part of this package could
// have guessed at. That is the whole point: a table of a screen's own
// offered opencode a model called llama-3.3, which no opencode has ever
// answered to — a phase built with it is a phase that cannot run.

import (
	"slices"
	"strings"
	"testing"
)

// zetaEngines is one engine this build has never heard of, with both dials
// and with the "whatever the engine is configured for" choice the port puts
// in front of them.
func zetaEngines() []EngineInfo {
	return []EngineInfo{{
		Name:      "zeta",
		Available: true,
		Models:    []ChoiceInfo{{ID: "", Label: "default"}, {ID: "zeta/one", Label: "one"}, {ID: "zeta/two", Label: "two"}},
		Efforts:   []ChoiceInfo{{ID: "", Label: "default"}, {ID: "brisk", Label: "brisk"}},
		CanThink:  true,
	}}
}

// The flow editor's per-phase dials. They were three lists written out in
// flowsdelta.go, and the model one offered every engine sonnet — claude's
// alone — so a codex phase could be built holding a model codex refuses.
func TestTheFlowEditorDialsAreTheEnginesOwn(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = zetaEngines
	m = m.startCreateFlow()

	for _, tc := range []struct {
		field int
		got   func(Model) string
		want  []string
	}{
		{flowFieldEngine, func(m Model) string { return m.flows.cur().Engine }, []string{"zeta"}},
		{flowFieldModel, func(m Model) string { return m.flows.cur().Model }, []string{"zeta/one", "zeta/two"}},
		{flowFieldEffort, func(m Model) string { return m.flows.cur().Effort }, []string{"brisk"}},
	} {
		m.flows.field = tc.field
		// Around the dial twice, so a list longer than the engine's would
		// show up as a value that is not on it rather than as an order.
		for range 2 * len(tc.want) {
			next, _ := m.handleFlowFieldDelta(1)
			m = next

			if got := tc.got(m); !slices.Contains(tc.want, got) {
				t.Errorf("field %d cycled onto %q, which is not one of zeta's %v", tc.field, got, tc.want)
			}
		}
	}
}

// The effort knob on the start dialog. Its list was written out in
// startdials.go and held xhigh, which codex does not have.
func TestCyclingTheEffortKnobStaysOnTheEnginesOwn(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = zetaEngines

	efforts, _ := m.effortsFor("zeta")
	for range 2 * len(efforts) {
		m = m.cycleEffort()
		if !slices.Contains(efforts, m.knobs.Effort) {
			t.Fatalf("the effort knob cycled onto %q, which is not one of zeta's %v", m.knobs.Effort, efforts)
		}
	}
}

// TestNoScreenNamesAnEngineThisBuildDoesNotHave. Eleven places in this
// package answered "claude" whenever nothing named an engine — the knob
// chip, the supervisor thread, the overview pane, the flow builder, the
// phase a new flow is born with. On a build without claude every one of
// them named something that was not going to run, and the reader had no
// way to tell that from a choice they had made.
func TestNoScreenNamesAnEngineThisBuildDoesNotHave(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = zetaEngines

	// The saved engine is the reader's own word and outranks the roster,
	// so it is cleared here: what is under test is what the window says
	// when nobody has said anything.
	if err := m.opts.Settings.SetEngine(""); err != nil {
		t.Fatalf("clear the engine setting: %v", err)
	}

	if got := m.dialEngine(""); got != "zeta" {
		t.Fatalf("with nothing chosen the window names %q, want the only engine there is", got)
	}

	m = m.startCreateFlow()
	if got := m.flows.cur().Engine; got != "zeta" {
		t.Errorf("a new flow's first phase is born on %q, want zeta", got)
	}

	frame := strings.Join(m.flowsBuilderRows(30, 100), "\n")
	for _, invented := range []string{"claude", "codex", "opencode", "sonnet", "haiku"} {
		if strings.Contains(frame, invented) {
			t.Errorf("the flow builder names %q, which this build does not have:\n%s", invented, frame)
		}
	}

	if !strings.Contains(frame, "zeta") {
		t.Errorf("the flow builder names no engine at all:\n%s", frame)
	}
}

// A thread line's author was matched against a list of five names, so a
// sixth engine's answers were drawn in a person's colour and its markdown
// was re-wrapped as plain text.
func TestAThreadLineIsRecognisedAsAnEnginesByTheRoster(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = zetaEngines

	if !m.isEngineName("zeta") {
		t.Error("a line written by zeta is taken for a person's")
	}

	if !m.isEngineName("supervisor") {
		t.Error("the supervisor's own line is taken for a person's")
	}

	// A thread is a record and can hold an engine this build no longer
	// has, which a person did not type either.
	if !m.isEngineName("gemini") {
		t.Error("a line recorded from an engine this build dropped is taken for a person's")
	}

	if m.isEngineName("elio") {
		t.Error("a person's line is taken for an engine's")
	}
}
