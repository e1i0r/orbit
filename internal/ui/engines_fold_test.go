package ui

// Folding the engines list: an engine's models show under its name or stay
// behind it.

import (
	"slices"
	"strings"
	"testing"
)

// twoEnginesWithModels is what folding is for: more than one engine on the
// machine, each with a catalogue of its own.
func twoEnginesWithModels() []EngineInfo {
	return []EngineInfo{
		{
			Name:      "claude",
			Available: true,
			Models:    []ChoiceInfo{{ID: "", Label: "default"}, {ID: "opus", Label: "opus"}},
		},
		{
			Name:      "opencode",
			Available: true,
			Models:    []ChoiceInfo{{ID: "", Label: "default"}, {ID: "kimi", Label: "kimi"}, {ID: "grok", Label: "grok"}},
		},
	}
}

func foldableKnobs(t *testing.T) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m.opts.Engines = twoEnginesWithModels
	m.knobs.Engine = "claude"

	return m.openEngines()
}

// knobRows is what the list is showing, by row title.
func knobRows(m Model) []string {
	var titles []string

	for _, r := range m.collectEngineRows() {
		titles = append(titles, strings.TrimSpace(r.title))
	}

	return titles
}

// TestTheEngineInForceIsTheOneOpen, and the rest are their names: a list
// that opened every engine would be back to the wall of models folding is
// here to answer.
func TestTheEngineInForceIsTheOneOpen(t *testing.T) {
	m := foldableKnobs(t)

	titles := knobRows(m)
	if !slices.Contains(titles, "opus") {
		t.Errorf("the models of the engine in force are not on the list: %v", titles)
	}

	if slices.Contains(titles, "kimi") {
		t.Errorf("an engine nobody opened is showing its models: %v", titles)
	}
}

// TestAnEngineOpensWithoutBeingChosen: comparing two catalogues is the
// reason to open a second one, and a reader who had to switch engines to
// read one would have changed what the run does to look something up.
func TestAnEngineOpensWithoutBeingChosen(t *testing.T) {
	m := foldableKnobs(t)
	m = m.selectKnobEngine("opencode")

	next, _ := m.enginesKey(press("right"))

	m2 := asModel(t, next)
	if titles := knobRows(m2); !slices.Contains(titles, "kimi") {
		t.Fatalf("→ on opencode left its models off the list: %v", titles)
	}

	if m2.knobs.Engine != "claude" {
		t.Errorf("opening an engine changed the run's engine to %q", m2.knobs.Engine)
	}
}

// TestClosingFromInsideASectionLandsOnItsName. The row the cursor was on is
// gone once the section is shut, and a cursor left where it was would be
// standing on whatever slid up into the gap.
func TestClosingFromInsideASectionLandsOnItsName(t *testing.T) {
	m := foldableKnobs(t)

	m.engines.sel = 1 // the first model under claude

	next, _ := m.enginesKey(press("left"))

	m2 := asModel(t, next)

	rows := m2.collectEngineRows()

	idxs := m2.selectableEngineIndices(rows)
	if slices.Contains(knobRows(m2), "opus") {
		t.Fatalf("← inside claude's models left them on the list")
	}

	if on := rows[idxs[m2.engines.sel]]; on.kind != rowEngine || on.engine != "claude" {
		t.Errorf("after closing, the cursor is on %+v, want claude's own row", on)
	}
}

// TestEnterFoldsTheEngineAlreadyInForce: there is nothing left to choose on
// that row, so the key that chooses is free to be the key that folds.
func TestEnterFoldsTheEngineAlreadyInForce(t *testing.T) {
	m := foldableKnobs(t)
	m = m.selectKnobEngine("claude")

	shut, _ := m.enginesKey(press("enter"))

	m2 := asModel(t, shut)
	if slices.Contains(knobRows(m2), "opus") {
		t.Fatalf("⏎ on the engine in force left its models on the list")
	}

	if m2.knobs.Engine != "claude" {
		t.Errorf("folding changed the run's engine to %q", m2.knobs.Engine)
	}

	open, _ := m2.enginesKey(press("enter"))
	if m3 := asModel(t, open); !slices.Contains(knobRows(m3), "opus") {
		t.Errorf("⏎ again left claude's models off the list")
	}
}

// TestChoosingAnEngineOpensIt: the choice is made to be followed by a model,
// and a list that stayed shut would take two keys to say one thing.
func TestChoosingAnEngineOpensIt(t *testing.T) {
	m := foldableKnobs(t)
	m = m.selectKnobEngine("opencode")

	next, _ := m.enginesKey(press("enter"))

	m2 := asModel(t, next)
	if m2.knobs.Engine != "opencode" {
		t.Fatalf("⏎ on opencode = engine %q", m2.knobs.Engine)
	}

	if titles := knobRows(m2); !slices.Contains(titles, "kimi") {
		t.Errorf("the engine just chosen is not showing its models: %v", titles)
	}
}

// TestAShutEngineSaysHowMuchIsBehindIt. A name with nothing beside it is a
// name; the count is what tells a reader there is a catalogue under it and
// how big a list ← and → are hiding.
func TestAShutEngineSaysHowMuchIsBehindIt(t *testing.T) {
	m := foldableKnobs(t)

	drawn := strings.Join(m.enginesRows(m.frame.Body.H, m.frame.Body.W), "\n")
	if !strings.Contains(drawn, "3 models") {
		t.Errorf("the shut opencode row does not say what it holds:\n%s", drawn)
	}
}
