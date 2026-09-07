// Package compose is the form a task is written into, by hand or from the
// URL of an issue.
//
// It is entered through this file. State is the form, Env is the little
// world the window lends it — the words, the room, where flows are kept and
// where work can be started — and Out is what it asks for: a sentence, a
// task to write down, a flow to open the designer on.
package compose

import (
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/tracker"
	"github.com/e1i0r/orbit/internal/ui/clip"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/typing"
	"github.com/e1i0r/orbit/internal/words"
)

const (
	composeTabManual = 0
	composeTabURL    = 1
)

// State is the form: which tab is open, which row the cursor is on, what
// has been typed into each field, and what a URL in it was read as.
type State struct {
	tab   int // composeTabManual or composeTabURL
	field int // active field index within current tab

	// Where the first phase runs, which the form picks rather than asks.
	repoPath string

	id          typing.Field
	text        typing.Field
	url         typing.Field
	parsedIssue *tracker.Issue
	// readable is whether this machine can read the body of an issue of
	// that tracker's kind — whether there is a credential for it. A URL
	// names an issue; only the body says what the work is, and a run
	// started without it is a run that will either invent the requirements
	// or do nothing.
	readable bool
	// reading is whether the body is being fetched right now, readAt when
	// that started, and startAfterRead whether the key the reader pressed
	// was Save and start — the save is finished once the answer lands.
	reading        bool
	readAt         time.Time
	startAfterRead bool

	flows   []string
	flowIdx int
}

// Open brings the form up. under is the repository the reader was looking
// at, which the form prefers when it picks where the first phase runs.
func Open(under string, e Env) State {
	path := startsIn(under, e)

	flowsListed := flow.List(e.Flows)

	var flows []string
	for _, f := range flowsListed {
		flows = append(flows, f.Name)
	}

	if len(flows) == 0 {
		flows = flow.BuiltinNames()
	}

	return State{tab: composeTabManual, repoPath: path, flows: flows}
}

// OpenIn brings the form up on a repository named by something other than
// the cursor — the directory an interactive session was held in — with the
// caret already in the box the task is written in, which is the only thing
// left to say.
func OpenIn(repo string, e Env) State {
	s := Open(repo, e)
	if repo != "" {
		s.field = composeText
	}

	return s
}

// startsIn is the repository the first phase runs in: the one the hint
// names, by name or by path, and otherwise the first Orbit knows.
//
// The form used to ask, and now it picks. The preference is the task the
// cursor was on, because a reader writing a task while looking at another
// one is usually writing about the same code — but it is a guess either
// way, and a wrong guess costs one repo.joined event: the task goes on into
// whatever it is actually worked in, and says so.
func startsIn(hint string, e Env) string {
	if hint != "" {
		for _, r := range e.Places {
			if r.Path != "" && (strings.EqualFold(r.Name, hint) || r.Path == hint) {
				return r.Path
			}
		}

		// A checkout Orbit has never listed, named by the only thing that
		// can name one it does not know: where it is. This is the session a
		// reader opened by hand somewhere else and is now writing about.
		if strings.ContainsRune(hint, filepath.Separator) {
			return hint
		}
	}

	// A repository with no path is one the board knows only by name —
	// a checkout a task was carried into, which view.Task names and does
	// not locate. It is somewhere work happened and not somewhere work can
	// be started, so the fall back is the first one Orbit can actually
	// reach, and a board where it can reach none is answered with nothing.
	for _, r := range e.Places {
		if r.Path != "" {
			return r.Path
		}
	}

	return ""
}

