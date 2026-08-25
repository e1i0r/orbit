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
