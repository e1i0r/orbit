// Package panes is the body of the task screen: the twelve ways one run can
// be read.
//
// Each pane is a door, and every one of them answers the same shape —
// the lines to draw, and where in them the rows a click can land on were
// put. They are functions of the record and nothing else: what is folded,
// what is scrolled and what is on screen is the window's, handed over in
// the Env, because folding is what a reader does to a screen they come back
// to and the record has no opinion about it.
//
// The panes are here rather than in the window for the reason the other
// screens are: a pane is read by the reader and by nobody else, and a
// function that takes the whole Model can reach the board, the ports and
// the keyboard while it is drawing a table of costs.
package panes

import (
	"time"

	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// Env is the run being read, as the panes need it.
type Env struct {
	Words *words.Printer
	Frame layout.Frame
	// Now is the clock the elapsed columns are counted against, so that
	// every row of a frame is measured from the same instant.
	Now time.Time

	// Task is the run, and Gone says the board no longer has it: a pane
	// says so in a sentence rather than drawing an empty page.
	Task view.Task
	Gone bool

	// Entries is the record of that run, folded.
	Entries []view.Entry

	// Failed is what went wrong reading the record, already in the reader's
	// language, and empty when nothing did. It is said at the top of
	// whichever pane the reader is on, because a pane that is empty because
	// nothing happened and a pane that is empty because the read failed are
	// two different facts.
	Failed string

	// Raw is the reader's "show me what was written" switch: the panes that
	// set markdown draw the record itself instead, framing and all, because
	// a reader who asks for raw is asking what was written down and not what
	// was made of it.
	Raw bool

	// Priced is whether this run's engine is spoken about in money. It is
	// asked once for the whole pane rather than per row: every row is the
	// same task on the same engine, and asking per row would put a dollar
	// sign on one phase and not the next.
	Priced bool

	// Folded says whether a section of the overview is closed, RowOpen
	// whether one row of the pane being built shows everything it has, and
	// AttemptOpen whether an attempt shows the phases under it.
	Folded      func(key string) bool
	RowOpen     func(row int) bool
	AttemptOpen func(n int) bool
}

// row says whether one row of the pane shows everything it has, and no for
// a window built without that answer rather than reaching through it.
func (e Env) row(i int) bool {
	if e.RowOpen == nil {
		return false
	}

	return e.RowOpen(i)
}

// attempt says whether an attempt shows the phases under it, and yes for a
// window built without that answer: a run whose attempts nobody has shut
// shows all of them.
func (e Env) attempt(n int) bool {
	if e.AttemptOpen == nil {
		return true
	}

	return e.AttemptOpen(n)
}

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg { return words.Arg{Name: name, Value: value} }
