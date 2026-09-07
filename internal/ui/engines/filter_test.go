package engines

// Typing to find one model in a catalogue of sixty-five.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/roster"
)

// typeInKnobs opens the line and types into it, one keystroke at a time,
// through Key — the routing is half of what is under test.
func typeInKnobs(t *testing.T, s State, e Env, typed string) State {
	t.Helper()

	s, _ = s.Key(press("/"), e)
	if !s.typing {
		t.Fatal("the knobs screen did not take / as the key that filters it")
	}

	for _, r := range typed {
		s, _ = s.Key(press(string(r)), e)
	}

	return s
}

// knobModels is the models on the list as it stands.
func knobModels(s State, e Env) []string {
	var out []string

	for _, r := range s.collectEngineRows(e) {
		if r.kind == rowModel {
			out = append(out, strings.TrimSpace(r.title))
		}
	}

	return out
}

// TestTypingCutsTheCatalogueDown. Folding made sixty-five models possible to
// walk past; it did not make one of them possible to find.
func TestTypingCutsTheCatalogueDown(t *testing.T) {
	s, e := knobsOnALongList(t)
	s = typeInKnobs(t, s, e, "model-1")

	models := knobModels(s, e)
	if len(models) == 0 || len(models) >= 40 {
		t.Fatalf("filtering left %d models, want the ones that match and no more", len(models))
	}

	for _, name := range models {
		if !strings.Contains(name, "model-1") {
			t.Errorf("%q is on the filtered list, and it does not match what was typed", name)
		}
	}

	if said := s.knobFilterLine(s.shownModels(e), e); !strings.Contains(said, "model-1") {
		t.Errorf("the screen says %q, want the line the reader typed", said)
	}
}

// TestAFilterOpensWhatItMatched: a name with the models it was typed to find
// still folded under it says nothing.
func TestAFilterOpensWhatItMatched(t *testing.T) {
	e := world(t)
	e.Engines = enginesLongList
	e.Settled = "opencode"

	s := Open(0, Knobs{Engine: "opencode"})

	if len(knobModels(s, e)) != 0 {
		t.Fatal("the list opened with its models showing, and this test is about a shut one")
	}

	if models := knobModels(typeInKnobs(t, s, e, "model-2"), e); len(models) == 0 {
		t.Error("what the filter matched stayed folded away")
	}
}

