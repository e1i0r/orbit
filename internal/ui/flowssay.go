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
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

// flowDraftedMsg is what the engine answered, decoded.
type flowDraftedMsg struct {
	flow flow.Flow
	err  error
}

// sayKey is every key on this tab.
func (m Model) sayKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	st := &m.flows

	switch {
	case msg.Code == tea.KeyLeft:
		return m.turnSayEngine(-1), nil
	case msg.Code == tea.KeyRight:
		return m.turnSayEngine(1), nil
	case st.saying:
		// While the engine is out there is nothing to type into: what
		// comes back replaces the phases, and a sentence written in the
		// meantime would be a question nobody asked.
		return m, nil
	case msg.Code == tea.KeyEnter && (msg.Mod&tea.ModShift != 0 || msg.Mod&tea.ModAlt != 0):
		st.say += "\n"
		return m, nil
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

// turnSayEngine moves that choice along, and says which one it landed on:
// the draft costs a run, and the reader should know whose.
func (m Model) turnSayEngine(d int) Model {
	m.flows.sayEngine = nextOption(m.engineNames(), m.sayEngineName(), d)

	return m.say(m.opts.Words.T("flows.say_engine_now", "the draft will be asked of {engine}",
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

	engineName := m.sayEngineName()
	ask := m.opts.Draft

	// Said in the bar as well as on the tab: a gesture that starts
	// something the reader cannot see finishing is a gesture they press
	// twice.
	m = m.say(p.T("flows.say_sent", "asking {engine} for a draft — this takes as long as one of its runs",
		about("engine", engineName)))

	return m, func() tea.Msg {
		out, err := ask(engineName, flowDraftPrompt(said, m.engineNames()))
		if err != nil {
			return flowDraftedMsg{err: err}
		}

		fl, err := decodeDraft(out)

		return flowDraftedMsg{flow: fl, err: err}
	}
}

// drafted takes what came back into the fields, for the reader to check.
func (m Model) drafted(msg flowDraftedMsg) (Model, tea.Cmd) {
	p := m.opts.Words
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

	return m.say(p.T("flows.say_drafted", "drafted {n} phases — read them, change what is wrong, then save",
		about("n", fmt.Sprint(len(fl.Phases))))), nil
}

// decodeDraft reads the flow out of whatever the engine printed.
//
// The braces are found rather than the whole answer parsed: engines wrap
// JSON in prose and in fences however they feel like on the day, and a draft
// refused because the model said "here you go" first is a draft the reader
// has to ask for twice.
func decodeDraft(out string) (flow.Flow, error) {
	from := strings.Index(out, "{")
	to := strings.LastIndex(out, "}")

	if from < 0 || to <= from {
		return flow.Flow{}, fmt.Errorf("the engine answered with no flow in it: %s", firstLine(out))
	}

	body := []byte(out)
	raw := body[from : to+1]

	// A draft with no name of its own is given one rather than refused: the
	// name is what the reader types next anyway, and "the flow has no name"
	// tells them nothing about the flow they just asked for.
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return flow.Flow{}, fmt.Errorf("the engine's answer is not a flow: %w", err)
	}

	name, named := doc["name"].(string)
	if !named || strings.TrimSpace(name) == "" {
		name = "draft"
		doc["name"] = name

		filled, err := json.Marshal(doc)
		if err != nil {
			return flow.Flow{}, fmt.Errorf("read the engine's answer back: %w", err)
		}

		raw = filled
	}

	return flow.Decode(raw, name)
}

// firstLine is enough of an answer to say what went wrong without printing a
// page of it into a one-line note.
func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")

	return fit(line, 120)
}
