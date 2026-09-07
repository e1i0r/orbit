package ui

// The supervisor screen as the window reaches it: the key that opens it, the
// key that leaves it, and the reply landing back in the update loop. What the
// screen itself does is tested in internal/ui/supervisor.

import (
	"strings"
	"testing"
)

func TestSupervisorScreenOpenAndAbandon(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m = step(t, m, "S")
	if m.screen != screenSupervisor {
		t.Fatalf("m.screen = %v, want screenSupervisor", m.screen)
	}

	// Esc goes back to where it was opened from, which the screen hands
	// back rather than deciding.
	m = step(t, m, "esc")
	if m.screen != screenList {
		t.Errorf("m.screen = %v, want screenList after esc", m.screen)
	}
}

func TestSupervisorReplyMsgHandling(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openSupervisor()
	m.supervisorBusy = true

	m = next(t, m, supervisorReplyMsg{Text: "all tasks healthy"})
	if m.supervisorBusy {
		t.Error("the window is still waiting on an answer that has landed")
	}

	if !strings.Contains(m.message, "supervisor replied") {
		t.Errorf("the band says %q, want it to say the answer arrived", m.message)
	}
}
