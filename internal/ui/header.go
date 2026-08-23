package ui

// The window's furniture: the rule, the header line, the activity band and
// the key bar. None of these is a list of tasks, and every one of them is a
// pure function of the width it is given and the model it is called on —
// nothing here asks the terminal anything.

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

const (
	// pipOn and pipOff are the standing switch, as one cell. They are
	// glyphs rather than the words on and off because the word beside them
	// already says which switch it is, and a filled circle is read without
	// being read.
	pipOn  = "●"
	pipOff = "○"

	// dot joins the pieces of a sentence the window assembles out of facts
	// — an id, a phase, a model. A comma would imply somebody wrote the
	// sentence.
	dot = " · "

	// headerGap is the space between two of the header's right-hand fields
	// and hintGap the space between two hints in the key bar. Four and two,
	// for the reason layout's gap is two: one space reads as a wide space
	// inside a field rather than as a boundary between two.
	headerGap = "    "
	hintGap   = "  "

	// The rule's two greys. This is the only pair in the window and the
	// only place the terminal's background colour changes anything: the
	// rule is furniture, so it has to sit below the text on both, and one
	// grey cannot do that.
	ruleOnLight = "250"
	ruleOnDark  = "238"
)

// rule is the horizontal line between two regions.
//
// It is where m.dark is spent. The value arrives as a tea.BackgroundColorMsg
// answering the tea.RequestBackgroundColor that Init sends, which is the only
// way this program ever learns it — lipgloss.HasDarkBackground and
// compat.AdaptiveColor both read the terminal from wherever they are called,
// and inside a render that is a blocking read in the middle of a frame.
func (m Model) rule(w int) string {
	if w <= 0 {
		return ""
	}
	shade := lipgloss.LightDark(m.dark)(lipgloss.Color(ruleOnLight), lipgloss.Color(ruleOnDark))
	return lipgloss.NewStyle().Foreground(shade).Render(strings.Repeat("─", w))
}

// headerLine is the program, the folder it was opened on, and the three
// standing facts.
//
// The right-hand fields are dropped from the right when the terminal cannot
// hold them all, which is why they are ordered least important last: the
// repository count is a number a reader can get elsewhere, and the unread
// pair is the brake that stops tasks from starting. Losing the brake to fit
// a count would be losing the one field on this line that changes what
// happens next.
func (m Model) headerLine(w int) string {
	left := " " + Paint(Accent).Render("orbit") + "  " + Paint(Dim).Render(m.opts.Root)
	fields := m.headerFields()
	for {
		right := strings.Join(fields, headerGap)
		gap := w - lipgloss.Width(left) - lipgloss.Width(right)
		if gap >= 1 {
			return left + strings.Repeat(" ", gap) + right
		}
		if len(fields) == 0 {
			return fit(left, w)
		}
		fields = fields[:len(fields)-1]
	}
}

// headerFields are the standing facts, most important first.
func (m Model) headerFields() []string {
	p := m.opts.Words
	unread, limit := board.Unread(m.board), m.unreadCap()

	pip, role := pipOff, Dim
	if m.autopilotOn() {
		pip, role = pipOn, Live
	}
	autopilot := Paint(Dim).Render(p.T("header.autopilot", "autopilot")) + " " + Paint(role).Render(pip)

	brake := Dim
	if limit > 0 && unread >= limit {
		brake = Warn
	}
	fields := []string{autopilot, Paint(brake).Render(p.T("header.unread", "unread {n}/{cap}",
		about("n", strconv.Itoa(unread)), about("cap", strconv.Itoa(limit))))}
	return append(fields, Paint(Dim).Render(p.P("header.repos", m.board.Repos, "{n} repo", "{n} repos")))
}

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
func (m Model) filterLine() string {
	p := m.opts.Words
	typed := m.filter
	if typed == "" {
		typed = Paint(Dim).Render(p.T("filter.placeholder", "repository, id or title"))
	}
	shown := 0
	for _, r := range m.rows() {
		if !r.head && !r.blank {
			shown++
		}
	}
	return Paint(Accent).Render("/"+typed) + dot + Paint(Dim).Render(p.T("band.shown", "{n} of {total} shown",
		about("n", strconv.Itoa(shown)), about("total", strconv.Itoa(len(m.board.Tasks)))))
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
		dot + p.T("band.write_one", "press n to write one")
}

// barLine is what can be pressed right now.
//
// It drops whole hints from the right rather than truncating them, because
// half a hint is a key a reader has to guess the rest of. Help and quit are
// never dropped: they are how a reader who is lost gets out, and a bar that
// drops them is a bar that fails exactly when it is needed.
func (m Model) barLine(w int) string {
	tail := Paint(Dim).Render("[" + m.keys.Help.Help().Key + "] [" + m.keys.Quit.Help().Key + "]")
	hints := m.hints()
	for {
		line := " " + strings.Join(append(hints, tail), hintGap)
		if lipgloss.Width(line) <= w || len(hints) == 0 {
			return fit(line, w)
		}
		hints = hints[:len(hints)-1]
	}
}

// hints are the bar's entries, in the order they are given up backwards.
//
// Everything about the task under the cursor comes from Affordances, so a
// key the bar offers is a key that will not be refused when it is pressed.
// The bar shows what can be done; the menu, one level down, shows what
// cannot and why.
func (m Model) hints() []string {
	var out []string
	r, ok := m.selected()
	if ok {
		out = append(out, hint("↑↓", m.opts.Words.T("key.move", "move")), hintFor(m.keys.Open))
	}
	out = append(out, hintFor(m.keys.New))
	if ok && !r.head {
		for _, a := range m.keys.Affordances(r.task, m.conditions()) {
			if a.OK && a.Key.Help().Key != m.keys.Open.Help().Key {
				out = append(out, hintFor(a.Key))
			}
		}
	}
	return append(out, hintFor(m.keys.Filter))
}

// hintFor is one binding as the bar prints it, and hint is the same for the
// arrow pair, which is two keys with one meaning and so has no binding of
// its own.
func hintFor(b key.Binding) string { return hint(b.Help().Key, b.Help().Desc) }

func hint(glyph, desc string) string {
	return Paint(Accent).Render("["+glyph+"]") + " " + Paint(Dim).Render(desc)
}
