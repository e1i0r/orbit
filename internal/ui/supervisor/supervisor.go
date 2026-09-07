// Package supervisor is the screen a conversation with the supervisor is
// held on: the thread, what is being typed into it, the conversations it can
// be moved between, and the column of what Orbit knows down its side.
//
// It is entered through this file and nothing else. State is what the screen
// is, Env is the little world the window lends it, and Out is what it asks
// the window for when it has done what it can itself — a sentence for the
// band, a question that has gone out, the screen it is leaving for.
package supervisor

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// Env is what this screen needs of the world: the words it speaks, the keys
// it answers, the room it has, the doors of the record it writes through,
// and the few standing facts it puts in its heading. Nothing else — it is
// handed no board and no Model, so it can decide nothing about a task.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	Frame layout.Frame
	Now   time.Time
	// Spinner is the frame the window's own clock is on, borrowed rather
	// than kept: a second clock would turn it twice as fast.
	Spinner func(theme.Role) string
	// Busy is whether a question is already out. The window owns it because
	// the band and the spinner read it too.
	Busy bool

	// The record's doors. Any of them may be nil — a window built without
	// one says so rather than pretending the gesture worked.

	// Log is every line of every conversation, folded: the retractions
	// marked and the removed conversations already gone.
	Log func() ([]view.SupervisorLine, error)
	// Knows is what Orbit has learned about the code being worked in.
	Knows func() []knowledge.Fact
	// Record writes one line into one conversation.
	Record func(conversation, by, channel, message string) error
	// Retract takes back the turn written at that moment.
	Retract func(at time.Time) error
	// Ask sends the question and answers with the command that carries the
	// reply back. The window makes it: it knows which engine is dialled and
	// what message its own update loop is waiting for.
	Ask func(conversation, text string) tea.Cmd
	// NewID is the id a conversation started right now carries.
	NewID func() string
	// Forget takes one conversation off the list.
	Forget func(id string) error
	// Learn writes down a fact about the code, and Note puts a line in one
	// task's notes.
	Learn func(stops bool, scope, repo, phrase string) error
	Note  func(id, text string) error

	// What the heading and the thread say about the world outside.

	// Engine is the one that answers here, already dialled.
	Engine string
	// IsEngine says whether a line was written by one of them, which is how
	// a turn is drawn as an answer rather than as something somebody typed.
	IsEngine func(name string) bool
	// Autopilot is whether the board is running itself.
	Autopilot bool
	// Repo is the repository a fact with no scope is about. The window
	// works it out from what is selected: this screen has no board.
	Repo string
	// Tasks is what a mention can name.
	Tasks []Mention
}

// A Mention is a task the line can name: its id, and the title nobody
// remembers the id without.
type Mention struct {
	ID    string
	Title string
}

// Out is what the screen asks the window for, having done what it can.
type Out struct {
	// Said is a sentence for the band.
	Said string
	// Leave is the reader closing the screen; Back is what they came from.
	Leave bool
	Back  int
	// Asking is a question that has gone out: the window marks itself busy
	// and starts the frame clock, because the spinner is the window's.
	Asking bool
	Cmd    tea.Cmd
}

// said is one sentence and nothing else.
func said(text string) Out { return Out{Said: text} }

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}

// State is the screen. It is a value, copied for every message the window
// handles, which is why the one thing that must survive a copy — the last
// rendering of the thread — hangs off it as a pointer.
type State struct {
	// back is the screen this one was opened from, kept as the window's own
	// number: which screens there are is the window's business, and this
	// one only hands it back when it leaves.
	back  int
	input string

	offset int
	lines  []view.SupervisorLine
	err    error
	// knows is what Orbit has learned about the code being worked in, drawn
	// down the side. It is read when the thread is, for the reason Sync
	// gives.
	knows []knowledge.Fact

	// picking is the mode that takes a turn back: ↑↓ choose a line instead
	// of scrolling and ↵ withdraws it instead of sending. It is a mode
	// rather than a key on its own because on this screen every printable
	// key already types, and the arrows already scroll — there was no free
	// gesture left for "which line".
	picking bool
	pick    int

	// conversation is the one the screen has open, and all is every line of
	// every conversation — lines above is only this one's. The list needs
	// all of them to say what there is; everything else on this screen —
	// what is drawn, what a retraction picks from, how many messages it
	// says — is about the conversation being read.
	conversation string
	all          []view.SupervisorLine
	// list is whether the conversations are up instead of the thread, and
	// listSel is which of them the cursor is on.
	list    bool
	listSel int

	// follow is whether the thread is pinned to its own end. It replaces a
	// sentinel offset of 999999, which was the bug behind "the scroll does
	// not work": one press of ↑ took it to 999998, still far past the
	// bottom, so the thread did not move until the key had been pressed a
	// million times. Every movement is clamped where it is made now, and
	// this says what "at the bottom" means without a magic number.
	follow bool

	// thread is the last rendering, kept so that a redraw which changes
	// nothing does not render forty messages again. See cache.go.
	thread *threadCache
}

// Open is the screen coming up, on the conversation last spoken in. back is
// the screen it was opened from, which leaving it returns to.
func Open(back int, e Env) State {
	s := State{back: back, follow: true, thread: &threadCache{}}

	// The thread has to be read before there is anything to choose from,
	// and openLatest reads it again once it knows which one it is opening.
	return s.Sync(e).openLatest(e)
}

// Back is the screen this one was opened from.
func (s State) Back() int { return s.back }

// Conversation is the one open, which is where a line written by something
// other than this screen — the breaker, a delivery — belongs.
func (s State) Conversation() string { return s.conversation }

// Follow pins the thread to its own end, for an answer that has just landed.
func (s State) Follow() State {
	s.follow = true

	return s
}

// Type puts text in the line being written, which is what a paste is.
func (s State) Type(text string) State {
	// While a line is being picked there is nothing being typed into, and
	// text arriving would land in a field nobody can see.
	if !s.picking {
		s.input += text
	}

	return s
}

// Sync reads the thread and what is known, and throws the last rendering
// away.
//
// Read here and not while drawing: the port reads two directories off disk,
// and a frame is drawn ten times a second. This is also what puts a rule on
// the side the moment it is written, since writing one syncs.
//
// The facts before the thread and not after them, because the two are
// separate doors: a window handed one and not the other has to get the one
// it was handed, and reading them in one order made the side depend on a
// port that has nothing to do with it.
func (s State) Sync(e Env) State {
	if e.Knows != nil {
		s.knows = e.Knows()
	}

	if e.Log == nil {
		return s
	}

	all, err := e.Log()
	s.all, s.err = all, err
	s.lines = linesIn(all, s.conversation)
	// What was drawn is not this thread any more. Saying so outright is
	// what makes a retraction a change: taking a turn back adds a line
	// rather than removing one, so the length alone would not.
	s.thread.invalidate()

	return s
}

// View is the whole screen: a heading, the thread under it, and what you are
// about to say at the foot.
func (s State) View(h, w int, e Env) []string {
	return s.rows(h, w, e)
}
