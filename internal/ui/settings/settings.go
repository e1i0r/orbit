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
}

// Env is what this screen needs of the world, and nothing more. It is built
// by the window on every call rather than held here: a port that changed
// between two keystrokes is the window's to notice, not this screen's to
// cache.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	Store Store
	Dials Dials
	// Engines, Models and Efforts are the build's catalogue, which this
	// screen reads and never decides: an engine the build does not have is
	// still the reader's setting, and the dial simply has no pill lit.
	Engines func() []string
	Models  func(engine string) (ids, labels []string)
	Efforts func(engine string) (ids, labels []string)
	Flows   func() []string
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

// Store is the settings file as this screen holds it. It is two interfaces
// and not one: what the table reads is in reader, what a change writes is in
// writer, and each door names only the half it uses — a door that can only
// read cannot write by accident, and the compiler is what says so.
//
// Both are declared here rather than imported because the caller is what has
// one: the window passes whatever satisfies this, and the settings file's
// own shape stays its own.
type Store interface {
	reader
	writer
}

// reader is the settings the table shows.
type reader interface {
	Language() string
	Autopilot() bool
	UnreadCap() int
	Engine() string
	Model() string
	Flow() string
	Theme() string
}

// writer is the settings a change puts back.
type writer interface {
	SetLanguage(string) error
	SetAutopilot(bool) error
	SetUnreadCap(int) error
	SetEngine(string) error
	SetModel(string) error
	SetFlow(string) error
	SetTheme(string) error
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
