// Package menu is what can be done to the thing under the pointer,
// including what cannot, and why not.
//
// It is a package of its own because it answers one question — "what are my
// moves here?" — out of lists that must never be allowed to drift: the
// affordances, which is what the key bar's shortlist is cut from; the
// command table, which is what the palette reaches; and the panes of the
// task being read. The menu shows all of them whole, and the bar and the
// line each show their own half.
//
// No entry has behaviour of its own. Choosing one answers with the
// keystroke its binding names — which the window puts through the same map
// a pressed key goes through, so a refused verb refuses here with the
// sentence it refuses everywhere else — or with the command it names, run
// through the same watch every other command is.
package menu

import (
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/words"
)

// Env is what this screen needs of the world: the words it speaks, the keys
// it answers, the room it has, and the lists it is a menu of.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	Frame layout.Frame

	// Detail says the menu was opened inside a task. The panes are moves
	// there and nowhere else, and the title says which menu this is.
	Detail bool

	// Commands is the table, in the order a reader learned it.
	Commands []Command

	// Panes are the task's, each with the key that opens it. They are
	// handed over rather than listed here: a menu with its own copy of the
	// twelve stops being true the day one of them changes.
	Panes []Pane

	// Verbs is what can be done to one task, refusals included. ok is
	// false for a task that left the board while its menu was up, which is
	// a menu with nothing on it and a line saying so.
	Verbs func(id string) (list []keymap.Affordance, ok bool)

	// Args is what a command about that task is run with: the repository
	// and the id, already filled in, because the menu was opened on the
	// task and the reader should not have to type either.
	Args func(id string) []string
}

// A Command is one row of the menu that runs something: what the window
// shows of a command, and nothing of what the command does.
type Command struct {
	Name string
	// About is what it does and Because is why it is refused, both already
	// in the reader's language. A refusal replaces the description rather
	// than joining it: the reason is the part a reader acts on.
	About   string
	Because string
	Refused bool
	// AboutATask keeps a verb off the board's menu. That menu is opened on
	// no row, so there is no task for such a verb to be about: choosing
	// one ran the command bare and answered with its usage.
	AboutATask bool
	// Children are the family's words after the parent's. The menu lists
	// them under it; every door runs them through the parent.
	Children []Child
}

// Child is one subcommand of a family: its own word and what it does.
type Child struct {
	Name  string
	About string
}

// A Pane is one of the task's, and the key that opens it.
type Pane struct {
	Key    string
	Title  string
	Detail string
}

// An Entry is one drawn row: the glyph or the name, the sentence describing
// it, and — where it cannot be done here — the reason on the same line.
//
// What choosing it does is one of three, and at most one is set. Pane shows
// a pane, Command runs a command, and anything else sends Glyph as a
// keystroke.
type Entry struct {
	Glyph  string // the keystroke the entry sends, when it has one
	Title  string // what the entry is called
	Detail string // its description, dimmed, when it has one
	Dim    bool   // refused here
	Reason string // why, when Dim
	Head   bool   // names the block below it; not a thing to choose

	Pane    string   // the pane it opens
	Command string   // the command it runs
	Args    []string // what that command is run with
	Says    bool     // the command takes a message: the box is opened for it
}

// Out is what the menu asks the window for. At most one of Pane, Send and
// Run is set, and Leave says the menu itself comes down.
type Out struct {
	Leave bool
	// Pane is the key of the pane to show.
	Pane string
	// Send is a keystroke to put through the keyboard's own map, which is
	// how a verb chosen here is the verb pressed.
	Send string
	// Run is the command to run and Args what with. Ask says it takes a
	// message instead: the window opens the box rather than running it.
	Run  string
	Args []string
	Ask  bool
}

// State is the menu while it is up, and nothing while it is down. task is
// what the menu is about, and empty means the board's menu — the commands
// that are not about any one task.
//
// offset is the first entry drawn, moved only to keep the selection on
// screen: a task's panes and verbs together are more rows than a small
// window has, and a verb the reader cannot see is a verb they do not have.
type State struct {
	open   bool
	task   string
	sel    int
	offset int
}

// Open brings the menu up on a target, its cursor on the first entry there
// is to choose — which is not the first row, on a menu that names its
// blocks.
func Open(id string, e Env) State {
	s := State{open: true, task: id}
	s.sel = max(0, choice(s.entries(e), 0, 1))

	return s.keepSeen(e)
}

// Up is whether the menu is showing, which decides who owns the keyboard
// and what the body draws.
func (s State) Up() bool { return s.open }

// Task is what the menu is about, empty on the board's own.
func (s State) Task() string { return s.task }

// At is where the cursor is, which the pointer reads to tell a click that
// moves the selection from one that acts on it.
func (s State) At() int { return s.sel }

// Point puts the cursor on one entry.
func (s State) Point(i int) State {
	s.sel = i
	return s
}

// Entries is the menu as it stands right now, recomputed from the lists
// rather than remembered — a menu frozen at open time would keep offering
// verbs for a run that finished while it was up.
func (s State) Entries(e Env) []Entry { return s.entries(e) }