// Key is one press.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	switch {
	case msg.Code == tea.KeyEscape || key.Matches(msg, e.Keys.Back):
		return State{}, Out{Leave: true}
	case msg.Code == tea.KeyEnter || key.Matches(msg, e.Keys.Open):
		if msg.Mod&tea.ModCtrl != 0 {
			return s.Submit(true, e)
		}

		if msg.Mod&tea.ModShift != 0 || msg.Mod&tea.ModAlt != 0 {
			if s.tab == composeTabManual && s.field == composeText {
				s.text.Insert("\n")
				return s, Out{}
			}
		}

		return s.composeNext(false, e)
	case (msg.Code == 'r' || msg.Code == 'R') && msg.Mod&tea.ModCtrl != 0:
		return s.Submit(true, e)
	case (msg.Code == 'a' || msg.Code == 'A') && msg.Mod&tea.ModCtrl != 0:
		return s.composeCaret((*typing.Field).SelectAll), Out{}
	case (msg.Code == 'c' || msg.Code == 'C') && msg.Mod&tea.ModCtrl != 0:
		return s.composeCopy(false), Out{}
	case (msg.Code == 'x' || msg.Code == 'X') && msg.Mod&tea.ModCtrl != 0:
		return s.composeCopy(true), Out{}
	case (msg.Code == 'v' || msg.Code == 'V') && msg.Mod&tea.ModCtrl != 0:
		if pasted := clip.Read(); pasted != "" {
			return s.Type(pasted), Out{}
		}

		return s, Out{}
	case msg.Text == "+":
		if s.isComposeFlowField() {
			return s, Out{Flow: New}
		}
	case msg.Text == "i" || msg.Text == "I":
		if s.isComposeFlowField() {
			return s, Out{Flow: s.chosenFlow()}
		}
	// The arrows themselves always move, and the letters bound alongside
	// them only where nothing is being typed into. Up carries k and Down
	// carries j, so a form that matched the binding everywhere could not be
	// used to write "webhook" or "json": the two letters walked the fields
	// instead of landing in them.
	case msg.Code == tea.KeyUp || (s.isPillField() && key.Matches(msg, e.Keys.Up)):
		return s.composeVertical(-1, msg.Mod, e), Out{}
	case msg.Code == tea.KeyDown || (s.isPillField() && key.Matches(msg, e.Keys.Down)):
		return s.composeVertical(1, msg.Mod, e), Out{}
	case msg.Code == tea.KeyLeft:
		return s.composeArrow(-1, msg.Mod), Out{}
	case msg.Code == tea.KeyRight:
		return s.composeArrow(1, msg.Mod), Out{}
	case key.Matches(msg, e.Keys.PrevTab):
		return s.composeMove(-1), Out{}
	case msg.Code == tea.KeyTab || key.Matches(msg, e.Keys.NextTab):
		if msg.Mod&tea.ModShift != 0 {
			return s.composeMove(-1), Out{}
		}

		return s.composeTab(1), Out{}
	case msg.Code == tea.KeyBackspace:
		return s.composeEdit(func(in *typing.Field) { in.Backspace() }), Out{}
	case msg.Code == tea.KeyDelete:
		return s.composeEdit(func(in *typing.Field) { in.DeleteForward() }), Out{}
	case msg.Code == tea.KeyHome:
		return s.composeJump((*typing.Field).LineStart, msg.Mod), Out{}
	case msg.Code == tea.KeyEnd:
		return s.composeJump((*typing.Field).LineEnd, msg.Mod), Out{}
	}

	if (msg.Text == "1" || msg.Text == "2") && (s.isPillField() || s.typed() == "") {
		if msg.Text == "1" {
			s.tab = composeTabManual
		} else {
			s.tab = composeTabURL
		}

		s.field = firstComposeField(s.tab)

		return s, Out{}
	}

	if msg.Text != "" && !s.isPillField() {
		return s.composeEdit(func(in *typing.Field) { in.Insert(msg.Text) }), Out{}
	}

	return s, Out{}
}

// Starts is where the first phase of this task would run, which the form
// picks rather than asks. The window puts a reader on this form because of a
// directory, and this is what it picked.
func (s State) Starts() string { return s.repoPath }

// Flow is the flow the dial is on.
func (s State) Flow() string { return s.chosenFlow() }

// OnPills is whether the cursor is on a row of pills rather than in a field,
// which is what decides whether a key types or does something.
func (s State) OnPills() bool { return s.isPillField() }

// Type puts text where the caret is, which is what a paste is.
func (s State) Type(text string) State {
	return s.composeEdit(func(in *typing.Field) { in.Insert(text) })
}

// Write puts a flow's name on the dial, for the designer coming back with
// one the reader just made.
func (s *State) Write(name string) { s.setFlow(name) }

// Refresh reads the flows again, for a designer that has just added one.
func (s *State) Refresh(src flow.Source) { s.refreshFlows(src) }

// Env is what this screen needs of the world: the words it speaks, the keys
// it answers, the room it has, where flows are kept, and the checkouts a
// task can be started in. It is handed no board — the one thing it wants
// from one, which repository the reader was looking at, arrives as a name.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	Frame layout.Frame
	Now   time.Time
	// Flows is where the flows are kept, for the dial at the top of the
	// form.
	Flows flow.Source
	// Places is where work can be started: the checkouts Orbit knows, in
	// the order it found them.
	Places []Place
	// Autopilot is whether the board is running itself, which is what the
	// button at the foot of the form says will happen to a task saved here.
	Autopilot bool
	// ValidID says whether an id can be written, and why not when it
	// cannot. Nil in a build with no store to ask.
	ValidID func(id string) error
	// Read fetches an issue's body from its tracker. A URL names an issue;
	// only the body says what the work is.
	Read func(iss tracker.Issue) (tracker.Issue, error)
}

// A Place is a checkout a task can be started in: what it is called and
// where it is. A repository with no path is one the board knows only by
// name — somewhere work happened, not somewhere work can be started.
type Place struct {
	Name string
	Path string
}

// Out is what the screen asks the window for, having done what it can.
type Out struct {
	// Said is a sentence for the band.
	Said string
	// Leave is the reader closing the form.
	Leave bool
	// Write is the task to write down, and nothing when there is none.
	Write *Task
	// Flow is a flow to open the designer on: its name, or New for one
	// nobody has written yet.
	Flow string
	// Waiting is a question that has gone out to a tracker: the window
	// starts the frame clock, because the spinner is the window's.
	Waiting bool
	Cmd     tea.Cmd
}

// New is what Flow says when the reader asked for a flow that does not exist
// yet, rather than one of the ones that do.
const New = "\x00new"

// A Task is the form's answer: what to write down, and whether to start it
// now. The window turns it into the command that writes it, because what a
// command does is the window's business and not this form's.
type Task struct {
	ID    string
	Repo  string
	Flow  string
	Text  string
	Start bool
}

// said is one sentence and nothing else.
func said(text string) Out { return Out{Said: text} }

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}
