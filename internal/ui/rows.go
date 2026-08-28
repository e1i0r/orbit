package ui

// The body: which rows the board has, where the cursor is among them, and
// what one row of it says.
//
// A row here is a line of the body and not a task: the band headers and the
// blank line between two bands are rows too. That is what makes the cursor
// one index into one list — the alternative, a band index and a row index
// inside it, is two numbers that have to agree, and in the program this
// replaces they did not.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// row is one line of the body.
type row struct {
	band  view.Band
	head  bool      // a band header, which the cursor may rest on and open
	blank bool      // the line between two bands; the cursor steps over it
	n     int       // how many tasks the band holds, for a header
	task  view.Task // the task, for a row that is one
}

// rows is the body as a list of lines, in the order they are drawn.
//
// A band with nothing in it is not drawn at all — no header, no blank line.
// An empty band is a heading over nothing, and four of them is a screen that
// says nothing four times.
func (m Model) rows() []row {
	filter := strings.ToLower(strings.TrimSpace(m.filter))

	var out []row

	bands := view.Bands()
	if m.queueFilter != nil {
		bands = []view.Band{*m.queueFilter}
	}

	for _, b := range bands {
		tasks := m.tasksIn(b, filter)
		// The count over a band is the board's own, computed by
		// view.BandOf on the very tasks below it, so the number and the
		// rows are one answer. A filter is the one thing that separates
		// them, and then the count is what is shown rather than what
		// exists — because that is what the reader is looking at.
		count := m.board.Counts[b]
		if filter != "" || m.repoFilter != "" {
			count = len(tasks)
		}

		if count == 0 {
			continue
		}

		if len(out) > 0 {
			out = append(out, row{blank: true})
		}

		out = append(out, row{band: b, head: true, n: count})
		if !m.expanded[b] {
			continue
		}

		for _, t := range tasks {
			out = append(out, row{band: b, task: t})
		}
	}

	return out
}

// tasksIn is every task of one band the filter lets through, in the board's
// own order — which is stable across refreshes, so a cursor resting on a row
// stays on that row.
func (m Model) tasksIn(b view.Band, filter string) []view.Task {
	var out []view.Task

	for _, t := range m.board.Tasks {
		if view.BandOf(t) == b && matches(t, filter) && matchesRepo(t, m.repoFilter) {
			out = append(out, t)
		}
	}

	return out
}

// matchesRepo is whether the task belongs to the filtered repository.
func matchesRepo(t view.Task, repoFilter string) bool {
	if repoFilter == "" {
		return true
	}

	return strings.EqualFold(t.Repo, repoFilter)
}

// matches is the filter, over the three fields a reader would type: the
// repository, the id, and the words of the title.
func matches(t view.Task, filter string) bool {
	if filter == "" {
		return true
	}

	for _, field := range []string{t.Repo, t.ID, t.Title} {
		if strings.Contains(strings.ToLower(field), filter) {
			return true
		}
	}

	return false
}

// selected is the row under the cursor, and whether there is one.
func (m Model) selected() (row, bool) {
	rows := m.rows()
	if m.cursor < 0 || m.cursor >= len(rows) || rows[m.cursor].blank {
		return row{}, false
	}

	return rows[m.cursor], true
}

// firstTask is the first row that is a task rather than a header. It is
// where the cursor opens, because the first thing a reader wants is the
// first thing that needs them, not the heading above it.
func (m Model) firstTask() int {
	for i, r := range m.rows() {
		if !r.blank && !r.head {
			return i
		}
	}

	return 0
}

// move steps the cursor by n rows, stepping over the blanks and stopping at
// either end. It does not wrap: a list that wraps moves the eye to the other
// end of the screen for a keystroke that meant "next".
func (m Model) move(n int) Model {
	rows := m.rows()

	step := 1
	if n < 0 {
		step = -1
	}

	for range n * step {
		next := seek(rows, m.cursor, step)
		if next == m.cursor {
			break
		}

		m.cursor = next
	}

	return m.follow()
}

// moveTo puts the cursor on one row, or on the nearest one it may rest on.
func (m Model) moveTo(i int) Model {
	rows := m.rows()
	m.cursor = min(max(i, 0), max(len(rows)-1, 0))

	return m.clampCursor()
}

// seek is the next row in one direction the cursor may rest on, or from
// when there is none.
func seek(rows []row, from, step int) int {
	for i := from + step; i >= 0 && i < len(rows); i += step {
		if !rows[i].blank {
			return i
		}
	}

	return from
}

// clampCursor puts the cursor back on the list after the list changed under
// it — a refresh that finished three tasks, a filter that hid the row it was
// on. A cursor pointing past the end draws nothing and refuses every verb,
// which reads as a window that has stopped responding.
func (m Model) clampCursor() Model {
	rows := m.rows()
	if len(rows) == 0 {
		m.cursor, m.offset = 0, 0
		return m
	}

	m.cursor = min(max(m.cursor, 0), len(rows)-1)
	if rows[m.cursor].blank {
		m.cursor = seek(rows, m.cursor, -1)
	}

	return m.follow()
}

// follow scrolls the body just enough to keep the cursor on screen, and no
// further. The list does not centre the cursor: a body that scrolls on every
// keystroke costs a reader the one thing a list is for, which is that the
// rows stay where they were.
func (m Model) follow() Model {
	h := m.frame.Body.H

	rows := len(m.rows())
	if h <= 0 || rows <= h {
		m.offset = 0
		return m
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	// page and not h: when the list runs below the fold the body spends its
	// last row on saying so, and scrolling to h would park the cursor
	// underneath that sentence.
	if shown := page(h, rows, m.offset); m.cursor >= m.offset+shown {
		m.offset = min(m.cursor-shown+1, rows-h)
	}

	m.offset = min(max(m.offset, 0), rows-h)

	return m
}
