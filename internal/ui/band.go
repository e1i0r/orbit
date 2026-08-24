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

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// bandLine is the activity band, and it never comes back empty.
func (m Model) bandLine(w int) string {
	left := " " + m.bandLeft()
	right := m.bandRight()
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)

	if leftW+rightW+4 <= w {
		space := w - leftW - rightW
		return left + strings.Repeat(" ", space) + right
	}
	return fit(left, w)
}

func (m Model) bandLeft() string {
	switch {
	case m.filtering:
		return m.filterLine()
	case m.confirm == confirmCancel:
		return Paint(Warn).Render(m.opts.Words.T("msg.confirm_cancel",
			"cancel {id}? press y to confirm, anything else to leave it running",
			about("id", m.confirmID)))
	case m.message != "" && m.now.Sub(m.messageAt) < messageLife:
		return Paint(Accent).Render(m.message)
	case m.filter != "" || m.repoFilter != "" || m.queueFilter != nil:
		return m.filterLine()
	}
	for _, t := range m.board.Tasks {
		if view.BandOf(t) == view.Running {
			return m.runningLine(t)
		}
	}
	return Paint(Dim).Render(m.idleLine())
}

func (m Model) bandRight() string {
	p := m.opts.Words
	var chips []string

	// Autopilot chip
	pip, role := pipOff, Dim
	if m.autopilotOn() {
		pip, role = pipOn, Live
	}
	chips = append(chips, Paint(Dim).Render("⚡ "+p.T("header.autopilot", "autopilot"))+" "+Paint(role).Render(pip))

	// Model / knob chip
	chip := m.knobChip()
	if chip != "" {
		chips = append(chips, Paint(Accent).Render("🧠 "+chip))
	} else {
		chips = append(chips, Paint(Dim).Render("🧠 claude"))
	}
	return strings.Join(chips, "    ")
}

// filterLine is what is being typed, and how much of the board it is
// hiding. Saying the second half is the rule the plan states as "say it when
// you show less than you have": a filter is the one thing on this screen
// that can hide a task the reader is certain they wrote.
func (m Model) filterLine() string {
	p := m.opts.Words
	filter := strings.ToLower(strings.TrimSpace(m.filter))
	shown := 0
	for _, t := range m.board.Tasks {
		if matches(t, filter) && matchesRepo(t, m.repoFilter) && (m.queueFilter == nil || view.BandOf(t) == *m.queueFilter) {
			shown++
		}
	}
	var parts []string
	if m.queueFilter != nil {
		parts = append(parts, Paint(Accent).Render(m.bandName(*m.queueFilter)))
	}
	if m.filter != "" || m.filtering {
		typed, role := m.filter, Accent
		if typed == "" {
			typed, role = p.T("filter.placeholder", "repository, id or title"), Dim
		}
		parts = append(parts, Paint(role).Render("/"+typed))
	}
	if m.repoFilter != "" {
		parts = append(parts, Paint(Accent).Render(p.T("band.repo_filter_tag", "repo:{repo}", about("repo", m.repoFilter))))
	}
	parts = append(parts, Paint(Dim).Render(p.T("band.shown", "{n} of {total} shown",
		about("n", strconv.Itoa(shown)), about("total", strconv.Itoa(len(m.board.Tasks))))))
	line := strings.Join(parts, dot)
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

// commandSaid is what the band says about a palette command that came back.
//
// The error is passed on exactly as the command phrased it — the same rule
// controlSaid keeps, for the same reason. The success is one sentence per
// nothing: unlike a control word, there is no verb to conjugate, only a
// name that finished.
func (m Model) commandSaid(msg commandMsg) string {
	if msg.Err != nil {
		return msg.Err.Error()
	}
	return m.opts.Words.T("msg.command_done", "{name} finished", about("name", msg.Name))
}
