package ui

// Typing to find one model in a catalogue of sixty-five.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// typeInKnobs opens the line and types into it, one keystroke at a time,
// through Update — the routing is half of what is under test.
func typeInKnobs(t *testing.T, m Model, typed string) Model {
	t.Helper()

	m = asModel(t, must(m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})))
	if !m.engines.typing {
		t.Fatal("the knobs screen did not take / as the key that filters it")
	}

	for _, r := range typed {
		m = asModel(t, must(m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})))
	}

	return m
}

// knobModels is the models on the list as it stands.
func knobModels(m Model) []string {
	var out []string

	for _, r := range m.collectEngineRows() {
		if r.kind == rowModel {
			out = append(out, strings.TrimSpace(r.title))
		}
	}

	return out
}

// TestTypingCutsTheCatalogueDown. Folding made sixty-five models possible to
// walk past; it did not make one of them possible to find.
func TestTypingCutsTheCatalogueDown(t *testing.T) {
	m := typeInKnobs(t, knobsOnALongList(t), "model-1")

	models := knobModels(m)
	if len(models) == 0 || len(models) >= 40 {
		t.Fatalf("filtering left %d models, want the ones that match and no more", len(models))
	}

	for _, name := range models {
		if !strings.Contains(name, "model-1") {
			t.Errorf("%q is on the filtered list, and it does not match what was typed", name)
		}
	}

	if said := m.knobFilterLine(m.shownModels()); !strings.Contains(said, "model-1") {
		t.Errorf("the screen says %q, want the line the reader typed", said)
	}
}

// TestAFilterOpensWhatItMatched: a name with the models it was typed to find
// still folded under it says nothing.
func TestAFilterOpensWhatItMatched(t *testing.T) {
	m, _ := testModel(t, 100, 20)
	m.opts.Engines = enginesLongList
	m.knobs.Engine = "opencode"
	m = m.openEngines()

	if len(knobModels(m)) != 0 {
		t.Fatal("the list opened with its models showing, and this test is about a shut one")
	}

	if models := knobModels(typeInKnobs(t, m, "model-2")); len(models) == 0 {
		t.Error("what the filter matched stayed folded away")
	}
}

// TestAnEngineWithNothingMatchingLeavesTheList, rather than sitting there as
// a name whose models are all gone.
func TestAnEngineWithNothingMatchingLeavesTheList(t *testing.T) {
	m, _ := testModel(t, 100, 24)
	m.opts.Engines = func() []EngineInfo {
		return []EngineInfo{
			{Name: "claude", Available: true, Models: []ChoiceInfo{{ID: "opus", Label: "opus"}}},
			{Name: "codex", Available: true, Models: []ChoiceInfo{{ID: "gpt-5", Label: "gpt-5"}}},
		}
	}
	m = m.openEngines()

	m = typeInKnobs(t, m, "opus")

	var names []string

	for _, r := range m.collectEngineRows() {
		if r.kind == rowEngine {
			names = append(names, r.title)
		}
	}

	if len(names) != 1 || names[0] != "claude" {
		t.Errorf("the engines on the list are %v, want only the one holding what matched", names)
	}
}

// TestAnEngineIsAlsoAThingToTypeFor. "codex" is a reasonable thing to write
// when what you want is everything codex runs.
func TestAnEngineIsAlsoAThingToTypeFor(t *testing.T) {
	m := typeInKnobs(t, knobsOnALongList(t), "opencode")

	if got, want := len(knobModels(m)), 40; got != want {
		t.Errorf("typing the engine's own name left %d models, want all %d of them", got, want)
	}
}

// TestNothingMatchingSaysSo, rather than leaving an empty screen the reader
// has to explain to themselves.
func TestNothingMatchingSaysSo(t *testing.T) {
	m := typeInKnobs(t, knobsOnALongList(t), "zzz")

	if models := knobModels(m); len(models) != 0 {
		t.Fatalf("a filter nothing matches left %v", models)
	}

	if said := m.knobFilterLine(m.shownModels()); !strings.Contains(said, "no model matches") {
		t.Errorf("the screen says %q, want it to say nothing matched", said)
	}
}

