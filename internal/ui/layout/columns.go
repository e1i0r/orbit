package layout

// The column plan: how wide each field of a task's row may be at a given
// terminal width, and — the part that is actually the decision — the order
// in which fields are given up when it will not all fit.

import (
	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// StateKey names the state column's cell budget in en.json. It is the one
// number in this package that is not decided here: how many cells the state
// word needs is a fact about the words, so it is declared where the words
// are and read back through the budgets function Columns is handed.
const StateKey = "col.state"

// The fixed numbers of a row, in cells.
const (
	// defaultStateCells is the state column's width when no budget is
	// declared — a catalogue that would not parse, or a build where en.json
	// lost its numbers. It is deliberately smaller than the declared
	// budget: a Printer with no catalogue is a Printer answering in
	// English, and English is the shorter of the two languages Orbit ships.
	// The alternative reading of a missing budget — zero, so no column —
	// would delete the word that says what the task is doing, which is the
	// one field a row cannot be read without.
	defaultStateCells = 10

	// elapsedCells is the elapsed column's width. Seven cells is the widest
	// thing the clock produces — "12d 23h" — and it is fixed rather than
	// measured because a column whose width follows the widest row would
	// change width as time passes, moving every field left of it while the
	// reader is looking at them.
	elapsedCells = 7

	// titleFloor is the narrowest a title may be squeezed to before the row
	// gives up on being a table at all. Below twenty cells a title cuts to
	// two or three words, which is not enough to tell two tasks in one
	// repository apart — and telling them apart is the only reason the
	// title is on the row.
	titleFloor = 20

	// gap is the space between two adjacent columns. Two cells, because one
	// reads as a wide space inside a field rather than as a boundary
	// between two, and three is a column of its own.
	gap = 2
)

// Plan is how many cells each field of a row gets. A field of zero cells is
// a field this width cannot afford, and the row does not draw it.
//
// Fallback says the row has given up on alignment: the terminal cannot give
// the state word its budget beside a readable title, so the row becomes
// id, state and elapsed and nothing else. It is a separate flag rather than
// something a caller infers from the widths because it changes how the row
// is drawn, not only how wide its fields are.
type Plan struct {
	Repo, ID, Title, State, Model, Elapsed int
	Fallback                               bool
}

// Width is how many cells the plan actually occupies, gaps included. A field
// of zero cells takes no gap either, which is what makes a dropped column
// give its space back to the title rather than leaving a hole where it was.
func (p Plan) Width() int {
	total, columns := 0, 0
	for _, cells := range []int{p.Repo, p.ID, p.Title, p.State, p.Model, p.Elapsed} {
		if cells <= 0 {
			continue
		}
		total += cells
		columns++
	}
	if columns > 1 {
		total += gap * (columns - 1)
	}
	return total
}

// Columns plans one row of the board at width w.
//
// w is the width available to a row, which is not the terminal's width: the
// caller subtracts whatever gutter it draws the cursor in first. Keeping the
// gutter out of here is what lets the same plan serve a list that has one
// and a detail pane that does not.
//
// budgets is the Printer's Cells method, taken as a function so that this
// package imports nothing to do with language. A caller with no catalogue at
// all can pass one that always answers zero.
//
// The drop order is the decision, and it is the list below. The model goes
// first, because which model ran is the field a reader asks about after the
// fact rather than while scanning. The repository column goes next, and it
// is already gone before any of this when the board holds a single
// repository — a column with one value in it is not information. The title
// absorbs whatever is left over and is the first thing to shrink when there
// is nothing left over. The id and the elapsed time never drop: the id is
// how a reader says which task, and the elapsed time is the only number on
// the row.
func Columns(w int, tasks []view.Task, budgets func(string) int) Plan {
	state := budgets(StateKey)
	if state <= 0 {
		state = defaultStateCells
	}
	id := widest(tasks, func(t view.Task) string { return t.ID })
	model := widest(tasks, func(t view.Task) string { return t.Model })
	repo := repoColumn(tasks)

	// Each attempt is the row with one more column given up. The title
	// takes what is left after the columns and their gaps, and the first
	// attempt whose title is still readable is the plan.
	for _, try := range []Plan{
		{Repo: repo, ID: id, State: state, Model: model, Elapsed: elapsedCells},
		{Repo: repo, ID: id, State: state, Elapsed: elapsedCells},
		{ID: id, State: state, Elapsed: elapsedCells},
	} {
		if title := w - try.Width() - gap; title >= titleFloor {
			try.Title = title
			return try
		}
	}
	return unaligned(w, id, state)
}

// unaligned is the row that has given up on columns: id, state and elapsed,
// with the state word whole.
//
// The state word is what the fallback exists to protect. A row that has lost
// its title and its repository still says what the task is doing and how
// long it has been doing it, and that is the least a row can say and still
// be worth a line of the terminal.
//
// It is written to be total rather than to be right at every width, because
// the widths it is reached at are below the one the window accepts at all:
// a terminal that cannot afford this row has already been refused by Fit,
// and what happens here only has to be free of negative numbers and of a row
// that overruns its width.
func unaligned(w, id, state int) Plan {
	p := Plan{ID: id, State: state, Elapsed: elapsedCells, Fallback: true}
	if over := p.Width() - w; over > 0 {
		// The id is the one of the three that is cut rather than dropped.
		// Half an id still says which task — ids share a prefix and differ
		// at the end, so the cut is visible — where half an elapsed time
		// says nothing and half a state word is a different word.
		p.ID = max(p.ID-over, 0)
	}
	for _, cells := range []*int{&p.Elapsed, &p.State, &p.ID} {
		if p.Width() <= w {
			break
		}
		*cells = 0
	}
	return p
}

// widest measures one field across the board, in cells and never in bytes:
// an accented repository name is one column narrower than its length in
// bytes suggests, and a column planned in bytes is a table that looks
// crooked in exactly the places nobody tests.
func widest(tasks []view.Task, field func(view.Task) string) int {
	cells := 0
	for _, t := range tasks {
		cells = max(cells, lipgloss.Width(field(t)))
	}
	return cells
}

// repoColumn is the width of the repository column, and zero when the board
// holds fewer than two repositories.
//
// A column with one value in it is not information — it is the same word
// written once per row, taking cells from the title, which is the field that
// actually distinguishes the rows. The program this replaces could not make
// this decision at all: it was a one-repository tool with a repository
// column bolted on, so the column was always there and always said the same
// thing.
func repoColumn(tasks []view.Task) int {
	seen := map[string]bool{}
	cells := 0
	for _, t := range tasks {
		if t.Repo == "" {
			continue
		}
		seen[t.Repo] = true
		cells = max(cells, lipgloss.Width(t.Repo))
	}
	if len(seen) < 2 {
		return 0
	}
	return cells
}

// Column names one field of a task's row. ColumnNone is the zero value and
// means no field: the gap between two of them, or a cell past the end of
// the last.
type Column int

const (
	ColumnNone Column = iota
	ColumnRepo
	ColumnID
	ColumnTitle
	ColumnState
	ColumnModel
	ColumnElapsed
)

// ColumnAt says which field of a row the cell x falls in, and whether it
// fell in one at all.
//
// x is counted from the first cell of the row, which is not the first cell
// of the terminal: the caller subtracts whatever gutter it draws the cursor
// mark in first, exactly as it already does before calling Columns.
//
// It walks the widths the plan handed out and never the text that was drawn
// into them, and that is the whole of why it lives here. Every field is cut
// to its budget and padded back out to it, so the plan's numbers are where
// the columns actually are; measuring the drawn row would find the end of a
// word rather than the end of a column, and would find it in bytes rather
// than in cells, which are two different places the moment a title carries
// an accent.
//
// The order is the order Width sums and the order the row is drawn in, and
// the three have to stay the same list. A field of zero cells is skipped
// and takes no gap with it, which is what makes a dropped column give its
// space to the title rather than leave a hole where it was.
//
// A gap belongs to neither of the columns it separates. Two cells wide, it
// answers ColumnNone — a caller that wants to treat a click there as a
// click on the row is free to, and this function will not decide it for it.
func (p Plan) ColumnAt(x int) (Column, bool) {
	if x < 0 {
		return ColumnNone, false
	}
	at, first := 0, true
	for _, c := range []struct {
		cells  int
		column Column
	}{
		{p.Repo, ColumnRepo},
		{p.ID, ColumnID},
		{p.Title, ColumnTitle},
		{p.State, ColumnState},
		{p.Model, ColumnModel},
		{p.Elapsed, ColumnElapsed},
	} {
		if c.cells <= 0 {
			continue
		}
		if !first {
			if at += gap; x < at {
				return ColumnNone, false
			}
		}
		first = false
		if x < at+c.cells {
			return c.column, true
		}
		at += c.cells
	}
	return ColumnNone, false
}
