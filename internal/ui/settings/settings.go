package settings

// The settings screen, and the only names anything outside it may say.
//
// Every other file here is a satellite of one of the doors — Open, Rows,
// Key, Apply and View — and holds nothing exported of its own. What the
// screen needs of the world arrives in Env, and what it asks of the world
// goes back in Out: it writes to the settings file through the port it was
// handed, and everything else — the band, which screen is up, the dials the
// rest of the window keeps — is the window's to do.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/words"
)

// State is the screen: which row the cursor is on, whether it is being typed
// into, and what has been typed so far.
type State struct {
	sel     int
	editing bool
	typed   string
	// flows is what the flow dial offers, read once when the screen opens
	// rather than while it draws.
	//
	// Rows is called from View, from every keypress and from every mouse
	// event, so asking the flows directory inside it would be one
	// os.ReadDir per frame — and, worse, two readings taken at two moments
	// deciding the same dial.
	flows []string
	// off is the first line of the table on show. Lines and not rows,
	// because a click lands on a line: the mouse has to add back exactly
	// what the drawing took off, and a count of rows would have to be
	// multiplied out at both ends by whoever remembered to.
	off int
}

// Env is what this screen needs of the world, and nothing more. It is built
// by the window on every call rather than held here: a port that changed
// between two keystrokes is the window's to notice, not this screen's to
// cache.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	// Frame is the room the window lends it. The table is taller than the
	// screen, so how many lines there are to scroll into is part of every
	// gesture and not only of drawing: a cursor moved is a cursor that has
	// to be brought back on screen, and that cannot be worked out at draw
	// time from a height the key press already threw away.
	Frame layout.Frame
	Store Store
	Dials Dials
	// Kept is every setting the vocabulary declares, in the order it
	// declares them, and Choose writes one by the name it gives it.
	//
	// The table used to be written out here by hand, row by row, and every
	// setting added since went into the vocabulary and not into it: six of
	// thirteen were declared and never drawn, among them whether Orbit may
	// interrupt you and which account may command it over a chat. A table
	// read off the declaration cannot fall behind it.
	//
	// Functions and not an interface, for the reason Engines and Models
	// are: what they answer is shaped by internal/verb, which this package
	// may not name, so the window converts on the way in.
	Kept   func() []Kept
	Choose func(key, value string) error
	// Engines, Models and Efforts are the build's catalogue, which this
	// screen reads and never decides: an engine the build does not have is
	// still the reader's setting, and the dial simply has no pill lit.
	Engines func() []string
	Models  func(engine string) (ids, labels []string)
	Efforts func(engine string) (ids, labels []string)
	Flows   func() []string
}

// Kept is one setting as this screen draws it: what it is called, what it
// holds now, and what it means.
//
// The screen's own shape and not internal/verb's, for the reason every
// screen here has one: what crosses into a screen is data, through a port.
type Kept struct {
	Name  string
	Value string
	About string
	Group string // the heading it is listed under
}

// Dials are the four choices a run is made with, as the window holds them.
// Two of them — effort and thinking — are not in the settings file at all,
// so a change to either comes back in Out for the window to keep.
type Dials struct {
	Engine   string
	Model    string
	Effort   string
	Thinking string
}

// Store is the settings file as this screen holds it, and it is down to one
// question.
//
// It used to be two interfaces of thirteen methods — one getter and one
// setter per setting — and that shape is what let six settings be declared
// and never drawn: reaching this screen meant adding a fourteenth and
// fifteenth method to a port every implementation had to keep. The table now
// arrives through Env.Kept and goes back through Env.Choose, which name
// settings rather than having a method each, so a setting added to the
// vocabulary is a row here the same afternoon.
//
// What is left is the one question neither of those answers.
//
// It is declared here rather than imported because the caller is what has
// one: the window passes whatever satisfies this, and the settings file's
// own shape stays its own.
type Store interface {
	// Fresh is what one setting reads as when nobody has chosen anything.
	//
	// Asked rather than known, because what a setting comes as is declared
	// once in internal/verb beside what it means and what it accepts — and
	// a screen that kept its own copy of the defaults would be the copy
	// that drifts. A name this screen has that the table does not answers
	// empty, which is what the two dials the file does not hold come back
	// as, and the right answer for them.
	Fresh(key string) string
}

// Out is what the screen asks the window for, having done what it could
// itself. Every field is empty when there is nothing to ask.
type Out struct {
	// Said is a sentence for the band.
	Said string
	// Close is the reader leaving the screen.
	Close bool
	// Dials is the two knobs the window keeps, when one of them moved.
	Dials *Dials
	// Lang is the new language, when it changed: the window reloads its
	// catalogue, which is not something a screen can do to itself.
	Lang string
	Cmd  tea.Cmd
}

// Open is the screen as it comes up, with the flows read once.
func Open(e Env) State {
	var names []string
	if e.Flows != nil {
		names = e.Flows()
	}

	return State{flows: names}
}
