package ui

// The designer's third tab: say what the flow should do, and let an engine
// write the first draft of it.
//
// What comes back is never saved. It lands in the same fields the other two
// tabs edit, and the reader looks at it, changes what is wrong and presses
// Save — because a flow is a standing instruction that will spend money on
// every task written against it, and one nobody read is one nobody agreed
// to.

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

// flowDraftedMsg is what the engine answered, decoded.
type flowDraftedMsg struct {
	id   int
	flow flow.Flow
	err  error
	// mended is whether the engine had to be asked a second time, because
	// what it wrote first was not JSON. It is said out loud: two runs were
	// paid for, and a draft that needed mending is one to read twice.
	mended bool
}

// The three things on the say tab, in the order tab moves between them.
const (
	sayOnEngine = iota
	sayOnModel
	sayOnText
	sayThings
)

// sayKey is every key on this tab.
func (m Model) sayKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	st := &m.flows

	switch {
	case msg.Code == tea.KeyTab && msg.Mod&tea.ModShift == 0:
		st.sayFocus = (st.sayFocus + 1) % sayThings
		return m, nil
	case msg.Code == tea.KeyTab || msg.Code == tea.KeyUp:
		st.sayFocus = (st.sayFocus - 1 + sayThings) % sayThings
		return m, nil
	case msg.Code == tea.KeyDown:
		st.sayFocus = (st.sayFocus + 1) % sayThings
		return m, nil
	case msg.Code == tea.KeyLeft:
		return m.turnSayDial(-1), nil
	case msg.Code == tea.KeyRight:
		return m.turnSayDial(1), nil
	case st.saying:
		// While the engine is out there is nothing to type into: what
		// comes back replaces the phases, and a sentence written in the
		// meantime would be a question nobody asked.
		return m, nil
	case msg.Code == tea.KeyEnter && (msg.Mod&tea.ModShift != 0 || msg.Mod&tea.ModAlt != 0):
		st.say += "\n"
		return m, nil
	case msg.Code == tea.KeyEnter && st.sayFocus == sayOnEngine:
		return m.openPicker(flowFieldSayEngine), nil
	case msg.Code == tea.KeyEnter && st.sayFocus == sayOnModel:
		return m.openPicker(flowFieldSayModel), nil
	case msg.Code == tea.KeyEnter:
		return m.draftFlow()
	case msg.Code == tea.KeyBackspace:
		st.say = trimLastRune(st.say)
		return m, nil
	case (msg.Code == 'v' || msg.Code == 'V') && msg.Mod&tea.ModCtrl != 0:
		st.say += strings.TrimRight(readClipboard(), "\r\n")
		return m, nil
	}

	if msg.Text != "" {
		st.say += msg.Text
	}

	return m, nil
}

// sayEngineName is the engine this tab asks: the one chosen on it, or the
// window's own when nobody has chosen.
func (m Model) sayEngineName() string {
	return m.dialEngine(m.flows.sayEngine)
}

// sayModelName is the model that engine is asked on, which is its own
// default until somebody picks one.
func (m Model) sayModelName() string {
	return m.flows.sayModel
}

// turnSayDial moves whichever of the two dials the reader is on, and says
// where it landed: the draft costs a run, and they should know whose.
//
// Changing the engine forgets the model, because a model is one engine's own
// name for it — opus is claude's, and agy has never heard of it.
func (m Model) turnSayDial(d int) Model {
	p := m.opts.Words

	if m.flows.sayFocus == sayOnModel {
		mdls, _ := m.modelsFor(m.sayEngineName())
		m.flows.sayModel = nextOption(append([]string{""}, mdls...), m.flows.sayModel, d)

		return m.say(p.T("flows.say_model_now", "the draft will be asked on {model}",
			about("model", orDef(m.flows.sayModel, p.T("flows.dial_default", "default")))))
	}

	m.flows.sayEngine = nextOption(m.engineNames(), m.sayEngineName(), d)
	m.flows.sayModel = ""

	return m.say(p.T("flows.say_engine_now", "the draft will be asked of {engine}",
		about("engine", m.sayEngineName())))
}

