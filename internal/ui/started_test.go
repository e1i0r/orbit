package ui

import (
	"strings"
	"testing"
)

// TestAStartThatWaitsSaysSo rather than naming a process that is not there:
// a start the queue holds answers a pid of zero.
func TestAStartThatWaitsSaysSo(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	if said := m.startedSaid(startedMsg{ID: "Q-1"}); !strings.Contains(said, "queue") {
		t.Errorf("a start left waiting says %q, want that it is waiting in the queue", said)
	}
}
