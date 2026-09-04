package ui

import (
	"io"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/view"
)

func TestDeliverActionsNoTask(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = nil
	m.detail = ""

	// deliverPR with no selected task
	nextM, cmd := m.deliverPR()
	if nextM == nil || cmd != nil {
		t.Errorf("deliverPR with no task = (%v, %v), want say message", nextM, cmd)
	}

	// fixChecks with no selected task
	nextM, cmd = m.fixChecks()
	if nextM == nil || cmd != nil {
		t.Errorf("fixChecks with no task = (%v, %v), want say message", nextM, cmd)
	}

	// addMoreTests with no selected task
	nextM, cmd = m.addMoreTests()
	if nextM == nil || cmd != nil {
		t.Errorf("addMoreTests with no task = (%v, %v), want say message", nextM, cmd)
	}

	// updatePRBranch with no selected task
	nextM, cmd = m.updatePRBranch()
	if nextM == nil || cmd != nil {
		t.Errorf("updatePRBranch with no task = (%v, %v), want say message", nextM, cmd)
	}

	// mergePR with no selected task
	nextM, cmd = m.mergePR()
	if nextM == nil || cmd != nil {
		t.Errorf("mergePR with no task = (%v, %v), want say message", nextM, cmd)
	}

	// closePR with no selected task
	nextM, cmd = m.closePR()
	if nextM == nil || cmd != nil {
		t.Errorf("closePR with no task = (%v, %v), want say message", nextM, cmd)
	}
}

func TestDeliverActionsSelectedTask(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "ACME-100", Repo: "acme", RepoPath: "/path/to/acme", Band: view.Done},
	}
	m.detail = "ACME-100"
	m.screen = screenDetail

	nextM, cmd := m.deliverPR()
	if nextM == nil || cmd == nil {
		t.Errorf("deliverPR on selected task should return non-nil cmd")
	}

	nextM, cmd = m.fixChecks()
	if nextM == nil || cmd == nil {
		t.Errorf("fixChecks on selected task should return non-nil cmd")
	}

	nextM, cmd = m.addMoreTests()
	if nextM == nil || cmd == nil {
		t.Errorf("addMoreTests on selected task should return non-nil cmd")
	}

	nextM, cmd = m.updatePRBranch()
	if nextM == nil || cmd == nil {
		t.Errorf("updatePRBranch on selected task should return non-nil cmd")
	}

	nextM, cmd = m.mergePR()
	if nextM == nil || cmd == nil {
		t.Errorf("mergePR on selected task should return non-nil cmd")
	}

	nextM, cmd = m.closePR()
	if nextM == nil || cmd == nil {
		t.Errorf("closePR on selected task should return non-nil cmd")
	}
}

// TestAVerbSaysWhatWasAskedForWhenItLands. Fix checks and more tests are
// both `orbit note` underneath, and what the band said when the command came
// back was "note finished": the name of a command the reader never pressed,
// naming no task, in front of a screen whose caption said FIX CHECKS.
func TestAVerbSaysWhatWasAskedForWhenItLands(t *testing.T) {
	for _, c := range []struct {
		press func(Model) (tea.Model, tea.Cmd)
		want  string
	}{
		{Model.fixChecks, "checks pass again"},
		{Model.addMoreTests, "asked for more tests"},
	} {
		m, _ := testModel(t, 100, 30)
		m.board.Tasks = []view.Task{{ID: "ACME-100", Repo: "acme", RepoPath: "/path/to/acme", Band: view.Done}}
		m.detail, m.screen = "ACME-100", screenDetail
		m.opts.Do = func(string, []string, io.Writer) error { return nil }

		next, cmd := c.press(m)

		running := asModel(t, next)
		wantBand(t, running, c.want)

		// And the sentence is still the reader's once the note is filed.
		for _, one := range commandsIn(t, cmd) {
			msg, ok := one().(commandMsg)
			if !ok {
				continue
			}

			landed, _ := running.Update(msg)
			wantBand(t, asModel(t, landed), c.want)
		}
	}
}
