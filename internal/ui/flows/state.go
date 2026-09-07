package flows

// What the designer is holding: the list it is showing, the flow being
// edited, and the three things the tab that writes a flow in words is
// waiting on.
//
// It is beside the doors rather than in them because every one of them reads
// it, and because this is the file to open when the question is "what does
// this screen remember".

import (
	"time"

	"github.com/e1i0r/orbit/internal/flow"
)

// State is the designer: what is being listed, edited, drawn and waited on.
type State struct {
	from           From
	sel            int
	creating       bool
	isEditing      bool
	readOnly       bool
	showingDetail  bool
	isBuiltin      bool
	confirmDiscard bool
	confirmDelete  bool
	// engine is the one a phase this editor invents is born on. It is
	// carried here because ensurePhase is reached from cur(), which has no
	// Model to ask, and internal/flow refuses a phase that names no engine
	// — so it is this or a name made up in a package that cannot know one.
	engine string
	// listed and detail are what this screen is showing: read when the
	// screen opens, and again whenever it changes something.
	//
	// They are not read where they are drawn. View is called from
	// View, so reading there makes every frame of this screen one
	// os.ReadDir plus one os.ReadFile per flow, on the thread that draws —
	// and Hit would walk the very same directory again, on every mouse
	// event, to work out where the rows it had not drawn would be. Two
	// readings of one directory, taken at two moments, deciding the same
	// layout: a flow saved between the draw and the click moves every row
	// under the cursor, and the click lands on a different flow than the
	// one the reader was pointing at.
	listed      []flow.Listed
	detail      map[string]resolved
	field       int
	template    string
	flowName    string
	description string
	activePhase int
	phases      []flow.Phase
	// checksDraft is what has been typed into a loop's checks field,
	// checksTyped is whether anything has been, and checksFor is the phase
	// it was typed for. The three keep a half-written line from being
	// rewritten by its own parse: see setLoopChecks. A draft left behind on
	// another phase is ignored rather than cleared, so switching phases
	// needs no bookkeeping anywhere.
	checksDraft string
	checksTyped bool
	checksFor   int
	// picker is the list of choices while one is open over the form: see
	// flowspicker.go. scroll is the row of the form the window starts at,
	// for the times it is taller than the terminal.
	picker pickerState
	scroll int
	// tab is which of the designer's three views is open: see flowstabs.go.
	tab int
	// say is the sentence the third tab turns into a flow, saying is
	// whether an engine is out answering it, and sayNote is what the last
	// attempt said when it did not come back with one.
	say     string
	saying  bool
	sayNote string
	// sayEngine and sayModel are who that tab asks. Empty is the window's
	// own default and the engine's own default model, and both are held
	// apart from the phases' engines: which engine writes the draft and
	// which engine runs the flow are two decisions, and the second is one
	// the draft itself makes.
	//
	// sayFocus is which of the tab's three things the keys are aimed at.
	sayEngine string
	sayModel  string
	sayFocus  int
	// sayAt is when the question went out, for the count of seconds beside
	// the spinner, and sayID is which question is being waited on. The id
	// is what lets somebody stop waiting: an answer that arrives for a
	// question they walked away from is dropped rather than landing in a
	// form they have moved on to.
	sayAt time.Time
	sayID int
	// attempts is the flow's cap, carried although no field shows it: this
	// editor rebuilds the whole flow when it saves, so what it does not hold
	// is lost by opening a flow and saving it.
	attempts int
}

// ensurePhase gives an editor with no phases one, on the engine the editor
// was opened with.
//
// It names only the engine — internal/flow refuses a phase that names none —
// and leaves the model and the effort to whatever the run is set to. Naming a
// model here would pin every new flow to sonnet, which is claude's alone and
// which a build on another engine cannot run.
func (s *State) ensurePhase() {
	if len(s.phases) == 0 {
		s.phases = []flow.Phase{
			{Name: "1-implement", Engine: s.engine, Thinking: "adaptive", Permissions: []string{"repo"}},
		}
		s.activePhase = 0
	}

	if s.activePhase < 0 {
		s.activePhase = 0
	}

	if s.activePhase >= len(s.phases) {
		s.activePhase = len(s.phases) - 1
	}
}

func (s *State) cur() *flow.Phase {
	s.ensurePhase()
	return &s.phases[s.activePhase]
}