// TestAnEngineWithNothingMatchingLeavesTheList, rather than sitting there as
// a name whose models are all gone.
func TestAnEngineWithNothingMatchingLeavesTheList(t *testing.T) {
	e := world(t)
	e.Engines = func() []roster.Engine {
		return []roster.Engine{
			{Name: "claude", Available: true, Models: []roster.Choice{{ID: "opus", Label: "opus"}}},
			{Name: "codex", Available: true, Models: []roster.Choice{{ID: "gpt-5", Label: "gpt-5"}}},
		}
	}

	s := typeInKnobs(t, Open(0, Knobs{}), e, "opus")

	var names []string

	for _, r := range s.collectEngineRows(e) {
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
	s, e := knobsOnALongList(t)
	s = typeInKnobs(t, s, e, "opencode")

	if got, want := len(knobModels(s, e)), 40; got != want {
		t.Errorf("typing the engine's own name left %d models, want all %d of them", got, want)
	}
}

// TestNothingMatchingSaysSo, rather than leaving an empty screen the reader
// has to explain to themselves.
func TestNothingMatchingSaysSo(t *testing.T) {
	s, e := knobsOnALongList(t)
	s = typeInKnobs(t, s, e, "zzz")

	if models := knobModels(s, e); len(models) != 0 {
		t.Fatalf("a filter nothing matches left %v", models)
	}

	if said := s.knobFilterLine(s.shownModels(e), e); !strings.Contains(said, "no model matches") {
		t.Errorf("the screen says %q, want it to say nothing matched", said)
	}
}

// TestTheCursorSurvivesTheListShrinkingUnderIt: a cursor past the end of the
// list is a ⏎ that chooses nothing.
func TestTheCursorSurvivesTheListShrinkingUnderIt(t *testing.T) {
	s, e := knobsOnALongList(t)
	for range 20 {
		s = s.walkKnobs(true, e)
	}

	s = typeInKnobs(t, s, e, "model-11")

	n := len(selectableEngineIndices(s.collectEngineRows(e)))
	if s.sel >= n {
		t.Fatalf("the cursor is on row %d of %d after the list shrank", s.sel, n)
	}

	// And the row it landed on is a real one, which selectedKnob asserts.
	if got := selectedKnob(t, s, e); got == "" {
		t.Error("the cursor is on a row with no name")
	}
}

// TestEnterKeepsTheFilterAndEscClearsIt. They are different gestures on
// purpose: filtering down to three models and then choosing one is the whole
// point of having filtered.
func TestEnterKeepsTheFilterAndEscClearsIt(t *testing.T) {
	s, e := knobsOnALongList(t)
	s = typeInKnobs(t, s, e, "model-1")

	kept, _ := s.Key(press("enter"), e)
	if kept.typing || kept.filter != "model-1" {
		t.Errorf("after ⏎ the line is typing=%v filter=%q, want the filter kept and the line closed",
			kept.typing, kept.filter)
	}

	cleared, _ := s.Key(press("esc"), e)
	if cleared.typing || cleared.filter != "" {
		t.Errorf("after esc the line is typing=%v filter=%q, want the whole catalogue back",
			cleared.typing, cleared.filter)
	}
}

// TestAModelFoundByTypingIsChosenWithTwoPressesOfEnter: the first closes the
// line with the list still cut down, and the second takes what is under the
// cursor — which the arrows moved there while the line was open.
func TestAModelFoundByTypingIsChosenWithTwoPressesOfEnter(t *testing.T) {
	s, e := knobsOnALongList(t)
	s = typeInKnobs(t, s, e, "model-17")

	s = s.walkKnobs(true, e) // off the engine's name, onto what it found

	want := selectedKnob(t, s, e)

	s, _ = s.Key(press("enter"), e)
	s, _ = s.Key(press("enter"), e)

	if s.knobs.Model != want {
		t.Errorf("the model in force is %q, want the %q the filter found", s.knobs.Model, want)
	}
}

// TestBackspaceGivesTheCatalogueBackOneLetterAtATime.
func TestBackspaceGivesTheCatalogueBackOneLetterAtATime(t *testing.T) {
	s, e := knobsOnALongList(t)
	s = typeInKnobs(t, s, e, "model-11")

	few := len(knobModels(s, e))

	s, _ = s.Key(press("backspace"), e)

	if more := len(knobModels(s, e)); more <= few {
		t.Errorf("backspacing left %d models where the longer line had %d", more, few)
	}
}

// TestLeavingTheScreenLeavesNothingTyped, so the next reader who opens the
// knobs is not looking at somebody else's search.
func TestLeavingTheScreenLeavesNothingTyped(t *testing.T) {
	s, e := knobsOnALongList(t)
	s = typeInKnobs(t, s, e, "model-1")
	s, _ = s.Key(press("esc"), e)
	s, _ = s.Key(press("esc"), e)

	if s.filter != "" || s.typing {
		t.Errorf("the knobs kept filter=%q typing=%v after the screen was left",
			s.filter, s.typing)
	}
}

// TestTheWaysOutSayTheFilterIsThere, on a screen whose verbs are a line at
// the foot of it.
func TestTheWaysOutSayTheFilterIsThere(t *testing.T) {
	s, e := knobsOnALongList(t)

	if !strings.Contains(strings.Join(drawn(s, e), "\n"), "filter") {
		t.Error("the knobs screen does not say that / filters it")
	}
}
