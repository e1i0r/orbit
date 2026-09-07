package ui

// The one table of engines, seen from the screens that draw dials off it.
//
// Every test here hands the window an engine no part of this package could
// have guessed at. That is the whole point: a table of a screen's own
// offered opencode a model called llama-3.3, which no opencode has ever
// answered to — a phase built with it is a phase that cannot run.

import (
	"slices"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/roster"
)

// zetaEngines is one engine this build has never heard of, with both dials
// and with the "whatever the engine is configured for" choice the port puts
// in front of them.
func zetaEngines() []roster.Engine {
	return []roster.Engine{{
		Name:      "zeta",
		Available: true,
		Models:    []roster.Choice{{ID: "", Label: "default"}, {ID: "zeta/one", Label: "one"}, {ID: "zeta/two", Label: "two"}},
		Efforts:   []roster.Choice{{ID: "", Label: "default"}, {ID: "brisk", Label: "brisk"}},
		CanThink:  true,
	}}
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
