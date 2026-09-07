package engines

// engines_more_coverage_test.go is engines.go's own remaining branches:
// applyEngineChoice's four row kinds, enginesKey's navigation wraparound
// and its two ways out, abandonEngines from the start dialog, setOpt
// itself, and the two collectEngineRows sections that only show when an
// engine has no effort dial or no thinking mode.

import (
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/roster"
	"github.com/e1i0r/orbit/internal/words"
)

// enginesTestList is a small engine roster with one available engine that
// has both dials, one disabled engine with setup steps, and one available
// engine with neither dial — every shape collectEngineRows branches on,
// none of which the fixture's default single-engine fallback offers.
func enginesTestList() []roster.Engine {
	return []roster.Engine{
		{
			Name:      "claude",
			Available: true,
			Models:    []roster.Choice{{ID: "", Label: "default"}, {ID: "opus", Label: "opus"}, {ID: "sonnet", Label: "sonnet"}},
			Efforts:   []roster.Choice{{ID: "", Label: "default"}, {ID: "low", Label: "low"}, {ID: "high", Label: "high"}},
			CanThink:  true,
		},
		{
			// Not installed, and still carrying its dials: what an engine
			// offers and whether this machine can run it are two facts,
			// and the port answers both for every engine.
			Name:      "codex",
			Available: false,
			Setup:     func(*words.Printer) []string { return []string{"install codex", "run codex login"} },
			Models:    []roster.Choice{{ID: "", Label: "default"}, {ID: "o3", Label: "o3"}, {ID: "o3-mini", Label: "o3-mini"}},
			Efforts:   []roster.Choice{{ID: "", Label: "default"}, {ID: "low", Label: "low"}, {ID: "high", Label: "high"}},
		},
		{
			Name:      "bare",
			Available: true,
		},
	}
}

// TestEveryKindOfRowDoesItsOwnThingWhenItIsChosen.
func TestEveryKindOfRowDoesItsOwnThingWhenItIsChosen(t *testing.T) {
	s, e := open(t)

	// A disabled row opens the setup steps instead of applying anything.
	bare := engineRow{kind: rowEngine, engine: "codex", disabled: true, setup: []string{"a"}}

	shown, _ := s.take(bare, e)
	if !shown.showingSetup || shown.setupEngine != "codex" {
		t.Fatalf("a disabled row left showingSetup=%v for %q", shown.showingSetup, shown.setupEngine)
	}

	chosen, out := s.take(engineRow{kind: rowEngine, engine: "codex"}, e)
	if chosen.knobs.Engine != "codex" || chosen.knobs.Model != "" {
		t.Errorf("choosing an engine = %+v, want codex and the model cleared", chosen.knobs)
	}

	// And it is written down: a dial the reader moved is a dial they expect
	// to still be there tomorrow.
	if len(out.Set) != 1 || out.Set[0] != (Setting{Name: "engine", Value: "codex"}) {
		t.Errorf("choosing an engine wrote %+v", out.Set)
	}

	model, out := s.take(engineRow{kind: rowModel, engine: "claude", id: "opus"}, e)
	if model.knobs.Engine != "claude" || model.knobs.Model != "opus" {
		t.Errorf("choosing a model = %+v", model.knobs)
	}

	if len(out.Set) != 1 || out.Set[0] != (Setting{Name: "model", Value: "opus"}) {
		t.Errorf("choosing a model wrote %+v", out.Set)
	}

	if effort, _ := s.take(engineRow{kind: rowEffort, id: "high"}, e); effort.knobs.Effort != "high" {
		t.Errorf("choosing an effort = %+v", effort.knobs)
	}

	if think, _ := s.take(engineRow{kind: rowThinking, id: "on"}, e); think.knobs.Thinking != "on" {
		t.Errorf("choosing a thinking mode = %+v", think.knobs)
	}
}

// TestKnobChipDefaultsEngineName is a knob set with something chosen but
// no engine named — the model, effort or thinking dial can be turned
// without ever visiting the engine row — where the chip falls back to
// "claude" rather than leaving the chip's first word blank.
func TestKnobChipDefaultsEngineName(t *testing.T) {
	_, e := open(t)

	got := Chip(Knobs{Model: "opus"}, e)
	if !strings.Contains(got, "claude") {
		t.Errorf("the chip reads %q, want it to default the engine name to claude", got)
	}
}

// TestAKeyTheScreenHasNoUseForChangesNothing, rather than panicking or
// reaching the board underneath.
func TestAKeyTheScreenHasNoUseForChangesNothing(t *testing.T) {
	s, e := open(t)
	before := s.sel

	after, out := s.Key(press("z"), e)
	if after.sel != before || out.Leave {
		t.Errorf("an unmatched key left sel=%d leave=%v", after.sel, out.Leave)
	}
}

// TestLeavingGoesBackToWhereItWasOpenedFrom, which the screen was handed and
// hands back: which screens there are is the window's business.
func TestLeavingGoesBackToWhereItWasOpenedFrom(t *testing.T) {
	_, e := open(t)

	const fromTheStartDialog = 7

	_, out := Open(fromTheStartDialog, Knobs{}).Key(press("esc"), e)
	if !out.Leave || out.Back != fromTheStartDialog {
		t.Errorf("esc answered leave=%v back=%d, want it back where it came from", out.Leave, out.Back)
	}
}

