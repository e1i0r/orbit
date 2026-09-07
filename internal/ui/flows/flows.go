package flows

import (
	"slices"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// about names a value and what the sentence calls it. It is the window's
// shorthand, kept here as well because a screen writes as many sentences as
// the window does.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}

const (
	flowFieldTemplate = iota
	flowFieldName
	flowFieldDescription
	flowFieldPhaseSelect
	flowFieldPhaseName
	flowFieldIsLoop
	flowFieldLoopTurns
	flowFieldLoopUntil
	flowFieldEngine
	flowFieldModel
	flowFieldEffort
	flowFieldThinking
	flowFieldFeedOutput
	flowFieldWait
	flowFieldPrompt
	flowFieldAddPhase
	flowFieldDelPhase
	flowFieldSave
	flowFieldCount

	// The say tab's two dials. They are past the count on purpose: they are
	// not fields of the form — tab does not stop on them and fieldsShown
	// never lists them — but the picker is told which dial it is choosing
	// for, and this is that name.
	flowFieldSayEngine
	flowFieldSayModel
)

// Env is what this screen needs of the world: the words it speaks, the keys
// it answers, the room it has, where flows are kept, and the engine it can
// ask to draft one. Nothing else — a screen that could reach the board could
// decide something about a task, and this one only edits flows.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	Frame layout.Frame
	Flows flow.Source
	Now   time.Time
	// Spinner is the frame the window's own clock is on. It is borrowed
	// rather than kept because a second clock would turn it twice as fast,
	// and what is animated on this screen is one question going out.
	Spinner func(theme.Role) string
	// Draft asks an engine one question and nothing else: the third tab
	// turns a sentence into a flow with it.
	Draft func(engineName, model, prompt string) (string, error)
	// Engine is the one a run uses when nothing names another: the dials
	// fall back to it, and a phase this editor invents is born on it.
	Engine string
	// Engines, Models and Efforts are the build's catalogue, for the dials.
	Engines func() []string
	Models  func(engine string) (ids, labels []string)
	Efforts func(engine string) (ids, labels []string)
}

// Out is what the screen asks the window for, having done what it can itself.
type Out struct {
	// Said is a sentence for the band.
	Said string
	// Leave is the reader closing the designer; Back says where to.
	Leave bool
	Back  From
	// Chose is the flow the reader picked, for the form that sent them here
	// to Write into itself.
	Chose string
	// Waiting is a question that has gone out to an engine: the window
	// starts the frame clock, because the spinner is the window's and two
	// of them would turn it twice as fast.
	Waiting bool
	Cmd     tea.Cmd
}

func said(text string) Out { return Out{Said: text} }

// From is the screen the designer was opened from, and so the one leaving it
// goes back to. It is a name of this package's own because the window's list
// of screens is the window's business.
type From int

// The three places the designer is reached from.
const (
	FromBoard From = iota
	FromCompose
	FromStart
)

// Open is the designer coming up, listing what is there. from says where the
// reader came from, because leaving goes back to it.
func Open(from From, e Env) State {
	s := State{
		from:     from,
		template: "ninguna",
		sel:      -1,
	}
	s.ensurePhase()
	s.refresh(e.Flows)

	return s
}

// resolved is one flow as this screen holds it: what Resolve answered, or
// the error when it could not. The error is kept because the list names a
// file that does not parse rather than hiding it — "there is a file called
// that" is what the reader is asking — and the row says why.
type resolved struct {
	flow flow.Flow
	err  error
}

// refresh reads the flow directory once, for the screen to draw from and to
// hit-test against.
func (s *State) refresh(src flow.Source) {
	s.listed = flow.List(src)
	s.detail = make(map[string]resolved, len(s.listed))

	for _, d := range s.listed {
		fl, err := flow.Resolve(src, d.Name)
		s.detail[d.Name] = resolved{flow: fl, err: err}
	}
}

// shown is the flow of that name as the screen last read it.
func (s *State) shown(name string) resolved {
	return s.detail[name]
}

// Preview opens one flow read-only, which is what the compose form asks for
// when somebody wants to see what a flow does before choosing it.
func Preview(name string, from From, e Env) State {
	fl, err := flow.Resolve(e.Flows, name)
	if err != nil {
		return Open(from, e)
	}

	s := State{
		from:          from,
		showingDetail: true,
		flowName:      name,
		description:   fl.Description,
		phases:        fl.Phases,
		attempts:      fl.Attempts,
		isBuiltin:     slices.Contains(flow.BuiltinNames(), name),
		activePhase:   0,
	}
	s.ensurePhase()

	return s
}

// leave closes the designer and says where the reader came from, so the
// window can put them back there.
func (s State) leave() (State, Out) {
	return State{}, Out{Leave: true, Back: s.from}
}

// Key answers one keystroke with the screen as it now is and whatever the
// window has to do about it.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	if s.creating {
		return s.flowsFormKey(msg, e)
	}

	if s.showingDetail {
		return s.flowDetailKey(msg, e)
	}

	return s.flowsListKey(msg, e)
}

func (s State) flowDetailKey(msg tea.KeyPressMsg, e Env) (State, Out) {
	switch {
	case key.Matches(msg, e.Keys.Back):
		if s.from == FromCompose {
			return s.leave()
		}

		s.showingDetail = false

		return s, Out{}
	case key.Matches(msg, e.Keys.Open):
		if s.from == FromCompose {
			next, out := s.leave()
			out.Chose = s.flowName

			return next, out
		}

		s.showingDetail = false

		return s, Out{}
	case msg.Text == "e" || msg.Text == "E":
		return s.editNamedFlow(s.flowName, e)
	}

	return s, Out{}
}

func (s State) flowsListKey(msg tea.KeyPressMsg, e Env) (State, Out) {
	p := e.Words

	if s.confirmDelete {
		switch {
		case msg.Text == "y" || msg.Text == "Y" || msg.Text == "s" || msg.Text == "S" || key.Matches(msg, e.Keys.Open):
			return s.confirmDeleteFlow(e)
		default:
			s.confirmDelete = false
			return s, Out{Said: p.T("flows.deletion_cancelled", "deletion cancelled")}
		}
	}

	descriptors := s.listed

	switch {
	case key.Matches(msg, e.Keys.Back):
		return s.leave()
	case key.Matches(msg, e.Keys.Up):
		if s.sel > -1 {
			s.sel--
		}

		return s, Out{}
	case key.Matches(msg, e.Keys.Down):
		if s.sel < len(descriptors)-1 {
			s.sel++
		}

		return s, Out{}
	case key.Matches(msg, e.Keys.Start):
		return s.startCreateFlow(e), Out{}
	case key.Matches(msg, e.Keys.Open):
		if s.sel == -1 {
			return s.startCreateFlow(e), Out{}
		}

		if s.sel >= 0 && s.sel < len(descriptors) {
			return Preview(descriptors[s.sel].Name, s.from, e), Out{}
		}

		return s.editSelectedFlow(e)
	case msg.Text == "e" || msg.Text == "E":
		return s.editSelectedFlow(e)
	case msg.Text == "d" || msg.Text == "D":
		return s.deleteSelectedFlow(e)
	case msg.Text == "n" || msg.Text == "N":
		return s.startCreateFlow(e), Out{}
	}

	return s, Out{}
}
