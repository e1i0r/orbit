package ui

// The affordances a window answers with: what the engine under the cursor is
// asked about, and which verbs a task offers on the screen it is drawn on.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

func TestTheEngineIsAskedAboutOneTaskAtATime(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m.opts.CanResume = func(engine string) bool { return engine == "claude" }
	for engine, want := range map[string]bool{"claude": true, "codex": false, "": false} {
		if got := m.conditions(view.Task{ID: "ACME-1", Engine: engine}).CanResume; got != want {
			t.Errorf("a task on %q is told CanResume=%v, want %v", engine, got, want)
		}
	}

	m.opts.CanResume = nil
	if m.conditions(view.Task{ID: "ACME-1", Engine: "claude"}).CanResume {
		t.Error("a window with no way to ask about engines answered yes")
	}
}

// TestAskIsListedAndRefused is the tool being honest about its own gap in
// the same voice it uses about an engine's. The verb exists in the menu, it
// is refused, and the reason says what to do instead.
