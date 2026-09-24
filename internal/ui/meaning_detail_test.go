package ui

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// TestQuestionMarkOnATasksScreenSaysWhatTheKeyDoesThere. p, E and k are
// taken by the task screen for jobs of their own, and ? answered with what
// they do on the board: pause, the engines screen, up.
func TestQuestionMarkOnATasksScreenSaysWhatTheKeyDoesThere(t *testing.T) {
	m, _ := openIn(t, words.For("en"), "ACME-2698", fixtureEntries(), wideDiff())

	for k, want := range map[string]string{
		"P": "pull request",
		"J": "merges",
		"U": "up to date",
	} {
		next, _ := m.armTip().tipKey(keystroke(k))
		if got := asModel(t, next).message; !strings.Contains(got, want) {
			t.Errorf("? %s on a task's screen said %q, want it to be about %s", k, got, want)
		}
	}

	m.screen = screenList

	next, _ := m.armTip().tipKey(keystroke("p"))
	if got := asModel(t, next).message; !strings.Contains(got, "stop") {
		t.Errorf("? p on the board said %q, want the pause", got)
	}
}