// TestTheSetupStepsOwnTheKeyboardWhileTheyAreUp.
func TestTheSetupStepsOwnTheKeyboardWhileTheyAreUp(t *testing.T) {
	s, e := open(t)
	s.showingSetup, s.setupEngine = true, "codex"

	// Any other key does nothing while the steps are up.
	if after, _ := s.Key(press("x"), e); !after.showingSetup {
		t.Error("an unrelated key closed the setup steps")
	}

	// Back closes them, and not the screen.
	after, out := s.Key(press("esc"), e)
	if after.showingSetup || out.Leave {
		t.Errorf("esc left showingSetup=%v leave=%v, want the steps closed and the screen up",
			after.showingSetup, out.Leave)
	}
}

// TestTheArrowsWrapAndEnterTakesWhatIsUnderTheCursor.
func TestTheArrowsWrapAndEnterTakesWhatIsUnderTheCursor(t *testing.T) {
	s, e := open(t)

	idxs := selectableEngineIndices(s.collectEngineRows(e))
	if len(idxs) == 0 {
		t.Fatal("the fixture catalogue has no row to stand on")
	}

	// Up from the first row wraps to the last.
	s.sel = 0
	if up, _ := s.Key(press("up"), e); up.sel != len(idxs)-1 {
		t.Errorf("up from the first row = %d, want %d", up.sel, len(idxs)-1)
	}

	// Down from the last row wraps to the first.
	s.sel = len(idxs) - 1
	if down, _ := s.Key(press("down"), e); down.sel != 0 {
		t.Errorf("down from the last row = %d, want 0", down.sel)
	}

	// ⏎ applies whatever row is under the cursor.
	s.sel = 0

	took, _ := s.Key(press("enter"), e)
	if took.knobs.Engine == "" && !took.showingSetup {
		t.Error("⏎ neither took a choice nor opened the setup steps")
	}

	// Back leaves the screen and reports the dials in the band.
	_, out := s.Key(press("esc"), e)
	if !out.Leave || out.Said == "" {
		t.Errorf("esc answered leave=%v said=%q", out.Leave, out.Said)
	}
}

// TestCollectEngineRowsNoDialsSections is the "bare" engine from
// enginesTestList: available, but with no effort dial and no thinking
// mode, which are the two header-only sections collectEngineRows draws in
// their place.
func TestCollectEngineRowsNoDialsSections(t *testing.T) {
	_, e := open(t)
	s := Open(0, Knobs{Engine: "bare"})

	rows := s.collectEngineRows(e)

	var sawNoEffort, sawNoThinking bool

	for _, r := range rows {
		if r.kind == rowHeader {
			if strings.Contains(r.title, "no effort dial") {
				sawNoEffort = true
			}

			if strings.Contains(r.title, "no thinking mode") {
				sawNoThinking = true
			}
		}
	}

	if !sawNoEffort {
		t.Errorf("expected a 'no effort dial' row for an engine with no efforts")
	}

	if !sawNoThinking {
		t.Errorf("expected a 'no thinking mode' row for an engine that cannot think")
	}
}

// TestTheEnginesScreenShowsWhatThePortSaysAndNothingElse. A claude table on
// the screen to fall back on draws claude, opus, sonnet, haiku for a port
// that answers nothing — or answers something else. A window
// that shows an engine the build cannot run, and hides one it can, is worse
// than a window that shows nothing.
func TestTheEnginesScreenShowsWhatThePortSaysAndNothingElse(t *testing.T) {
	e := world(t)
	e.Engines = func() []roster.Engine {
		return []roster.Engine{{
			Name:      "zeta",
			Available: true,
			Models:    []roster.Choice{{ID: "zeta/one", Label: "one"}, {ID: "zeta/two", Label: "two"}},
			Efforts:   []roster.Choice{{ID: "high", Label: "high"}},
		}}
	}

	s := Open(0, Knobs{Engine: "zeta"}).foldEngine("zeta", true)

	var named, models []string

	for _, r := range s.collectEngineRows(e) {
		switch r.kind {
		case rowEngine:
			named = append(named, r.engine)
		case rowModel:
			models = append(models, r.id)
		case rowHeader, rowEffort, rowThinking:
		}
	}

	if !slices.Equal(named, []string{"zeta"}) {
		t.Errorf("the screen offers %v, want only what the port answered", named)
	}

	if !slices.Equal(models, []string{"zeta/one", "zeta/two"}) {
		t.Errorf("the screen offers the models %v, want zeta's own", models)
	}
}

// TestAWindowWithNoEnginesPortInventsNone. Saying nothing is the only honest
// answer this package has: it may not name internal/engine, so any table it
// carried would be a copy waiting to drift from the one the build runs.
func TestAWindowWithNoEnginesPortInventsNone(t *testing.T) {
	e := world(t)
	e.Engines = nil

	s := Open(0, Knobs{})

	for _, r := range s.collectEngineRows(e) {
		if r.kind == rowEngine || r.kind == rowModel {
			t.Errorf("a window with no engines port offered %q", r.engine+r.id)
		}
	}
}
