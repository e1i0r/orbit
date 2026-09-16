package ui

// What a row says about a task whose engine has nothing left.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestARowNamesWhoCouldTakeTheTask is what makes the "needs you" row worth
// stopping for: told only that claude ran out, the reader has to go and find
// out who else could take it.
func TestARowNamesWhoCouldTakeTheTask(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	word, _ := m.stateWord(view.Task{Reason: view.Reason{
		Key: view.ReasonNeedsEngine,
		Args: []view.Arg{
			{Name: "engine", Value: "claude"},
			{Name: "phase", Value: "implement"},
			{Name: "engines", Value: "codex, opencode"},
		},
	}})

	for _, want := range []string{"claude", "implement", "codex", "opencode"} {
		if !strings.Contains(word, want) {
			t.Errorf("the row does not mention %q: %q", want, word)
		}
	}
}

// TestARowWithNoEngineSaysUntilWhen. Allowances come back, so an hour is an
// answer and "abandoned" is not — and a row with no hour to give says that
// rather than naming one it made up.
func TestARowWithNoEngineSaysUntilWhen(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	with := view.Reason{Key: view.ReasonNoEngine, Args: []view.Arg{
		{Name: "engine", Value: "claude"},
		{Name: "phase", Value: "implement"},
		{Name: "back", Value: "1h30m0s"},
	}}

	// "1h" and not "1h30m0s": the row spells a wait the way the engine's own
	// "back in" row spells one, so the two rows about waiting for an
	// allowance wait in the same words. The exact duration stays in the
	// record, where whatever reads it next will not have to parse prose.
	word, _ := m.stateWord(view.Task{Reason: with})
	if !strings.Contains(word, "1h") {
		t.Errorf("the row does not say until when: %q", word)
	}

	without := with
	without.Args = with.Args[:2]

	word, _ = m.stateWord(view.Task{Reason: without})
	if strings.Contains(word, "another") {
		t.Errorf("the row invented an hour nothing knows: %q", word)
	}

	if !strings.Contains(word, "no engine") {
		t.Errorf("the row does not say there is no engine: %q", word)
	}
}
