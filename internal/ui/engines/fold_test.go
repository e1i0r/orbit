package engines

// Folding the engines list: an engine's models show under its name or stay
// behind it.

import (
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/roster"
)

// twoEnginesWithModels is what folding is for: more than one engine on the
// machine, each with a catalogue of its own.
func twoEnginesWithModels() []roster.Engine {
	return []roster.Engine{
		{
			Name:      "claude",
			Available: true,
			Models:    []roster.Choice{{ID: "", Label: "default"}, {ID: "opus", Label: "opus"}},
		},
		{
			Name:      "opencode",
			Available: true,
			Models:    []roster.Choice{{ID: "", Label: "default"}, {ID: "kimi", Label: "kimi"}, {ID: "grok", Label: "grok"}},
		},
	}
}

func foldableKnobs(t *testing.T) (State, Env) {
	t.Helper()

	e := world(t)
	e.Engines = twoEnginesWithModels

	return Open(0, Knobs{Engine: "claude"}), e
}

// knobRows is what the list is showing, by row title.
func knobRows(s State, e Env) []string {
	var titles []string

	for _, r := range s.collectEngineRows(e) {
		titles = append(titles, strings.TrimSpace(r.title))
	}

	return titles
}

// TestTheKnobsOpenShut, every engine of them, including the one the run is
// already using: four catalogues are eighty-odd rows, and the screen is
// worth reading at a glance only while it is names and dials.
func TestTheKnobsOpenShut(t *testing.T) {
	s, e := foldableKnobs(t)

	titles := knobRows(s, e)
	for _, model := range []string{"opus", "kimi"} {
		if slices.Contains(titles, model) {
			t.Errorf("the screen opened with %q on it: %v", model, titles)
		}
	}

	if !slices.Contains(titles, "claude") || !slices.Contains(titles, "opencode") {
		t.Errorf("the engines themselves are not on the list: %v", titles)
	}
}

// TestAnEngineOpensWithoutBeingChosen: comparing two catalogues is the
// reason to open a second one, and a reader who had to switch engines to
// read one would have changed what the run does to look something up.
func TestAnEngineOpensWithoutBeingChosen(t *testing.T) {
	s, e := foldableKnobs(t)
	s = s.selectKnobEngine("opencode", e)

	s, _ = s.Key(press("right"), e)

	if titles := knobRows(s, e); !slices.Contains(titles, "kimi") {
		t.Fatalf("→ on opencode left its models off the list: %v", titles)
	}

	if s.knobs.Engine != "claude" {
		t.Errorf("opening an engine changed the run's engine to %q", s.knobs.Engine)
	}
}

// TestClosingFromInsideASectionLandsOnItsName. The row the cursor was on is
// gone once the section is shut, and a cursor left where it was would be
// standing on whatever slid up into the gap.
func TestClosingFromInsideASectionLandsOnItsName(t *testing.T) {
	s, e := foldableKnobs(t)
	s = s.foldEngine("claude", true)

	s.sel = 1 // the first model under claude

	s, _ = s.Key(press("left"), e)

	rows := s.collectEngineRows(e)

	idxs := selectableEngineIndices(rows)

	if slices.Contains(knobRows(s, e), "opus") {
		t.Fatalf("← inside claude's models left them on the list")
	}

	if on := rows[idxs[s.sel]]; on.kind != rowEngine || on.engine != "claude" {
		t.Errorf("after closing, the cursor is on %+v, want claude's own row", on)
	}
}

// TestEnterFoldsTheEngineAlreadyInForce: there is nothing left to choose on
// that row, so the key that chooses is free to be the key that folds.
func TestEnterFoldsTheEngineAlreadyInForce(t *testing.T) {
	s, e := foldableKnobs(t)
	s = s.selectKnobEngine("claude", e)

	s, _ = s.Key(press("enter"), e)

	if !slices.Contains(knobRows(s, e), "opus") {
		t.Fatalf("⏎ on the engine in force left its models off the list")
	}

	if s.knobs.Engine != "claude" {
		t.Errorf("folding changed the run's engine to %q", s.knobs.Engine)
	}

	shut, _ := s.Key(press("enter"), e)
	if slices.Contains(knobRows(shut, e), "opus") {
		t.Errorf("⏎ again left claude's models on the list")
	}
}

// TestChoosingAnEngineOpensIt: the choice is made to be followed by a model,
// and a list that stayed shut would take two keys to say one thing.
func TestChoosingAnEngineOpensIt(t *testing.T) {
	s, e := foldableKnobs(t)
	s = s.selectKnobEngine("opencode", e)

	s, _ = s.Key(press("enter"), e)
	if s.knobs.Engine != "opencode" {
		t.Fatalf("⏎ on opencode = engine %q", s.knobs.Engine)
	}

	if titles := knobRows(s, e); !slices.Contains(titles, "kimi") {
		t.Errorf("the engine just chosen is not showing its models: %v", titles)
	}
}

// TestAShutEngineSaysWhatIsBehindIt: the model it would run with, and how
// many there are to choose from. Folded is the state this screen is usually
// in, so what the fold hides has to be said on the line that hides it.
func TestAShutEngineSaysWhatIsBehindIt(t *testing.T) {
	s, e := foldableKnobs(t)
	s.knobs.Model = "opus"

	shown := strings.Join(drawn(s, e), "\n")
	if !strings.Contains(shown, "3 models") {
		t.Errorf("the shut opencode row does not say what it holds:\n%s", shown)
	}

	if !strings.Contains(shown, "opus") {
		t.Errorf("the shut claude row does not say which model it would run:\n%s", shown)
	}
}
