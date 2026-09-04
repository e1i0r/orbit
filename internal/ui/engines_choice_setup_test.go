package ui

// engines_more_coverage_test.go is engines.go's own remaining branches:
// applyEngineChoice's four row kinds, enginesKey's navigation wraparound
// and its two ways out, abandonEngines from the start dialog, setOpt
// itself, and the two collectEngineRows sections that only show when an
// engine has no effort dial or no thinking mode.

import (
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// enginesTestList is a small engine roster with one available engine that
// has both dials, one disabled engine with setup steps, and one available
// engine with neither dial — every shape collectEngineRows branches on,
// none of which the fixture's default single-engine fallback offers.
func enginesTestList() []EngineInfo {
	return []EngineInfo{
		{
			Name:      "claude",
			Available: true,
			Models:    []ChoiceInfo{{ID: "", Label: "default"}, {ID: "opus", Label: "opus"}, {ID: "sonnet", Label: "sonnet"}},
			Efforts:   []ChoiceInfo{{ID: "", Label: "default"}, {ID: "low", Label: "low"}, {ID: "high", Label: "high"}},
			CanThink:  true,
		},
		{
			// Not installed, and still carrying its dials: what an engine
			// offers and whether this machine can run it are two facts,
			// and the port answers both for every engine.
			Name:      "codex",
			Available: false,
			Setup:     func(*words.Printer) []string { return []string{"install codex", "run codex login"} },
			Models:    []ChoiceInfo{{ID: "", Label: "default"}, {ID: "o3", Label: "o3"}, {ID: "o3-mini", Label: "o3-mini"}},
			Efforts:   []ChoiceInfo{{ID: "", Label: "default"}, {ID: "low", Label: "low"}, {ID: "high", Label: "high"}},
		},
		{
			Name:      "bare",
			Available: true,
		},
	}
}

func TestApplyEngineChoiceEveryRowKind(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// A disabled row opens the setup screen instead of applying anything.
	m2 := m.applyEngineChoice(engineRow{kind: rowEngine, engine: "codex", disabled: true, setup: []string{"a"}})
	if !m2.engines.showingSetup || m2.engines.setupEngine != "codex" {
		t.Fatalf("expected a disabled row to open setup for codex, got %+v", m2.engines)
	}

	m3 := m.applyEngineChoice(engineRow{kind: rowEngine, engine: "codex"})
	if m3.knobs.Engine != "codex" || m3.knobs.Model != "" {
		t.Errorf("rowEngine choice = %+v, want engine codex and model cleared", m3.knobs)
	}

	m4 := m.applyEngineChoice(engineRow{kind: rowModel, engine: "claude", id: "opus"})
	if m4.knobs.Engine != "claude" || m4.knobs.Model != "opus" {
		t.Errorf("rowModel choice = %+v", m4.knobs)
	}

	m5 := m.applyEngineChoice(engineRow{kind: rowEffort, id: "high"})
	if m5.knobs.Effort != "high" {
		t.Errorf("rowEffort choice = %+v", m5.knobs)
	}

	m6 := m.applyEngineChoice(engineRow{kind: rowThinking, id: "on"})
	if m6.knobs.Thinking != "on" {
		t.Errorf("rowThinking choice = %+v", m6.knobs)
	}
}

// TestKnobChipDefaultsEngineName is a knob set with something chosen but
// no engine named — the model, effort or thinking dial can be turned
// without ever visiting the engine row — where knobChip falls back to
// "claude" rather than leaving the chip's first word blank.
func TestKnobChipDefaultsEngineName(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.knobs = Knobs{Model: "opus"}

	got := m.knobChip()
	if !strings.Contains(got, "claude") {
		t.Errorf("knobChip() = %q, want it to default the engine name to claude", got)
	}
}

// TestEnginesKeyUnmatchedKey is a keystroke that names none of enginesKey's
// four verbs, which the switch answers with a no-op rather than a panic.
func TestEnginesKeyUnmatchedKey(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openEngines()
	before := m.engines.sel
	m2raw, cmd := m.enginesKey(press("z"))

	m2 := asModel(t, m2raw)
	if cmd != nil || m2.engines.sel != before || m2.screen != screenEngines {
		t.Errorf("expected an unmatched key to change nothing, got sel=%d screen=%v cmd=%v", m2.engines.sel, m2.screen, cmd)
	}
}

func TestSetOpt(t *testing.T) {
	// The fixture's settings port does not round-trip Model(), so setOpt is
	// checked the way applySetting reports it: through the band.
	m, _ := testModel(t, 100, 30)
	m2 := m.setOpt("model", "opus")
	wantBand(t, m2, "opus")
}

func TestAbandonEnginesFromStart(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenStart
	m = m.openEngines()

	m2 := m.abandonEngines()
	if m2.screen != screenStart {
		t.Errorf("abandonEngines from the start dialog = %v, want screenStart", m2.screen)
	}
}

func TestEnginesKeyShowingSetup(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.engines.showingSetup, m.engines.setupEngine = true, "codex"

	// Any other key does nothing while setup steps are up.
	m2raw, _ := m.enginesKey(press("x"))

	m2 := asModel(t, m2raw)
	if !m2.engines.showingSetup {
		t.Errorf("expected an unrelated key to leave the setup screen up")
	}

	// Back closes it.
	m3raw, _ := m.enginesKey(press("esc"))

	m3 := asModel(t, m3raw)
	if m3.engines.showingSetup {
		t.Errorf("expected Back to close the setup screen")
	}
}

func TestEnginesKeyNavigationAndChoice(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = enginesTestList
	m = m.openEngines()

	rows := m.collectEngineRows()

	idxs := m.selectableEngineIndices(rows)
	if len(idxs) == 0 {
		t.Fatalf("expected at least one selectable row")
	}

	// Up from the first row wraps to the last.
	m.engines.sel = 0
	m2raw, _ := m.enginesKey(press("up"))

	m2 := asModel(t, m2raw)
	if m2.engines.sel != len(idxs)-1 {
		t.Errorf("Up from the first row = %d, want %d", m2.engines.sel, len(idxs)-1)
	}

	// Down from the last row wraps to the first.
	m.engines.sel = len(idxs) - 1
	m3raw, _ := m.enginesKey(press("down"))

	m3 := asModel(t, m3raw)
	if m3.engines.sel != 0 {
		t.Errorf("Down from the last row = %d, want 0", m3.engines.sel)
	}

	// Open applies whatever row is under the cursor.
	m.engines.sel = 0
	m4raw, _ := m.enginesKey(press("enter"))

	m4 := asModel(t, m4raw)
	if m4.knobs.Engine == "" && !m4.engines.showingSetup {
		t.Errorf("expected Open to either apply a choice or open setup")
	}

	// Back leaves the screen and reports the dials in the band.
	m5raw, _ := m.enginesKey(press("esc"))

	m5 := asModel(t, m5raw)
	if m5.screen == screenEngines {
		t.Errorf("expected Back to leave the engines screen")
	}
}

// TestCollectEngineRowsNoDialsSections is the "bare" engine from
// enginesTestList: available, but with no effort dial and no thinking
// mode, which are the two header-only sections collectEngineRows draws in
// their place.
func TestCollectEngineRowsNoDialsSections(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = enginesTestList
	m.knobs.Engine = "bare"

	rows := m.collectEngineRows()

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
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = func() []EngineInfo {
		return []EngineInfo{{
			Name:      "zeta",
			Available: true,
			Models:    []ChoiceInfo{{ID: "zeta/one", Label: "one"}, {ID: "zeta/two", Label: "two"}},
			Efforts:   []ChoiceInfo{{ID: "high", Label: "high"}},
		}}
	}
	m.knobs.Engine = "zeta"
	m = m.foldEngine("zeta", true)

	var engines, models []string

	for _, r := range m.collectEngineRows() {
		switch r.kind {
		case rowEngine:
			engines = append(engines, r.engine)
		case rowModel:
			models = append(models, r.id)
		case rowHeader, rowEffort, rowThinking:
		}
	}

	if !slices.Equal(engines, []string{"zeta"}) {
		t.Errorf("the screen offers %v, want only what the port answered", engines)
	}

	if !slices.Equal(models, []string{"zeta/one", "zeta/two"}) {
		t.Errorf("the screen offers the models %v, want zeta's own", models)
	}
}

// TestAWindowWithNoEnginesPortInventsNone. Saying nothing is the only honest
// answer this package has: it may not name internal/engine, so any table it
// carried would be a copy waiting to drift from the one the build runs.
func TestAWindowWithNoEnginesPortInventsNone(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = nil

	for _, r := range m.collectEngineRows() {
		if r.kind == rowEngine || r.kind == rowModel {
			t.Errorf("a window with no engines port offered %q", r.engine+r.id)
		}
	}
}
