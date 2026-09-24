package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestANoSaysTheTaskIsLeftAsItWas. Answering no to cancel, requeue or
// skip took the question down and said nothing, so the reader could not
// tell a no that was heard from a key that was lost.
func TestANoSaysTheTaskIsLeftAsItWas(t *testing.T) {
	for _, c := range []confirm{confirmCancel, confirmRequeue, confirmSkip} {
		m, _ := testModel(t, 100, 30)
		m.confirm, m.confirmID = c, "ACME-2662"

		next, cmd := m.confirmKey(tea.KeyPressMsg{Code: 'n', Text: "n"})
		if got := asModel(t, next).message; cmd != nil || !strings.Contains(got, "ACME-2662 is left as it was") {
			t.Errorf("n to question %v said %q, want the task left as it was", c, got)
		}
	}
}
