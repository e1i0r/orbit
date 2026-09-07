package supervisor

// Every key this screen answers, and the scrolling they do.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/clip"
	"github.com/e1i0r/orbit/internal/ui/spoken"
)

// Key is one press. Which keyboard it goes to is the mode: the conversation
// list has its own, so has the picker that takes a line back, and so has the
// list of offers over an unfinished word.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	switch {
	case s.list:
		return s.conversationKey(msg, e)
	case s.picking:
		return s.pickingKey(msg, e)
	case len(s.completions(e)) > 0 && offering(msg):
		// A list is up over an unfinished word, so ↑↓ choose from it and ↵
		// finishes it rather than sending half a gesture. There is no key to
		// dismiss it and none is needed: a space ends the word, and the list
		// is only ever there while one is unfinished.
		return s.completionKey(msg, e), Out{}
	case msg.Code == tea.KeyEscape || key.Matches(msg, e.Keys.Back):
		return s.leave()
	case (msg.Code == 'r' || msg.Code == 'R') && msg.Mod&tea.ModCtrl != 0:
		return s.startPicking(e), Out{}
	case ctrlLetter(msg, 'l'):
		return s.openConversationList(), Out{}
	case ctrlLetter(msg, 'n'):
		return s.startConversation(e)
	case msg.Code == tea.KeyUp:
		return s.Scroll(-1, e), Out{}
	case msg.Code == tea.KeyDown:
		return s.Scroll(1, e), Out{}
	case msg.Code == tea.KeyPgUp:
		return s.Scroll(-s.threadPage(e), e), Out{}
	case msg.Code == tea.KeyPgDown:
		return s.Scroll(s.threadPage(e), e), Out{}
	case msg.Code == tea.KeyHome:
		s.offset, s.follow = 0, false
		return s, Out{}
	case msg.Code == tea.KeyEnd:
		s.follow = true
		return s, Out{}
	case msg.Code == tea.KeyEnter || key.Matches(msg, e.Keys.Open):
		if msg.Mod&tea.ModShift != 0 || msg.Mod&tea.ModAlt != 0 {
			s.input += "\n"
			return s, Out{}
		}

		text := strings.TrimSpace(s.input)
		if text == "" {
			return s, Out{}
		}

		s.input = ""

		return s.Say(text, e)
	case (msg.Code == 'v' || msg.Code == 'V') && msg.Mod&tea.ModCtrl != 0:
		if pasted := clip.Read(); pasted != "" {
			s.input += pasted
		}

		return s, Out{}
	case msg.Code == tea.KeyBackspace || msg.Code == tea.KeyDelete:
		if len(s.input) > 0 {
			runes := []rune(s.input)
			s.input = string(runes[:len(runes)-1])
		}

		return s, Out{}
	case (msg.Code == 'u' || msg.Code == 'U') && msg.Mod&tea.ModCtrl != 0:
		s.input = ""
		return s, Out{}
	default:
		if msg.Text != "" {
			s.input += msg.Text
			return s, Out{}
		}
	}

	return s, Out{}
}

// leave closes the screen, and says which one it was opened from.
func (s State) leave() (State, Out) {
	return State{}, Out{Leave: true, Back: s.back}
}

// Say puts one line in the thread and asks the supervisor to answer it. It
// is a door because a caller with something of its own to say — the delivery
// keys, which send a line nobody typed — says it the same way.
func (s State) Say(text string, e Env) (State, Out) {
	// Four gestures share this one line, and which one was typed is read
	// before anything is sent: a rule is not a message the supervisor has to
	// interpret, it is a fact to write down. spoken.go is the whole grammar.
	if line := spoken.Parse(text); line.Kind != spoken.Message {
		// /brief is a question and not an action: it is sent the way a
		// typed sentence is, so the answer lands in the thread where the
		// person who asked will look for it.
		if line.Kind == spoken.Brief {
			return s.Say(s.briefQuestion(line.Phrase, e), e)
		}

		return s.act(line, e)
	}

	if e.Record != nil {
		// "operator" is who every other door writes, and the thread is one
		// conversation: a name hardcoded here made the same person read as
		// two participants depending on whether they typed in the window or
		// in a terminal, and put somebody else's name on the messages of
		// anyone who is not the author of this program.
		if err := e.Record(s.conversation, "operator", "tui", text); err != nil {
			return s, said(err.Error())
		}
	}

	s = s.Sync(e)
	s.follow = true

	if e.Ask == nil {
		return s, said(e.Words.T("supervisor.cannot_ask", "this window cannot ask the supervisor"))
	}

	return s, Out{
		Said:   e.Words.T("supervisor.thinking", "supervisor is thinking..."),
		Asking: true,
		Cmd:    e.Ask(s.conversation, text),
	}
}

