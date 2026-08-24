package ui

// One row of the board, drawn: which columns it shows, what its state
// column says, and how a field is cut to the cells the plan gave it.
//
// The state word is here rather than in internal/view because the fold
// produces keys and this is where keys become sentences — which is what
// keeps a Spanish reader from being handed an English word folded three
// layers down.

import (
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// columnGap is the space between two columns of a row, and it has to be the
// two cells layout.Plan.Width leaves room for. It is a separate constant
// because layout's own is unexported, and the pair is held together by the
// measured render: a row drawn with three cells where the plan allowed two
// is a row wider than the terminal it was planned for.
const columnGap = 2

// markGlyph is what the cursor's row is marked with, in the gutter.
const markGlyph = "▸"

// drawRow lays one task out under the column plan.
//
// A field the plan gave no cells is not drawn and takes no gap either, which
// is what makes a dropped column give its space to the title rather than
// leave a hole where it was.
func (m Model) drawRow(r row, w int, selected bool) string {
	word, role := m.stateWord(r.task)
	fields := []struct {
		cells int
		text  string
		role  Role
		right bool
	}{
		{m.plan.Repo, r.task.Repo, Dim, false},
		{m.plan.ID, r.task.ID, Accent, false},
		{m.plan.Title, r.task.Title, Dim, false},
		{m.plan.State, word, role, false},
		{m.plan.Model, r.task.Model, Dim, false},
		{m.plan.Elapsed, elapsed(m.now, r.task.Since), Dim, true},
	}
	var parts []string
	for _, f := range fields {
		if f.cells <= 0 {
			continue
		}
		cell := pad(f.text, f.cells, f.right)
		if !selected {
			cell = Paint(f.role).Render(cell)
		}
		parts = append(parts, cell)
	}
	mark := strings.Repeat(" ", gutter)
	if selected {
		mark = markGlyph + strings.Repeat(" ", gutter-1)
	}
	line := fit(mark+strings.Join(parts, strings.Repeat(" ", columnGap)), w)
	if selected {
		// The cursor's block is painted over the whole row rather than
		// over each field, because a style nested inside another ends at
		// the inner one's reset — and the block would stop halfway.
		return Paint(Sel).Render(line)
	}
	return line
}

// stateWord is what the row says the task is doing, and the role it is
// painted in.
//
// The keys are internal/view's own reason keys, written out as literals
// because that is the only form internal/words can verify against es.json.
// The fold produces a key and this switch translates it, which is what keeps
// a Spanish reader from being handed an English word folded three layers
// down.
func (m Model) stateWord(t view.Task) (string, Role) {
	p := m.opts.Words
	switch t.Reason.Key {
	case view.ReasonFailed:
		return p.T("reason.failed", "failed: {phase}", reasonArgs(t.Reason)...), Bad
	case view.ReasonFailedToStart:
		return p.T("reason.failed_to_start", "would not start"), Bad
	case view.ReasonGate:
		return p.T("reason.gate", "waiting: {phase}", reasonArgs(t.Reason)...), Warn
	case view.ReasonHeld:
		return p.T("reason.held", "held: {phase}", reasonArgs(t.Reason)...), Warn
	case view.ReasonTimedOut:
		return p.T("reason.timed_out", "timed out"), Bad
	case view.ReasonAbandoned:
		return p.T("reason.abandoned", "abandoned"), Warn
	case view.ReasonCancelled:
		return p.T("reason.cancelled", "cancelled"), Dim
	}
	if t.Damaged > 0 {
		return p.T("state.unreadable", "record unreadable"), Bad
	}
	switch view.BandOf(t) {
	case view.Running:
		return m.phaseWord(t), Live
	case view.Done:
		return p.T("state.finished", "finished"), OK
	case view.ToDo:
		return p.T("state.not_started", "not started"), Dim
	}
	return "", Dim
}

// phaseWord is the state word for a task a process is holding: the phase it
// is in, and how far through the flow that is.
//
// The fraction appears from the second phase on. On the first it would say
// only that the flow has more phases, which the reader is about to find out
// anyway — and the state column is the narrowest thing on the row.
func (m Model) phaseWord(t view.Task) string {
	if t.Phase == "" {
		return m.opts.Words.T("state.running", "running")
	}
	if total := m.totals[t.Flow]; t.PhaseN > 1 && total > 0 {
		return m.opts.Words.T("state.phase_of", "{phase} {n}/{total}",
			about("phase", t.Phase), about("n", strconv.Itoa(t.PhaseN)), about("total", strconv.Itoa(total)))
	}
	return t.Phase
}

// phaseTotals is how many phases each flow on the board has, looked up once
// per board rather than once per row per frame.
//
// Only the builtin flows are answered. A flow from $ORBIT_HOME/flows needs a
// flow.Source to resolve, and a Source is built over the store, which the
// window cannot reach — so a task written against a user flow shows its
// phase without a fraction, which is the honest answer rather than a
// fraction of a total that was guessed.
func phaseTotals(tasks []view.Task) map[string]int {
	totals := map[string]int{}
	for _, t := range tasks {
		if t.Flow == "" {
			continue
		}
		if _, done := totals[t.Flow]; done {
			continue
		}
		if f, err := flow.Builtin(t.Flow); err == nil {
			totals[t.Flow] = len(f.Phases)
		}
	}
	return totals
}

// reasonArgs converts a reason's values into the printer's, field for
// field. The two types are identical on purpose and are separate on
// purpose: internal/view may not import internal/words, so the conversion
// happens here, where both are already in scope.
func reasonArgs(r view.Reason) []words.Arg {
	args := make([]words.Arg, 0, len(r.Args))
	for _, a := range r.Args {
		args = append(args, words.Arg{Name: a.Name, Value: a.Value})
	}
	return args
}

// elapsed is how long the row has been in the state it is in, in one unit.
//
// The units are not translated. They are the same four letters in the
// languages Orbit ships, and a column seven cells wide has no room for a
// word — a number with a letter after it is read the same way in both.
//
// A negative age is clamped rather than drawn. The record's clock is the
// record's word and the fold does not correct it, so a log whose clock went
// backwards reaches here as a task that started after it was read.
func elapsed(now, since time.Time) string {
	if since.IsZero() {
		return ""
	}
	d := max(now.Sub(since), 0)
	switch {
	case d < time.Minute:
		return strconv.Itoa(int(d.Seconds())) + "s"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h"
	}
	return strconv.Itoa(int(d.Hours()/24)) + "d"
}

// pad cuts one field to its budget and fills what is left, measuring in
// cells throughout: an accented word is one column narrower than its length
// in bytes, and a column planned in bytes looks crooked exactly where
// nobody tests.
func pad(text string, cells int, right bool) string {
	if cells <= 0 {
		return ""
	}
	text = ansi.Truncate(text, cells, "…")
	space := strings.Repeat(" ", max(cells-lipgloss.Width(text), 0))
	if right {
		return space + text
	}
	return text + space
}

// headRow is a band's name, how many tasks are in it, and — on the right —
// the one thing worth saying about the band as a whole.
//
// The cursor may rest here, and pressing open folds the band rather than
// opening a task. That is why the header takes the same one-cell mark a row
// does: a cursor that vanished when it stepped onto a heading would read as
// a cursor that had been lost.
func (m Model) headRow(r row, selected bool, w int) string {
	name, count, right := m.bandName(r.band), strconv.Itoa(r.n), m.headHint(r)
	mark := " "
	if selected {
		mark = markGlyph
	} else {
		// Painted only when the row is not the cursor's, for the reason
		// drawRow gives: the cursor's block is one style over the whole
		// line, and a style nested in it would end at the inner reset.
		name = Paint(Accent).Render(name)
		count = Paint(m.countRole(r)).Render(count)
		if right != "" {
			// Painting an empty string is not free: it renders as a pair
			// of escape codes, which is zero cells and not zero bytes, and
			// spread would then right-align nothing against the far edge
			// and pad the whole line to get there.
			right = Paint(Dim).Render(right)
		}
	}
	line := spread(mark+name+"  "+count, right, w)
	if selected {
		return Paint(Sel).Render(line)
	}
	return line
}

// bandName is the heading over one band.
//
// All four are painted in the same role, and that is deliberate: a heading
// is a heading. Painting NEEDS YOU amber and RUNNING teal would spend the
// two loudest colours on furniture, and then the amber on a row that has
// actually failed has nothing left to be louder than.
func (m Model) bandName(b view.Band) string {
	p := m.opts.Words
	switch b {
	case view.NeedsYou:
		return "🛑 " + p.T("band.needs_you", "NEEDS YOU")
	case view.Running:
		return "⚡ " + p.T("band.running", "RUNNING")
	case view.ToDo:
		return "📋 " + p.T("band.to_do", "TO DO")
	case view.Done:
		return "🏁 " + p.T("band.done", "DONE TODAY")
	}
	return ""
}

// countRole paints the number over a band, and only one of the four numbers
// is ever worth a colour: how many things are waiting on a person.
func (m Model) countRole(r row) Role {
	if r.band == view.NeedsYou && r.n > 0 {
		return Warn
	}
	return Dim
}

// headHint is the right-hand end of a band header.
//
// The unread cap is said on the band it stops — nothing new starts while too
// many finished tasks stand unread, and the place a reader looks to ask why
// nothing started is the band that things start from. Otherwise a shut band
// says how to open it, because a heading with a number and no rows under it
// otherwise looks like a bug.
func (m Model) headHint(r row) string {
	if r.band == view.ToDo {
		if unread := board.Unread(m.board); m.atUnreadCap(unread) {
			limit := m.unreadCap()
			return m.opts.Words.T("band.unread_cap", "unread cap reached · {n} of {cap}",
				about("n", strconv.Itoa(unread)), about("cap", strconv.Itoa(limit)))
		}
	}
	if !m.expanded[r.band] {
		return m.keys.Open.Help().Key + " " + m.keys.Open.Help().Desc
	}
	return ""
}
