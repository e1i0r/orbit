package ui

// The one table of engines, seen from the screens that draw dials off it.
//
// Every test here hands the window an engine no part of this package could
// have guessed at. That is the whole point: the compose form used to carry
// a table of its own, and it offered opencode a model called llama-3.3,
// which no opencode has ever answered to — a task composed with it was a
// task that could not run.

import (
	"slices"
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

func TestTheComposeDialsAreTheEnginesOwn(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = zetaEngines
	m = m.openCompose()

	if got := m.compose.engines; !slices.Equal(got, []string{"zeta"}) {
		t.Errorf("the engine dial offers %v, want only what the port answered", got)
	}

	if got := m.compose.models; !slices.Equal(got, []string{"zeta/one", "zeta/two"}) {
		t.Errorf("the model dial holds %v, want zeta's own ids", got)
	}

	if got := m.compose.efforts; !slices.Equal(got, []string{"brisk"}) {
		t.Errorf("the effort dial offers %v, want zeta's own", got)
	}

	// The id is what a task is composed with and the label is what is
	// drawn and measured. They are two strings for opencode, whose ids are
	// provider-qualified, and drawing the id would put the provider in
	// front of every position on the dial.
	if got := m.compose.modelLabel(0); got != "one" {
		t.Errorf("the model dial draws %q, want the label the port gave it", got)
	}

	if got := m.compose.chosenModel(); got != "zeta/one" {
		t.Errorf("the form composes with %q, want the id behind the label", got)
	}
}

// A click on an engine pill and the arrow keys are the same gesture, and the
// click used to carry its own copy of it: it moved the engine and left the
// effort dial showing the engine before it.
func TestChoosingAComposeEngineRefillsBothDialsThatHangOffIt(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()

	for i, want := range m.compose.engines {
		m = m.chooseComposeEngine(i)
		if got := m.compose.chosenEngine(); got != want {
			t.Fatalf("chooseComposeEngine(%d) chose %q, want %q", i, got, want)
		}

		models, _ := m.modelsFor(want)
		if !slices.Equal(m.compose.models, models) {
			t.Errorf("%s's model dial holds %v, want %v", want, m.compose.models, models)
		}

		efforts, _ := m.effortsFor(want)
		if !slices.Equal(m.compose.efforts, efforts) {
			t.Errorf("%s's effort dial holds %v, want %v", want, m.compose.efforts, efforts)
		}
	}
}

// A window with no engines port has nothing to say about engines, and says
// nothing. Inventing a table here is what put four of them in this package.
func TestAComposeFormWithNoEnginesPortInventsNone(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = nil
	m = m.openCompose()

	if n := len(m.compose.engines) + len(m.compose.models) + len(m.compose.efforts); n != 0 {
		t.Errorf("the form offers %d choices with no engines port", n)
	}

	if got := m.compose.chosenEngine() + m.compose.chosenModel() + m.compose.chosenEffort(); got != "" {
		t.Errorf("the form composes with %q with no engines port, want nothing", got)
	}
}