// draftFlow sends what was written to the engine.
func (m Model) draftFlow() (Model, tea.Cmd) {
	p := m.opts.Words

	said := strings.TrimSpace(m.flows.say)
	if said == "" {
		return m.say(p.T("flows.say_empty", "say what the flow should do first")), nil
	}

	if m.opts.Draft == nil {
		return m.say(p.T("flows.say_no_engine", "this build cannot ask an engine for a draft")), nil
	}

	m.flows.saying = true
	m.flows.sayNote = ""
	m.flows.sayAt = time.Now()
	m.flows.sayID++

	id := m.flows.sayID

	engineName, model := m.sayEngineName(), m.sayModelName()
	ask := m.opts.Draft

	// Said in the bar as well as on the tab: a gesture that starts
	// something the reader cannot see finishing is a gesture they press
	// twice.
	m = m.say(p.T("flows.say_sent", "asking {engine}…", about("engine", engineName)))

	// The frame clock is started here, the way the supervisor starts it:
	// the spinner beside the count of seconds is the difference between a
	// question that is out and a window that is wedged.
	m, frame := m.nextFrame()

	engines := m.engineNames()

	send := func() tea.Msg {
		out, err := ask(engineName, model, flowDraftPrompt(said, engines))
		if err != nil {
			return flowDraftedMsg{id: id, err: err}
		}

		fl, err := decodeDraft(out)
		if err == nil {
			return flowDraftedMsg{id: id, flow: fl}
		}

		// One more ask, with the decoder's own complaint in front of it.
		//
		// A model writing JSON by hand puts a quotation mark inside a
		// string — "go test ./..." is what the person asked for, and the
		// engine repeats it — and from there the document is broken in a
		// way no repair here can tell from a string that was never closed.
		// The engine that wrote it is the one thing that knows what it
		// meant, so it is asked, once, rather than the reader being handed
		// a decoder error about a field they never typed.
		out, retryErr := ask(engineName, model, mendDraftPrompt(out, err))
		if retryErr != nil {
			return flowDraftedMsg{id: id, err: err}
		}

		fl, err = decodeDraft(out)

		return flowDraftedMsg{id: id, flow: fl, err: err, mended: true}
	}

	return m, tea.Batch(send, frame)
}

// stopWaiting leaves the question out there and gives the screen back.
//
// Orbit did not spawn the engine with a handle it can kill — the port hands
// back an answer or an error, and nothing in between — so this is honest
// about what it does: it stops waiting, and the answer is dropped if it ever
// comes.
func (m Model) stopWaiting() Model {
	m.flows.saying = false
	m.flows.sayID++

	return m.say(m.opts.Words.T("flows.say_stopped", "stopped waiting; the answer will be dropped if it lands"))
}

// waitedFor is how long the question has been out, for the line that says so.
func (m Model) waitedFor() time.Duration {
	if m.flows.sayAt.IsZero() {
		return 0
	}

	return m.now.Sub(m.flows.sayAt).Round(time.Second)
}

// drafted takes what came back into the fields, for the reader to check.
func (m Model) drafted(msg flowDraftedMsg) (Model, tea.Cmd) {
	p := m.opts.Words

	// An answer to a question nobody is waiting on any more: the reader
	// pressed escape, or asked again, and what lands now would overwrite a
	// form they have gone back to editing by hand.
	if !m.flows.saying || msg.id != m.flows.sayID {
		return m, nil
	}

	m.flows.saying = false

	if msg.err != nil {
		m.flows.sayNote = msg.err.Error()

		return m.say(p.T("flows.say_failed", "no draft: {err}", about("err", firstLine(msg.err.Error())))), nil
	}

	fl := msg.flow
	if len(fl.Phases) == 0 {
		m.flows.sayNote = p.T("flows.say_no_phases", "the engine answered with no phases")

		return m.say(m.flows.sayNote), nil
	}

	m.flows.phases = fl.Phases
	m.flows.activePhase = 0
	m.flows.checksTyped = false
	m.flows.description = fl.Description

	// The name it chose is taken only when there is none: a reader who has
	// already named the flow they are editing has said what it is called.
	if strings.TrimSpace(m.flows.flowName) == "" {
		m.flows.flowName = fl.Name
	}

	m.flows.tab = flowTabFields
	m.flows.field = flowFieldPhaseSelect
	m.flows.scroll = 0

	if msg.mended {
		return m.say(p.T("flows.say_drafted_mended",
			"drafted {n} phases, after asking twice — its first answer was not JSON. Read them closely",
			about("n", fmt.Sprint(len(fl.Phases))))), nil
	}

	return m.say(p.T("flows.say_drafted", "drafted {n} phases — read them, change what is wrong, then save",
		about("n", fmt.Sprint(len(fl.Phases))))), nil
}

// decodeDraft is internal/flow's reader, under the name this screen calls
// it by. Getting a flow back out of a model's answer is about flows, and
// living there keeps this package free of the byte-counting that reading
// JSON needs and drawing a terminal forbids.
func decodeDraft(out string) (flow.Flow, error) { return flow.Draft(out) }

// firstLine is enough of an answer to say what went wrong without printing a
// page of it into a one-line note.
func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")

	return fit(line, 120)
}
