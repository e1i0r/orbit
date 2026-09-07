package flows

// Every key of the designer's form.
//
// It is its own file because the form is the half of this screen that takes
// input, and flows.go is the half that holds the state and opens the
// screens. What each key does to a field is in flowsfields.go.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/clip"
)

func (s State) flowsFormKey(msg tea.KeyPressMsg, e Env) (State, Out) {
	s.ensurePhase()

	if s.picker.open {
		return s.pickerKey(msg, e)
	}

	if d, held := flowTabKey(msg); held {
		return s.moveFlowTab(d), Out{}
	}

	if s.saying && (msg.Code == tea.KeyEscape || key.Matches(msg, e.Keys.Back)) {
		return s.stopWaiting(e)
	}

	p := e.Words

	if s.confirmDiscard {
		switch {
		case msg.Text == "y" || msg.Text == "Y" || msg.Text == "s" || msg.Text == "S" || key.Matches(msg, e.Keys.Open) || key.Matches(msg, e.Keys.Back):
			s.creating = false
			s.confirmDiscard = false

			return s, Out{Said: p.T("flows.changes_discarded", "changes discarded")}
		default:
			s.confirmDiscard = false
			return s, Out{Said: p.T("flows.editing_resumed", "editing resumed")}
		}
	}

	isText := s.typing()
	switch {
	case key.Matches(msg, e.Keys.Back):
		if s.flowName != "" || s.description != "" || len(s.phases) > 1 || s.cur().Prompt != "" {
			s.confirmDiscard = true

			return s, Out{Said: p.T("flows.confirm_discard", "discard changes? [y/n]")}
		}

		s.creating = false

		return s, Out{}
	case s.tab == flowTabSay:
		return s.sayKey(msg, e)
	case s.tab == flowTabDiagram:
		// The diagram is read, not typed into: the only keys it has are the
		// ones already taken above, and the mouse, which selects a phase.
		return s, Out{}
	case (msg.Code == tea.KeyTab && msg.Mod&tea.ModShift == 0) || msg.Code == tea.KeyDown || (!isText && key.Matches(msg, e.Keys.Down)):
		// The window's Up and Down are also bound to k and j, which is
		// right on a board and wrong in a field: typing "make cover" moved
		// the cursor twice and left "ma" and "e cover" in two other fields.
		// While something is being typed into, a letter is a letter.
		s.moveField(1)

		return s.followField(e), Out{}
	case key.Matches(msg, e.Keys.PrevTab) || msg.Code == tea.KeyUp || (!isText && key.Matches(msg, e.Keys.Up)):
		s.moveField(-1)
		return s.followField(e), Out{}
	case !isText && (msg.Code == tea.KeyLeft || msg.Code == tea.KeyRight):
		delta := 1
		if msg.Code == tea.KeyLeft {
			delta = -1
		}

		return s.handleFlowFieldDelta(delta, e)
	case msg.Code == tea.KeyEnter && (msg.Mod&tea.ModShift != 0 || msg.Mod&tea.ModAlt != 0):
		// Shift+Enter is a new line and not a submit, the way it is in the
		// compose form and in the supervisor's line: a phase's instructions
		// are a paragraph, and a field that cannot hold one sends the reader
		// to the JSON file this screen exists to replace.
		if s.multiline() {
			s.Write("\n")
		}

		return s, Out{}
	case key.Matches(msg, e.Keys.Open) || (!isText && msg.Text == " "):
		return s.handleFlowFieldAction(e)
	case (msg.Code == 'v' || msg.Code == 'V') && msg.Mod&tea.ModCtrl != 0:
		// ^V as well as cmd+V: the second arrives as a paste message the
		// terminal wraps, which is off in some terminals and never happens
		// over ssh at all.
		if clip := strings.TrimRight(clip.Read(), "\r\n"); clip != "" {
			s.Write(clip)
		}

		return s, Out{}
	case msg.Code == tea.KeyBackspace:
		s.rub()
		return s, Out{}
	}

	if msg.Text != "" {
		s.Write(msg.Text)
	}

	return s, Out{}
}

// Write puts what was typed at the end of the field under the cursor.
func (s *State) Write(text string) {
	switch s.field {
	case flowFieldName:
		s.flowName += text
	case flowFieldDescription:
		s.description += text
	case flowFieldPhaseName:
		s.cur().Name += text
	case flowFieldPrompt:
		s.edited().Prompt += text
	case flowFieldLoopUntil:
		s.setChecks(s.loopChecksText() + text)
	}
}
