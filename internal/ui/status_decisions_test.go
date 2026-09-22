package ui

// The one sentence the window says about the decision engine, and the three
// times it says nothing.
//
// It is a test of its own because the failure it stands for left no trace
// anywhere else: `decisions` was set, no key was in the environment, and a
// task ran the whole way to its gate and waited for a person. Every screen
// looked exactly as it looks when the engine is working. Both halves of the
// answer were readable the whole time.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// deciding is the fixture with the setting written through the real table
// and the environment's half stood in for.
func deciding(t *testing.T, state, missing string) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)

	file := newSettingsFile()
	if _, err := file.tbl.Choose(words.For("en"), "decisions", state); err != nil {
		t.Fatalf("set decisions to %q: %v", state, err)
	}

	file.missing = missing
	m.opts.Settings = file

	return m
}

// said is whether the status line carries the warning, at a width where
// every segment fits.
func said(m Model) bool {
	return strings.Contains(m.statusLine(200), "no TYPESAFE_API_KEY in the environment")
}

// TestTheWindowSaysWhenTheDecisionEngineCannotWork.
func TestTheWindowSaysWhenTheDecisionEngineCannotWork(t *testing.T) {
	for _, c := range []struct {
		state, missing string
		want           bool
		why            string
	}{
		// The failure this exists for, in both states that ask the engine
		// anything. Shadow counts: it writes down a decision per gate, and
		// a reader who turned it on to collect them gets an empty record
		// and no reason for it.
		{"on", "TYPESAFE_API_KEY", true, "decisions on with no key"},
		{"shadow", "TYPESAFE_API_KEY", true, "decisions in shadow with no key"},

		// Nothing to say. Off is what it ships as, and a reader who has not
		// turned it on is not waiting for it — the warning would be the
		// window nagging about a feature nobody asked for.
		{"off", "TYPESAFE_API_KEY", false, "decisions off and no key"},

		// Working. The quota field is silent about an engine paid per token
		// for the same reason: there is no action behind the sentence.
		{"on", "", false, "decisions on with a key"},
		{"shadow", "", false, "decisions in shadow with a key"},
		{"off", "", false, "decisions off with a key"},
	} {
		if got := said(deciding(t, c.state, c.missing)); got != c.want {
			t.Errorf("%s: the status line warns = %v, want %v\n  line: %q",
				c.why, got, c.want, deciding(t, c.state, c.missing).statusLine(200))
		}
	}
}

// TestTheWarningSurvivesANarrowTerminal is the half that is easy to write
// and easy to lose.
//
// The status line gives up fields from the right as the terminal narrows,
// and Elio works at 100 columns. A warning appended after the quota reading
// would be correct at 200 columns, tested at 200 columns, and absent on the
// one terminal it was written for.
func TestTheWarningSurvivesANarrowTerminal(t *testing.T) {
	m := deciding(t, "on", "TYPESAFE_API_KEY")

	for _, w := range []int{100, 80, 60} {
		if !strings.Contains(m.statusLine(w), "TYPESAFE_API_KEY") {
			t.Errorf("at %d columns the warning is gone: %q", w, m.statusLine(w))
		}
	}
}

// TestAWindowWithNoSettingsPortSaysNothing: a rendering test hands the
// window no settings file at all, and a nil port is not a missing key.
func TestAWindowWithNoSettingsPortSaysNothing(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Settings = nil

	if said(m) {
		t.Errorf("a window with no settings port warns about a key: %q", m.statusLine(200))
	}
}
