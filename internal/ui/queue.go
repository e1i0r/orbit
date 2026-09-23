package ui

import "github.com/e1i0r/orbit/internal/view"

// placeInQueue is where a waiting task stands in line: one for the next to
// start. The record is the queue, and the board carries every task, so the
// line is counted from the rows: the ones asked for earlier are ahead.
func (m Model) placeInQueue(t view.Task) int {
	place := 1

	for _, o := range m.board.Tasks {
		if o.ID != t.ID && o.Reason.Key == view.ReasonQueued && o.Since.Before(t.Since) {
			place++
		}
	}

	return place
}
