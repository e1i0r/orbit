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
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// applyBoard takes the next board and, with it, whatever the new board asks
// the window to do — including turn the spinner.
//
// The frame clock is started here rather than only by a gesture because a
// run started from a terminal is not a gesture: the first this window hears
// of it is a board arriving with a live task on it, and if that board does
// not ask for a frame nothing on screen ever moves.
func (m Model) applyBoard(msg boardMsg) (tea.Model, tea.Cmd) {
	next, cmd := m.takeBoard(msg)
	next, frame := next.nextFrame()

	return next, tea.Batch(cmd, frame)
}

// takeBoard is applyBoard's decision about the board itself, kept apart from
// the animation so that neither has to be read to change the other.
//
// A boardMsg with a zero ReadAt is a read that failed or an enumeration that
// found nothing to say. The board already on screen is kept in both cases: a
// window that blanks because one stat failed has thrown away the answer it
// spent the last half-second holding.
func (m Model) takeBoard(msg boardMsg) (Model, tea.Cmd) {
	if msg.Board.ReadAt.IsZero() {
		if len(msg.Board.Errs) > 0 {
			return m.say(m.errSaid(msg.Board.Errs[0])), nil
		}

		return m, nil
	}

	first := !m.seen
	m.board, m.seen = msg.Board, true
	m.totals = phaseTotals(msg.Board.Tasks)

	m = m.stillTaken().replan().clampCursor()
	if first {
		m.cursor = m.firstTask()
		m = m.follow()
	}

	m = m.selectPending()
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
		if nextM, cmd := m.autoStartNext(); cmd != nil {
			return nextM, cmd
		}

		if nextM, cmd := m.autoSuperviseNeedsYou(); cmd != nil {
			return nextM, cmd
		}

		return m, nil
	}

	m.notified = true

	m = m.say(m.opts.Words.P("msg.entered", len(msg.Changed.Entered), "{n} task needs you", "{n} tasks need you"))
	if nextM, cmd := m.autoSuperviseNeedsYou(); cmd != nil {
		return nextM, tea.Batch(tea.Raw("\a"), cmd)
	}

	if nextM, cmd := m.autoStartNext(); cmd != nil {
		return nextM, tea.Batch(tea.Raw("\a"), cmd)
	}

	return m, tea.Raw("\a")
}

// autoStartNext picks the first task in To Do and starts it under autopilot.
func (m Model) autoStartNext() (Model, tea.Cmd) {
	if !m.autopilotOn() || m.opts.Start == nil {
		return m, nil
	}

	waiting := board.Unreads(m.board)
	if m.atUnreadCap(len(waiting)) {
		return m, nil
	}

	flowName := flow.Default
	if m.opts.Settings != nil && m.opts.Settings.Flow() != "" {
		flowName = m.opts.Settings.Flow()
	}

	for _, t := range m.board.Tasks {
		if view.BandOf(t) == view.ToDo && !m.taken[t.ID] {
			m = m.took(t.ID, true)
			return m, start(m.opts.Start, t, flowName, len(waiting))
		}
	}

	return m, nil
}

// autoSuperviseNeedsYou inspects and remediates tasks needing attention under autopilot.
func (m Model) autoSuperviseNeedsYou() (Model, tea.Cmd) {
	if !m.autopilotOn() || m.opts.AutoSupervise == nil || m.supervisorBusy {
		return m, nil
	}

	var needing []string

	for _, t := range m.board.Tasks {
		if view.BandOf(t) == view.NeedsYou && !m.taken[t.ID+"-sup"] {
			needing = append(needing, t.ID)
		}
	}

	if len(needing) == 0 {
		return m, nil
	}

	for _, id := range needing {
		m = m.took(id+"-sup", true)
	}

	m.supervisorBusy = true

	eng := m.dialEngine(m.knobs.Engine)

	cmd := func() tea.Msg {
		ans, err := m.opts.AutoSupervise(eng, needing)
		return supervisorReplyMsg{Text: ans, Err: err}
	}
	m, frame := m.say(m.opts.Words.T("supervisor.acting", "supervisor is autonomously inspecting {n} task(s)...", about("n", strconv.Itoa(len(needing))))).nextFrame()

	return m, tea.Batch(cmd, frame)
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
