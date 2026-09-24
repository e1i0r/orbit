package ui

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// TestMergeAndUpdateAreReachedByTheirKeysOnATasksScreen. The deliver block
// drew M merge PR and u update PR, and neither key got there: M is the
// board menu, taken first, and u is the prompt tab's letter, taken before
// any verb. Merge is J and update is U, and u still opens the prompt.
func TestMergeAndUpdateAreReachedByTheirKeysOnATasksScreen(t *testing.T) {
	m, _ := openIn(t, words.For("en"), "ACME-2698", fixtureEntries(), wideDiff())

	next, cmd := m.detailKey(keystroke("J"))
	if got := asModel(t, next).message; cmd == nil || !strings.Contains(got, "merging pull request") {
		t.Errorf("J on a task's screen said %q, want the merge under way", got)
	}

	// This task has no checkout, so the answer is that there is nothing to
	// work in: what matters is that update PR is the verb that answered.
	next, _ = m.detailKey(keystroke("U"))
	if got := asModel(t, next); got.tab == tabPrompt || !strings.Contains(got.message, "ACME-2698") {
		t.Errorf("U on a task's screen opened tab %v and said %q, want update PR's answer", got.tab, got.message)
	}

	next, _ = m.detailKey(keystroke("u"))
	if got := asModel(t, next).tab; got != tabPrompt {
		t.Errorf("u on a task's screen opened tab %v, want the prompt", got)
	}
}
