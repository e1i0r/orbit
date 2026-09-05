package ui

import (
	"slices"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

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

type flowsState struct {
	fromScreen     screen
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
	// They are not read where they are drawn. flowsRows is called from
	// View, so reading there makes every frame of this screen one
	// os.ReadDir plus one os.ReadFile per flow, on the thread that draws —
	// and hitFlows would walk the very same directory again, on every mouse
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
func (st *flowsState) ensurePhase() {
	if len(st.phases) == 0 {
		st.phases = []flow.Phase{
			{Name: "1-implement", Engine: st.engine, Thinking: "adaptive", Permissions: []string{"repo"}},
		}
		st.activePhase = 0
	}

	if st.activePhase < 0 {
		st.activePhase = 0
	}

	if st.activePhase >= len(st.phases) {
		st.activePhase = len(st.phases) - 1
	}
}

func (st *flowsState) cur() *flow.Phase {
	st.ensurePhase()
	return &st.phases[st.activePhase]
}

func (m Model) openFlows() Model {
	prev := m.screen
	m.screen = screenFlows
	m.flows = flowsState{
		fromScreen: prev,
		template:   "ninguna",
		sel:        -1,
	}
	m.flows.ensurePhase()
	m.flows.refresh(m.opts.Flows)

	return m
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
func (st *flowsState) refresh(src flow.Source) {
	st.listed = flow.List(src)
	st.detail = make(map[string]resolved, len(st.listed))

	for _, d := range st.listed {
		fl, err := flow.Resolve(src, d.Name)
		st.detail[d.Name] = resolved{flow: fl, err: err}
	}
}

// shown is the flow of that name as the screen last read it.
func (st *flowsState) shown(name string) resolved {
	return st.detail[name]
}

func (m Model) openFlowPreview(name string) Model {
	fl, err := flow.Resolve(m.opts.Flows, name)
	if err != nil {
		return m.openFlows()
	}

	prev := m.screen
	m.screen = screenFlows
	m.flows = flowsState{
		fromScreen:    prev,
		showingDetail: true,
		flowName:      name,
		description:   fl.Description,
		phases:        fl.Phases,
		attempts:      fl.Attempts,
		isBuiltin:     slices.Contains(flow.BuiltinNames(), name),
		activePhase:   0,
	}
	m.flows.ensurePhase()

	return m
}

func (m Model) abandonFlows() Model {
	prev := m.flows.fromScreen

	m.flows = flowsState{}
	if prev == screenStart {
		m.screen = screenStart
		return m
	}

	if prev == screenCompose {
		m.screen = screenCompose
		m.compose.refreshFlows(m.opts.Flows)

		return m
	}

	m.screen = screenList

	return m
}

func (m Model) flowsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.flows.creating {
		return m.flowsFormKey(msg)
	}

	if m.flows.showingDetail {
		return m.flowDetailKey(msg)
	}

	return m.flowsListKey(msg)
}

func (m Model) flowDetailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		if m.flows.fromScreen == screenCompose {
			return m.abandonFlows(), nil
		}

		m.flows.showingDetail = false

		return m, nil
	case key.Matches(msg, m.keys.Open):
		if m.flows.fromScreen == screenCompose {
			m.compose.setFlow(m.flows.flowName)
			return m.abandonFlows(), nil
		}

		m.flows.showingDetail = false

		return m, nil
	case msg.Text == "e" || msg.Text == "E":
		return m.editNamedFlow(m.flows.flowName)
	}

	return m, nil
}

func (m Model) flowsListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	st := &m.flows
	p := m.opts.Words

	if st.confirmDelete {
		switch {
		case msg.Text == "y" || msg.Text == "Y" || msg.Text == "s" || msg.Text == "S" || key.Matches(msg, m.keys.Open):
			return m.confirmDeleteFlow()
		default:
			st.confirmDelete = false
			return m.say(p.T("flows.deletion_cancelled", "deletion cancelled")), nil
		}
	}

	descriptors := st.listed

	switch {
	case key.Matches(msg, m.keys.Back):
		return m.abandonFlows(), nil
	case key.Matches(msg, m.keys.Up):
		if st.sel > -1 {
			st.sel--
		}

		return m, nil
	case key.Matches(msg, m.keys.Down):
		if st.sel < len(descriptors)-1 {
			st.sel++
		}

		return m, nil
	case key.Matches(msg, m.keys.Start):
		return m.startCreateFlow(), nil
	case key.Matches(msg, m.keys.Open):
		if st.sel == -1 {
			return m.startCreateFlow(), nil
		}

		if st.sel >= 0 && st.sel < len(descriptors) {
			return m.openFlowPreview(descriptors[st.sel].Name), nil
		}

		return m.editSelectedFlow()
	case msg.Text == "e" || msg.Text == "E":
		return m.editSelectedFlow()
	case msg.Text == "d" || msg.Text == "D":
		return m.deleteSelectedFlow()
	case msg.Text == "n" || msg.Text == "N":
		return m.startCreateFlow(), nil
	}

	return m, nil
}
