package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/words"
)

// TestADeliverKeyMeansTheSameEverywhere. The task screen took p, D and R
// for the pull request, and on the board they are pause, delete and the
// repositories: a refusal that said "press p to stop it" was a lie on one
// of the two. The pull request's keys are capitals of their own now, and
// the board's letters mean on a task's screen what they mean on the board.
func TestADeliverKeyMeansTheSameEverywhere(t *testing.T) {
	task, _ := openIn(t, words.For("en"), "ACME-2698", fixtureEntries(), wideDiff())
	board := task
	board.screen, board.detail = screenList, ""
	board = onRow(t, board, "ACME-2698")

	for name, m := range map[string]Model{"the board": board, "a task's screen": task} {
		next, cmd := m.keyOn(t, "J")
		if got := next.message; cmd == nil || !strings.Contains(got, "merging pull request for ACME-2698") {
			t.Errorf("J on %s said %q, want the merge", name, got)
		}

		if next, _ := m.keyOn(t, "R"); next.screen != screenRepos {
			t.Errorf("R on %s opened screen %v, want the repositories", name, next.screen)
		}

		next, _ = m.keyOn(t, "D")
		if next.confirm != confirmDeleteTask && !strings.Contains(next.message, "delet") {
			t.Errorf("D on %s left confirm %v and said %q, want the delete", name, next.confirm, next.message)
		}

		if next, _ := m.keyOn(t, "p"); strings.Contains(next.message, "pull request") {
			t.Errorf("p on %s said %q, want the pause", name, next.message)
		}
	}
}

// keyOn presses one key on whichever screen the model is on.
func (m Model) keyOn(t *testing.T, k string) (Model, tea.Cmd) {
	t.Helper()

	if m.screen == screenDetail {
		next, cmd := m.detailKey(keystroke(k))

		return asModel(t, next), cmd
	}

	next, cmd := m.listKey(keystroke(k))

	return asModel(t, next), cmd
}
