package ui

import (
	"testing"

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
}
