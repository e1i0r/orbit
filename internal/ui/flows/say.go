package flows

// The designer's third tab: say what the flow should do, and let an engine
// Write the first draft of it.
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
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/clip"
	"github.com/e1i0r/orbit/internal/ui/prompt"
)

// DraftedMsg is what the engine answered, decoded.
type DraftedMsg struct {
	id   int
	flow flow.Flow
	err  error //nolint:unused // read by Took, which is the only reader there is
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
func (s State) sayKey(msg tea.KeyPressMsg, e Env) (State, Out) {
	switch {
	case msg.Code == tea.KeyTab && msg.Mod&tea.ModShift == 0:
		s.sayFocus = (s.sayFocus + 1) % sayThings
		return s, Out{}
	case msg.Code == tea.KeyTab || msg.Code == tea.KeyUp:
		s.sayFocus = (s.sayFocus - 1 + sayThings) % sayThings
		return s, Out{}
	case msg.Code == tea.KeyDown:
		s.sayFocus = (s.sayFocus + 1) % sayThings
		return s, Out{}
	case msg.Code == tea.KeyLeft:
		return s.turnSayDial(-1, e)
	case msg.Code == tea.KeyRight:
		return s.turnSayDial(1, e)
	case s.saying:
		// While the engine is out there is nothing to type into: what
		// comes back replaces the phases, and a sentence written in the
		// meantime would be a question nobody asked.
		return s, Out{}
	case msg.Code == tea.KeyEnter && (msg.Mod&tea.ModShift != 0 || msg.Mod&tea.ModAlt != 0):
		s.say += "\n"
		return s, Out{}
	case msg.Code == tea.KeyEnter && s.sayFocus == sayOnEngine:
		return s.openPicker(flowFieldSayEngine, e), Out{}
	case msg.Code == tea.KeyEnter && s.sayFocus == sayOnModel:
		return s.openPicker(flowFieldSayModel, e), Out{}
	case msg.Code == tea.KeyEnter:
		return s.draftFlow(e)
	case msg.Code == tea.KeyBackspace:
		s.say = cells.TrimLastRune(s.say)
		return s, Out{}
	case (msg.Code == 'v' || msg.Code == 'V') && msg.Mod&tea.ModCtrl != 0:
		s.say += strings.TrimRight(clip.Read(), "\r\n")
		return s, Out{}
	}

	if msg.Text != "" {
		s.say += msg.Text
	}

	return s, Out{}
}

// sayEngineName is the engine this tab asks: the one chosen on it, or the
// window's own when nobody has chosen.
func (s State) sayEngineName(e Env) string {
	return cells.OrDef(s.sayEngine, e.Engine)
}

// sayModelName is the model that engine is asked on, which is its own
// default until somebody picks one.
func (s State) sayModelName() string {
	return s.sayModel
}

// turnSayDial moves whichever of the two dials the reader is on, and says
// where it landed: the draft costs a run, and they should know whose.
//
// Changing the engine forgets the model, because a model is one engine's own
// name for it — opus is claude's, and agy has never heard of it.
func (s State) turnSayDial(d int, e Env) (State, Out) {
	p := e.Words

	if s.sayFocus == sayOnModel {
		mdls, _ := e.Models(s.sayEngineName(e))
		s.sayModel = cells.NextOption(append([]string{""}, mdls...), s.sayModel, d)

		return s, said(p.T("flows.say_model_now", "the draft will be asked on {model}",
			about("model", cells.OrDef(s.sayModel, p.T("flows.dial_default", "default")))))
	}

	s.sayEngine = cells.NextOption(e.Engines(), s.sayEngineName(e), d)
	s.sayModel = ""

	return s, said(p.T("flows.say_engine_now", "the draft will be asked of {engine}",
		about("engine", s.sayEngineName(e))))
}

// AskingOf is the engine the draft went out to, for the line in the band
// that says what is being waited for.
func (s State) AskingOf(e Env) string { return s.sayEngineName(e) }

// draftFlow sends what was written to the engine.
func (s State) draftFlow(e Env) (State, Out) {
	p := e.Words

	said := strings.TrimSpace(s.say)
	if said == "" {
		return s, Out{Said: p.T("flows.say_empty", "say what the flow should do first")}
	}

	if e.Draft == nil {
		return s, Out{Said: p.T("flows.say_no_engine", "this build cannot ask an engine for a draft")}
	}

	s.saying = true
	s.sayNote = ""
	s.sayAt = time.Now()
	s.sayID++

	id := s.sayID

	engineName, model := s.sayEngineName(e), s.sayModelName()
	ask := e.Draft

	engines := e.Engines()

	send := func() tea.Msg {
		out, err := ask(engineName, model, prompt.FlowDraft(said, engines))
		if err != nil {
			return DraftedMsg{id: id, err: err}
		}

		fl, err := decodeDraft(out)
		if err == nil {
			return DraftedMsg{id: id, flow: fl}
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
		out, retryErr := ask(engineName, model, prompt.MendDraft(out, err))
		if retryErr != nil {
			return DraftedMsg{id: id, err: err}
		}

		fl, err = decodeDraft(out)

		return DraftedMsg{id: id, flow: fl, err: err, mended: true}
	}

	// Said in the bar as well as on the tab: a gesture that starts something
	// the reader cannot see finishing is a gesture they press twice. The
	// window starts its own spinner off Waiting — the frame clock is the
	// window's, and a screen that started one would have two running.
	return s, Out{
		Said:    p.T("flows.say_sent", "asking {engine}…", about("engine", engineName)),
		Waiting: true,
		Cmd:     send,
	}
}

// stopWaiting leaves the question out there and gives the screen back.
//
// Orbit did not spawn the engine with a handle it can kill — the port hands
// back an answer or an error, and nothing in between — so this is honest
// about what it does: it stops waiting, and the answer is dropped if it ever
// comes.
func (s State) stopWaiting(e Env) (State, Out) {
	s.saying = false
	s.sayID++

	return s, Out{Said: e.Words.T("flows.say_stopped", "stopped waiting; the answer will be dropped if it lands")}
}

// waitedFor is how long the question has been out, for the line that says so.
func (s State) waitedFor(e Env) time.Duration {
	if s.sayAt.IsZero() {
		return 0
	}

	return e.Now.Sub(s.sayAt).Round(time.Second)
}

// Took takes what came back into the fields, for the reader to check.
func (s State) Took(msg DraftedMsg, e Env) (State, Out) {
	p := e.Words

	// An answer to a question nobody is waiting on any more: the reader
	// pressed escape, or asked again, and what lands now would overwrite a
	// form they have gone back to editing by hand.
	if !s.saying || msg.id != s.sayID {
		return s, Out{}
	}

	s.saying = false

	if msg.err != nil {
		s.sayNote = msg.err.Error()

		return s, Out{Said: p.T("flows.say_failed", "no draft: {err}", about("err", firstLine(msg.err.Error())))}
	}

	fl := msg.flow
	if len(fl.Phases) == 0 {
		s.sayNote = p.T("flows.say_no_phases", "the engine answered with no phases")

		return s, Out{Said: s.sayNote}
	}

	s.phases = fl.Phases
	s.activePhase = 0
	s.checksTyped = false
	s.description = fl.Description

	// The name it chose is taken only when there is none: a reader who has
	// already named the flow they are editing has said what it is called.
	if strings.TrimSpace(s.flowName) == "" {
		s.flowName = fl.Name
	}

	s.tab = flowTabFields
	s.field = flowFieldPhaseSelect
	s.scroll = 0

	if msg.mended {
		return s, said(p.T("flows.say_drafted_mended",
			"drafted {n} phases, after asking twice — its first answer was not JSON. Read them closely",
			about("n", fmt.Sprint(len(fl.Phases)))))
	}

	return s, said(p.T("flows.say_drafted", "drafted {n} phases — read them, change what is wrong, then save",
		about("n", fmt.Sprint(len(fl.Phases)))))
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

	return cells.Fit(line, 120)
}
