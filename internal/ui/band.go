package ui

// The activity band: the one region of the screen that never comes back
// empty.
//
// It is a separate file from header.go because it is a separate decision.
// The header states what does not change while the window is open; the band
// answers "what is happening right now", and its whole design is the order
// in which it prefers four different answers to that question. The sentences
// it says about a message that has just arrived are here too, next to the
// order that decides whether the reader ever sees them.

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// bandLine is the activity band, and it never comes back empty.
//
// The order is what makes that true: a message owns it while it is fresh,
// then whatever is running owns it, and when nothing runs it says so. A
// status area that goes blank reads as broken — that is the single most
// valuable thing the program this replaces taught, because that is exactly
// how it read to the person who reported it.
func (m Model) bandLine(w int) string {
	switch {
	case m.filtering:
		return fit(" "+m.filterLine(), w)
	case m.confirm == confirmCancel:
		return fit(" "+Paint(Warn).Render(m.opts.Words.T("msg.confirm_cancel",
			"cancel {id}? press y to confirm, anything else to leave it running",
			about("id", m.confirmID))), w)
	case m.message != "" && m.now.Sub(m.messageAt) < messageLife:
		return fit(" "+Paint(Accent).Render(m.message), w)
	case m.filter != "":
		// A filter that is applied but no longer being typed. It sits below
		// the message and above what is running, because it qualifies the
		// list rather than reporting an event: while one is on, every count
		// on the screen is smaller than the board's own and the band is the
		// only place that says so.
		return fit(" "+m.filterLine(), w)
	}
	for _, t := range m.board.Tasks {
		if view.BandOf(t) == view.Running {
			return fit(" "+m.runningLine(t), w)
		}
	}
	return fit(" "+Paint(Dim).Render(m.idleLine()), w)
}

// filterLine is what is being typed, and how much of the board it is
// hiding. Saying the second half is the rule the plan states as "say it when
// you show less than you have": a filter is the one thing on this screen
// that can hide a task the reader is certain they wrote.
//
// It has two forms because a filter has two lives. While it is being typed
// the reader is looking straight at it and the line is a cursor with a
// count. Once Enter hands the keyboard back the filter is still on, still
// hiding rows and still shrinking every count on the screen, so the line
// stays and gains the way out — the band is the only place on the frame
// that says a filter exists at all, and a reader who set one an hour ago
// should not have to open a help overlay to find out how to lift it.
//
// What is counted is the tasks the filter lets through and not the rows
// drawn, because a collapsed band draws a header over its matches without
// drawing them. Counting rows would say "two of fifteen" under a heading
// that says four, and the two numbers on one screen would disagree.
func (m Model) filterLine() string {
	p := m.opts.Words
	filter := strings.ToLower(strings.TrimSpace(m.filter))
	shown := 0
	for _, t := range m.board.Tasks {
		if matches(t, filter) {
			shown++
		}
	}
	typed, role := m.filter, Accent
	if typed == "" {
		typed, role = p.T("filter.placeholder", "repository, id or title"), Dim
	}
	line := Paint(role).Render("/"+typed) + dot + Paint(Dim).Render(p.T("band.shown", "{n} of {total} shown",
		about("n", strconv.Itoa(shown)), about("total", strconv.Itoa(len(m.board.Tasks)))))
	if m.filtering {
		return line
	}
	return line + dot + Paint(Dim).Render(p.T("band.filter_clear", "{key} clears it",
		about("key", m.keys.Back.Help().Key)))
}

// runningLine names the one task a process is holding right now.
//
// It is the first Running task in the board's order and not the one under
// the cursor: the band answers "what is happening", which is a question
// about the machine, and the row answers "what am I looking at". The record
// cannot yet say more than the phase and how long it has been in it — there
// are no per-tool events — so the band says that and stops rather than
// guessing at what the engine is doing.
func (m Model) runningLine(t view.Task) string {
	p := m.opts.Words
	pieces := []string{Paint(Accent).Render(t.ID), Paint(Live).Render(m.phaseWord(t))}
	if age := elapsed(m.now, t.Since); age != "" {
		pieces = append(pieces, p.T("band.elapsed", "{d} in", about("d", age)))
	}
	if engine := engineAndModel(t); engine != "" {
		pieces = append(pieces, engine)
	}
	if t.Flow != "" {
		pieces = append(pieces, t.Flow)
	}
	return strings.Join(pieces, dot)
}

// engineAndModel is which engine ran the phase and on which model, as one
// field. Neither word is translated: they are names the record carries.
func engineAndModel(t view.Task) string {
	switch {
	case t.Engine != "" && t.Model != "":
		return t.Engine + "/" + t.Model
	case t.Engine != "":
		return t.Engine
	}
	return t.Model
}

// idleLine is what the band says when nothing is running, and it says what
// there is instead rather than only what there is not.
func (m Model) idleLine() string {
	p := m.opts.Words
	nothing := p.T("band.nothing_running", "nothing is running")
	todo := m.board.Counts[view.ToDo]
	if todo == 0 {
		return nothing + dot + p.T("band.nothing_todo", "nothing to do")
	}
	return nothing + dot + p.P("band.todo", todo, "{n} to do", "{n} to do") +
		dot + p.T("band.write_one", "press n to start one")
}

// controlSaid is what the band says about a word that was written.
//
// A key per verb rather than one sentence with the verb dropped into it.
// The word on the wire is English because the control port is a protocol
// and not prose, so a single "asked {id} to {word}" puts an English
// infinitive inside a Spanish clause every time it is read in Spanish.
// Four whole sentences translate; one sentence with a hole in it does not.
// The last branch is for a word this window does not raise, and it is the
// only place a raw wire word can still reach the band.
func (m Model) controlSaid(msg controlMsg) string {
	if msg.Err != nil {
		return msg.Err.Error()
	}
	p, id := m.opts.Words, about("id", msg.ID)
	switch msg.Word {
	case "pause":
		return p.T("msg.asked_pause", "asked {id} to pause", id)
	case "resume":
		return p.T("msg.asked_resume", "asked {id} to resume", id)
	case "continue":
		return p.T("msg.asked_continue", "asked {id} to continue", id)
	case "cancel":
		return p.T("msg.asked_cancel", "asked {id} to cancel", id)
	}
	return p.T("msg.control_sent", "asked {id} to {word}", id, about("word", msg.Word))
}

// startedSaid is what the band says about a run that began.
func (m Model) startedSaid(msg startedMsg) string {
	if msg.Err != nil {
		return msg.Err.Error()
	}
	return m.opts.Words.T("msg.started", "{id} is running, as process {pid}",
		about("id", msg.ID), about("pid", strconv.Itoa(msg.Pid)))
}
