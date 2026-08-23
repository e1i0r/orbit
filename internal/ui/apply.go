package ui

// What a message leaves behind: the board taken or refused, the geometry
// re-fitted, the sentence in the band, and the language swapped under
// everything built from it.
//
// It is a file of its own because ui.go is the model and the switch over
// messages, and the two together were over the 300-line ceiling once the
// task view's fields and its two messages arrived. The cut is the one the
// ceiling asks for: the switch says which of these runs, and each of these
// says what one message does.

import (
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/words"
)

// applyBoard takes the next board, or explains why there is not one.
//
// A boardMsg with a zero ReadAt is a read that failed or an enumeration that
// found nothing to say. The board already on screen is kept in both cases: a
// window that blanks because one stat failed has thrown away the answer it
// spent the last half-second holding.
func (m Model) applyBoard(msg boardMsg) (tea.Model, tea.Cmd) {
	if msg.Board.ReadAt.IsZero() {
		if len(msg.Board.Errs) > 0 {
			return m.say(msg.Board.Errs[0].Error()), nil
		}
		return m, nil
	}
	first := !m.seen
	m.board, m.seen = msg.Board, true
	m.totals = phaseTotals(msg.Board.Tasks)
	m = m.replan().clampCursor()
	if first {
		m.cursor = m.firstTask()
		m = m.follow()
	}
	// A read failure is said when the count of them changes and not on
	// every refresh, because the poll is twice a second and one unreadable
	// log would otherwise own the band for as long as it stayed unreadable.
	if n := len(msg.Board.Errs); n != m.errs {
		m.errs = n
		if n > 0 {
			m = m.say(m.opts.Words.P("msg.unreadable", n, "{n} record could not be read", "{n} records could not be read"))
		}
	}
	if first || len(msg.Changed.Entered) == 0 {
		return m, nil
	}
	m.notified = true
	m = m.say(m.opts.Words.P("msg.entered", len(msg.Changed.Entered), "{n} task needs you", "{n} tasks need you"))
	return m, tea.Raw("\a")
}

// resize takes the new geometry, or refuses it with both numbers.
func (m Model) resize(w, h int) Model {
	m.width, m.height = w, h
	f, err := layout.Fit(w, h)
	if err != nil {
		var narrow layout.TooNarrowError
		if !errors.As(err, &narrow) {
			narrow = layout.TooNarrowError{Need: layout.MinWidth, Got: w}
		}
		m.tooNarrow, m.narrow = true, narrow
		return m
	}
	m.tooNarrow, m.frame = false, f
	return m.replan().follow().syncPanes()
}

// replan re-plans the columns, from the whole board rather than from the
// rows currently shown: a column that changed width while a filter was being
// typed would move every field on screen between two keystrokes.
func (m Model) replan() Model {
	m.plan = layout.Columns(m.frame.Body.W-gutter, m.board.Tasks, m.opts.Words.Cells)
	return m
}

// say puts one sentence in the activity band. An empty sentence is not a
// sentence: it would blank the band, and a status area that goes blank reads
// as broken.
func (m Model) say(text string) Model {
	if text == "" {
		return m
	}
	m.message, m.messageAt = text, m.now
	return m
}

// language rewrites the language, and everything built from it. The key map
// is rebuilt because a binding carries its own help text, and the help
// overlay reads the bindings.
func (m Model) language(lang string) Model {
	if m.opts.Settings != nil {
		if err := m.opts.Settings.SetLanguage(lang); err != nil {
			return m.say(err.Error())
		}
	}
	m.opts.Words = words.For(lang)
	m.keys = NewKeys(m.opts.Words)
	return m.replan().syncPanes()
}
