package ui

// A delivery nothing is carrying, said as broken while the window is open.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/ui/panes"
)

// settle asks, on the task view's clock, whether the verb the task is
// waiting on still has a window carrying it, and has it answered as broken
// when it has not. The next reading of the record then says so.
//
// Asked while the window is open and not only when one opens. ORB-121's
// CREATE PR lost its window thirteen seconds in, and the window that
// replaced it said "the supervisor is working on it" for twenty-seven
// minutes with nothing running behind it. A verb that has died is said to
// have died on the next tick.
//
// Nothing is asked while nothing is out, which is almost always: the
// question is a read of the record and a signal 0, and the tick is twice a
// second.
func (m Model) settle() tea.Cmd {
	port := m.opts.SettleDeliveries
	if port == nil {
		return nil
	}

	if _, out := panes.Waiting(panes.Env{Entries: m.entries}); !out {
		return nil
	}

	t := m.subject()

	return func() tea.Msg {
		if err := port(t); err != nil {
			logger.Error("ui/settle", "%s: the delivery nothing carries could not be closed: %v", t.ID, err)
		}

		return nil
	}
}
