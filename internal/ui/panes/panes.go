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

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/prose"
	"github.com/e1i0r/orbit/internal/ui/theme"
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

	// Flow is the pipeline this run was started under, already resolved
	// from the name the task carries: resolving reads the flow store, which
	// is a door of the window's. FlowFailed is what went wrong resolving it.
	Flow       flow.Flow
	FlowFailed string

	// Failed is what went wrong reading the record, already in the reader's
	// language, and empty when nothing did. It is said at the top of
	// whichever pane the reader is on, because a pane that is empty because
	// nothing happened and a pane that is empty because the read failed are
	// two different facts.
	Failed string

	// Word is what the state column says about this run and Role the colour
	// it is painted in. They are the window's reading of the record — the
	// same sentence the board's row carries — so that a task says the same
	// thing wherever it is drawn.
	Word string
	Role theme.Role

	// Live is the mark a running task carries: a frame of the spinner while
	// something is happening, a bolt when it is not. The animation is the
	// window's, and a pane draws whatever frame it is handed.
	Live string

	// Keys is what the reader can press, for the panes that name a key in a
	// sentence. A letter written into a sentence is a letter nothing keeps
	// true: this line read "press 't'", and t on that screen is the thinking
	// dial.
	Keys keymap.Keys

	// Expanded is the reader's [e]: every row of every pane open at once,
	// and the brief set whole rather than cut to its first rows.
	Expanded bool

	// Diff is the working tree's changes as the window last read them.
	Diff string

	// Dials are what this task would run on, already resolved through the
	// engines port — which is a door of the window's.
	Dials Dials

	// Files is the task's own directory as the window last listed it,
	// FilesKnown whether an answer has landed at all, and FilesFailed why
	// the listing could not be made. A directory nothing has answered about
	// and one that is empty are two different sentences.
	Files       []view.File
	FilesKnown  bool
	FilesFailed string

	// DiffKnown and DiffFailed are the same two answers about the working
	// tree, which the artifacts pane counts its second half by. DiffMissing
	// says the checkout is not there at all, which is not a failure to
	// report: a task whose worktree was removed changed nothing in it.
	DiffKnown   bool
	DiffFailed  string
	DiffMissing bool

	// Width is the terminal's rather than the body's. The diff is read
	// across the whole window, and cutting it to a column the body happens
	// to have would be cutting it twice.
	Width int

	// Collapsed is which files of the diff the reader has shut, and
	// Rationale whether the sentence the record gives for each change is
	// set beside it.
	Collapsed map[string]bool
	Rationale bool

	// Read is what a file the reader opened turned out to hold, and whether
	// it has been asked for at all. A file is read when it is opened and
	// not before: reading every file of the directory on every tick to draw
	// a list of names would be a read nobody asked for.
	Read func(name string) (File, bool)

	// Reach is what the window last read about what this change touches:
	// the history's own answer, the checks run on both sides, and the three
	// states each of those can be in. Nothing found and nothing read yet
	// are different facts, and a pane that folded them into one would tell
	// a reader a repository has no coupling when what happened is that git
	// timed out.
	Reach Reach

	// Said is the window's word for an error. The panes are handed
	// sentences rather than errors wherever there is one to hand over; this
	// is for the errors that arrive inside a reading, where a check that
	// could not be run carries its own.
	Said func(error) string

	// Spinner is a frame of the animation, painted, for the panes that are
	// waiting on something.
	Spinner string

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

// Dials are the four settings a run is made under, as the reader would read
// them: what the task itself carries where it has run, and what the knobs
// say where it has not.
type Dials struct {
	Engine   string
	Model    string
	Effort   string
	Thinking string
}

// folded says whether a section of the overview is closed. Absent is open,
// so a window nobody has folded anything in shows everything it has.
func (e Env) folded(key string) bool {
	if e.Folded == nil {
		return false
	}

	return e.Folded(key)
}

// sectionHead is one section's head, in the state that section is in.
func (e Env) sectionHead(key, label, note string, w int) string {
	return prose.Section(label, note, w, !e.folded(key))
}

// The sections of the overview that fold, in the order the pane draws them.
//
// The keys are this package's own vocabulary rather than the labels above
// them, which are translated and would change what a fold applies to when
// the window's language changed. The window remembers which are shut, and
// asks nothing about what they hold.
const (
	FoldPhases  = "phases"
	FoldChanges = "changes"
	FoldDeliver = "deliver"
)

// Sections is every one of them, in that order.
var Sections = []string{FoldPhases, FoldChanges, FoldDeliver}

// Changed is the paths a diff touched.
//
// It is asked from outside the overview because the artifacts pane counts
// the worktree by it: the worktree is a checkout Orbit does not keep, so the
// file that is not in the diff is the file the run left alone.
func Changed(diff string) []string { return parseDiffSummary(diff).files }

// A File is what an opened file turned out to hold, or why it could not be
// read. The two are one type because a row shows one or the other and never
// both.
//
// Whole is whether Text is all of it. A pane that showed the first part of a
// file without saying so would be a pane a reader draws conclusions from
// about the part that is not there.
type File struct {
	Text   string
	Whole  bool
	Failed string
}

// read is what one opened file holds, and no for a window built without that
// port rather than reaching through it.
func (e Env) read(name string) (File, bool) {
	if e.Read == nil {
		return File{}, false
	}

	return e.Read(name)
}

// Reach is what was read about what this change touches, and how far each
// reading got.
type Reach struct {
	// History is what the repository's own log says usually moves with
	// these files. Read is whether an answer has landed, Asking whether one
	// is out, and Failed why it could not be read.
	History repo.Impact
	Read    bool
	Asking  bool
	Failed  string

	// NoHistory says this build cannot read a repository at all, which is
	// not a failure: it is a window built without that port.
	NoHistory bool

	// Checks is the flow's own checks run on both sides, Compared whether
	// they have been, and Running whether they are out right now. Offered
	// is what would be run, named so the reader can see the cost before
	// paying it — this is a test suite twice on somebody's machine.
	Checks       []repo.Divergence
	Compared     bool
	ChecksFailed string
	Running      bool
	Offered      []repo.Check
	For          time.Duration
}

// said is the window's word for an error, and the error's own where the
// window handed no port for it.
func (e Env) said(err error) string {
	if err == nil {
		return ""
	}

	if e.Said == nil {
		return err.Error()
	}

	return e.Said(err)
}
