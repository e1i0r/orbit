package ui

// Every key of the designer's form.
//
// It is its own file because the form is the half of this screen that takes
// input, and flows.go is the half that holds the state and opens the
// screens. What each key does to a field is in flowsfields.go.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m Model) flowsFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	st := &m.flows
	st.ensurePhase()

	if st.picker.open {
		return m.pickerKey(msg)
	}

	if d, held := flowTabKey(msg); held {
		return m.moveFlowTab(d), nil
	}

	p := m.opts.Words

	if st.confirmDiscard {
		switch {
		case msg.Text == "y" || msg.Text == "Y" || msg.Text == "s" || msg.Text == "S" || key.Matches(msg, m.keys.Open) || key.Matches(msg, m.keys.Back):
			st.creating = false
			st.confirmDiscard = false

			return m.say(p.T("flows.changes_discarded", "changes discarded")), nil
		default:
			st.confirmDiscard = false
			return m.say(p.T("flows.editing_resumed", "editing resumed")), nil
		}
	}

	isText := st.typing()
	switch {
	case key.Matches(msg, m.keys.Back):
		if st.flowName != "" || st.description != "" || len(st.phases) > 1 || st.cur().Prompt != "" {
			st.confirmDiscard = true

			return m.say(p.T("flows.confirm_discard",
				"discard flow changes? [y] yes / [n] no (or press Esc again)")), nil
		}

		st.creating = false

		return m, nil
	case st.tab == flowTabSay:
		return m.sayKey(msg)
	case st.tab == flowTabDiagram:
		// The diagram is read, not typed into: the only keys it has are the
		// ones already taken above, and the mouse, which selects a phase.
		return m, nil
	case (msg.Code == tea.KeyTab && msg.Mod&tea.ModShift == 0) || msg.Code == tea.KeyDown || (!isText && key.Matches(msg, m.keys.Down)):
		// The window's Up and Down are also bound to k and j, which is
		// right on a board and wrong in a field: typing "make cover" moved
		// the cursor twice and left "ma" and "e cover" in two other fields.
		// While something is being typed into, a letter is a letter.
		st.moveField(1)

		return m.followField(), nil
	case key.Matches(msg, m.keys.PrevTab) || msg.Code == tea.KeyUp || (!isText && key.Matches(msg, m.keys.Up)):
		st.moveField(-1)
		return m.followField(), nil
	case !isText && (msg.Code == tea.KeyLeft || msg.Code == tea.KeyRight):
		delta := 1
		if msg.Code == tea.KeyLeft {
			delta = -1
		}

		return m.handleFlowFieldDelta(delta)
	case msg.Code == tea.KeyEnter && (msg.Mod&tea.ModShift != 0 || msg.Mod&tea.ModAlt != 0):
		// Shift+Enter is a new line and not a submit, the way it is in the
		// compose form and in the supervisor's line: a phase's instructions
		// are a paragraph, and a field that cannot hold one sends the reader
		// to the JSON file this screen exists to replace.
		if st.multiline() {
			st.write("\n")
		}

		return m, nil
	case key.Matches(msg, m.keys.Open) || (!isText && msg.Text == " "):
		return m.handleFlowFieldAction()
	case (msg.Code == 'v' || msg.Code == 'V') && msg.Mod&tea.ModCtrl != 0:
		// ^V as well as cmd+V: the second arrives as a paste message the
		// terminal wraps, which is off in some terminals and never happens
		// over ssh at all.
		if clip := strings.TrimRight(readClipboard(), "\r\n"); clip != "" {
			st.write(clip)
		}

		return m, nil
	case msg.Code == tea.KeyBackspace:
		st.rub()
		return m, nil
	}

	if msg.Text != "" {
		st.write(msg.Text)
	}

	return m, nil
}
