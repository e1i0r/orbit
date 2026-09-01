package ui

// enginesdraw_coverage_test.go is enginesRows's setup-screen branch and
// hitEngines's out-of-body and past-the-last-row answers, none of which
// engines_coverage_test.go's lifecycle walk reaches.

import (
	"strings"
	"testing"
)

func TestEnginesRowsShowingSetup(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = enginesTestList
	m = m.openEngines()
	m.engines.showingSetup, m.engines.setupEngine = true, "codex"

	rows := m.enginesRows(m.frame.Body.H, m.frame.Body.W)

	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "codex") {
		t.Errorf("expected the setup screen to name codex")
	}

	if !strings.Contains(joined, "install codex") {
		t.Errorf("expected codex's own setup steps listed")
	}
}

func TestEnginesRowsSelectedAndDisabled(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = enginesTestList
	m.knobs.Engine, m.knobs.Model = "claude", "opus"
	m = m.openEngines()

	rows := m.enginesRows(m.frame.Body.H, m.frame.Body.W)

	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "setup required") {
		t.Errorf("expected the disabled codex row to say setup is required")
	}
}

func TestHitEnginesOutsideBody(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m = m.openEngines()
	if got := m.hitEngines(10, 0); got.Kind != TargetNone {
		t.Errorf("hitEngines outside the body = %+v, want the zero Target", got)
	}
}

func TestHitEnginesPastLastRow(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m = m.openEngines()
	if got := m.hitEngines(10, m.frame.Body.Y+999); got.Kind != TargetNone {
		t.Errorf("hitEngines past the last row = %+v, want the zero Target", got)
	}
}

func TestHitEnginesSkipsHeaders(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = enginesTestList
	m = m.openEngines()

	// The very first body row is the "Engine & Model" section header,
	// which hitEngines steps over rather than treating as a target.
	if got := m.hitEngines(10, m.frame.Body.Y+4); got.Kind != TargetNone {
		t.Errorf("hitEngines on a header row = %+v, want the zero Target", got)
	}
}

// The screen where the engine is chosen says what is left of each one, and
// says it beside the engine rather than beside a model: a window belongs to
// the engine, and a model with a percentage next to it would be claiming a
// cap of its own.
func TestTheEnginePickerCarriesEachEnginesWindows(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openEngines()

	m.opts.Quota = func(engine string) QuotaReading {
		if engine != "claude" {
			return QuotaReading{Engine: engine, Sourced: false}
		}

		return QuotaReading{
			Engine:  engine,
			Sourced: true,
			Windows: []QuotaWindow{{Label: "5h", Pct: 2}, {Label: "7d", Pct: 77}},
		}
	}

	rows := m.enginesRows(30, 100)

	var claude, model string

	for _, row := range rows {
		if strings.Contains(row, "claude") {
			claude = row
		}

		if strings.Contains(row, "sonnet") {
			model = row
		}
	}

	if claude == "" {
		t.Fatal("no claude row on the engine screen")
	}

	for _, want := range []string{"2% 5h", "77% 7d"} {
		if !strings.Contains(claude, want) {
			t.Errorf("engine row %q does not carry %q", claude, want)
		}
	}

	if model != "" && strings.Contains(model, "%") {
		t.Errorf("model row %q carries a percentage, which is the engine's and not its own", model)
	}

	// An engine that is not installed carries nothing: that row is about the
	// setup it still needs, and a quota beside it is an answer to a question
	// nobody standing there is asking.
	for _, row := range rows {
		if strings.Contains(row, "setup required") && strings.Contains(row, "%") {
			t.Errorf("row %q carries a quota for an engine that is not installed", row)
		}
	}

	// An engine nobody can read says so where it is chosen, for the same
	// reason it says so on the status line: silence there is read as zero.
	unsourced := m.engineQuota(engineRow{kind: rowEngine, engine: "codex"})
	if !strings.Contains(unsourced, "no quota source") {
		t.Errorf("unsourced engine = %q, want it to say it has no source", unsourced)
	}

	// And one paid per token has no window to be at the end of.
	m.opts.Quota = func(engine string) QuotaReading {
		return QuotaReading{Engine: engine, Money: true}
	}

	if got := m.engineQuota(engineRow{kind: rowEngine, engine: "opencode"}); got != "" {
		t.Errorf("metered engine = %q, want nothing", got)
	}
}
