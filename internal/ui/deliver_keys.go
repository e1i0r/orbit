package ui

// The keys that act on a task's pull request, answered the same on the
// board and on a task's screen: the task is the one being read, or the one
// under the cursor.
//
// They were letters the task screen matched for itself — p, D, R — which
// on the board are pause, delete and the repositories. A key means one
// thing everywhere, so each is a binding of its own now, a capital no
// other verb has.

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// isDeliver is whether a keystroke is one of them.
func (m Model) isDeliver(k fmt.Stringer) bool {
	_, ok := m.deliverVerb(k)

	return ok
}

// deliverKey does what the key stands for.
func (m Model) deliverKey(k fmt.Stringer) (tea.Model, tea.Cmd) {
	do, ok := m.deliverVerb(k)
	if !ok {
		return m, nil
	}

	return do()
}

// deliverVerb is the gesture behind one of the keys.
func (m Model) deliverVerb(k fmt.Stringer) (func() (tea.Model, tea.Cmd), bool) {
	for _, v := range []struct {
		b  key.Binding
		do func() (tea.Model, tea.Cmd)
	}{
		{m.keys.CreatePR, m.deliverPR},
		{m.keys.UpdatePR, m.updatePRBranch},
		{m.keys.MergePR, m.mergePR},
		{m.keys.ClosePR, m.closePR},
		{m.keys.FixChecks, m.fixChecks},
		{m.keys.MoreTests, m.addMoreTests},
		{m.keys.Resolve, m.resolveComments},
		{m.keys.Review, m.reviewPR},
	} {
		if key.Matches(k, v.b) {
			return v.do, true
		}
	}

	return nil, false
}