// startPicking opens the mode that takes a turn back, on the last line said
// — which is the one somebody has just regretted nine times out of ten.
func (s State) startPicking(e Env) State {
	if len(s.lines) == 0 || e.Retract == nil {
		return s
	}

	s.picking = true
	s.pick = len(s.lines) - 1

	return s
}

// pickingKey is every key while a line is being picked.
func (s State) pickingKey(msg tea.KeyPressMsg, e Env) (State, Out) {
	switch {
	case msg.Code == tea.KeyEscape || key.Matches(msg, e.Keys.Back) ||
		((msg.Code == 'r' || msg.Code == 'R') && msg.Mod&tea.ModCtrl != 0):
		s.picking = false
		return s, Out{}
	case msg.Code == tea.KeyUp:
		s.pick = max(s.pick-1, 0)
		return s, Out{}
	case msg.Code == tea.KeyDown:
		s.pick = min(s.pick+1, len(s.lines)-1)
		return s, Out{}
	case msg.Code == tea.KeyEnter || key.Matches(msg, e.Keys.Open):
		return s.retractPicked(e)
	}

	return s, Out{}
}

// retractPicked takes back the line under the cursor.
func (s State) retractPicked(e Env) (State, Out) {
	s.picking = false
	if e.Retract == nil || s.pick >= len(s.lines) {
		return s, Out{}
	}

	l := s.lines[s.pick]
	if l.Retracted {
		return s, said(e.Words.T("supervisor.already_back", "that line was already taken back"))
	}

	if err := e.Retract(l.At); err != nil {
		return s, said(err.Error())
	}

	return s.Sync(e), said(e.Words.T("supervisor.took_back",
		"took that line back; the supervisor is no longer told it"))
}

// Wheel is the mouse doing what the arrows do: while a line is being picked
// it moves the pick, so the hand does not have to know which of the two it
// is holding.
func (s State) Wheel(d int, e Env) State {
	if s.picking {
		s.pick = min(max(s.pick+d, 0), max(len(s.lines)-1, 0))

		return s
	}

	return s.Scroll(d, e)
}

// Scroll moves the thread by d rows and lands somewhere real.
//
// The clamp is here, at the press, and not only at the drawing. An offset
// allowed to run past either end is what makes a scroll feel broken: the
// number keeps moving while the screen does not, and then the first ten
// presses of the other arrow appear to do nothing while it walks back.
//
// Reaching the end is what turns following back on, so a reader who scrolls
// down to the newest message is carried by the ones that arrive after it,
// and a reader who has scrolled up is left where they were reading.
func (s State) Scroll(d int, e Env) State {
	total, rows := s.threadSize(e)
	last := max(total-rows, 0)

	offset := s.offset
	if s.follow {
		offset = last
	}

	offset = min(max(offset+d, 0), last)
	s.offset = offset
	s.follow = offset >= last

	return s
}

// threadSize is how many rows the conversation is and how many are on
// screen, asked of the same functions that draw it.
func (s State) threadSize(e Env) (total, shown int) {
	cw, threadH := s.layout(max(e.Frame.Body.H, 1), max(e.Frame.Body.W, 1), e)
	rows, _ := s.threadLines(cw, e)

	return len(rows), max(threadH, 1)
}

// threadPage is one press of page up or down: a screenful less one row, so
// that the line you were reading is still there to pick the thread up from.
func (s State) threadPage(e Env) int {
	_, shown := s.threadSize(e)

	return max(shown-1, 1)
}