// TestTheCursorSurvivesTheListShrinkingUnderIt: a cursor past the end of the
// list is a ⏎ that chooses nothing.
func TestTheCursorSurvivesTheListShrinkingUnderIt(t *testing.T) {
	m := knobsOnALongList(t)
	for range 20 {
		m = m.walkKnobs(true)
	}

	m = typeInKnobs(t, m, "model-11")

	n := len(m.selectableEngineIndices(m.collectEngineRows()))
	if m.engines.sel >= n {
		t.Fatalf("the cursor is on row %d of %d after the list shrank", m.engines.sel, n)
	}

	// And the row it landed on is a real one, which selectedKnob asserts.
	if got := selectedKnob(t, m); got == "" {
		t.Error("the cursor is on a row with no name")
	}
}

// TestEnterKeepsTheFilterAndEscClearsIt. They are different gestures on
// purpose: filtering down to three models and then choosing one is the whole
// point of having filtered.
func TestEnterKeepsTheFilterAndEscClearsIt(t *testing.T) {
	m := typeInKnobs(t, knobsOnALongList(t), "model-1")

	kept := asModel(t, must(m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})))
	if kept.engines.typing || kept.engines.filter != "model-1" {
		t.Errorf("after ⏎ the line is typing=%v filter=%q, want the filter kept and the line closed",
			kept.engines.typing, kept.engines.filter)
	}

	cleared := asModel(t, must(m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})))
	if cleared.engines.typing || cleared.engines.filter != "" {
		t.Errorf("after esc the line is typing=%v filter=%q, want the whole catalogue back",
			cleared.engines.typing, cleared.engines.filter)
	}
}

// TestAModelFoundByTypingIsChosenWithTwoPressesOfEnter: the first closes the
// line with the list still cut down, and the second takes what is under the
// cursor — which the arrows moved there while the line was open.
func TestAModelFoundByTypingIsChosenWithTwoPressesOfEnter(t *testing.T) {
	m := typeInKnobs(t, knobsOnALongList(t), "model-17")

	m = m.walkKnobs(true) // off the engine's name, onto what it found

	want := selectedKnob(t, m)

	m = asModel(t, must(m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})))
	m = asModel(t, must(m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})))

	if m.knobs.Model != want {
		t.Errorf("the model in force is %q, want the %q the filter found", m.knobs.Model, want)
	}
}

// TestBackspaceGivesTheCatalogueBackOneLetterAtATime.
func TestBackspaceGivesTheCatalogueBackOneLetterAtATime(t *testing.T) {
	m := typeInKnobs(t, knobsOnALongList(t), "model-11")

	few := len(knobModels(m))

	m = asModel(t, must(m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})))

	if more := len(knobModels(m)); more <= few {
		t.Errorf("backspacing left %d models where the longer line had %d", more, few)
	}
}

// TestLeavingTheScreenLeavesNothingTyped, so the next reader who opens the
// knobs is not looking at somebody else's search.
func TestLeavingTheScreenLeavesNothingTyped(t *testing.T) {
	m := typeInKnobs(t, knobsOnALongList(t), "model-1")
	m = asModel(t, must(m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})))
	m = asModel(t, must(m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})))

	if m.engines.filter != "" || m.engines.typing {
		t.Errorf("the knobs kept filter=%q typing=%v after the screen was left",
			m.engines.filter, m.engines.typing)
	}
}

// TestTheWaysOutSayTheFilterIsThere, on a screen whose verbs are a line at
// the foot of it.
func TestTheWaysOutSayTheFilterIsThere(t *testing.T) {
	m := knobsOnALongList(t)

	drawn := strings.Join(m.enginesRows(m.frame.Body.H, m.frame.Body.W), "\n")
	if !strings.Contains(drawn, "filter") {
		t.Error("the knobs screen does not say that / filters it")
	}
}
