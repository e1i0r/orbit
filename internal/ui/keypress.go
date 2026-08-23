package ui

// Every keystroke the window answers, and the verbs they raise.
//
// It is a file of its own because the 300-line ceiling would not have both
// this and the model in one, and because the split falls where a reader
// would put it anyway: ui.go is what the window is, this is what it can be
// asked to do. What a verb is allowed to do is not decided here — that is
// affordance.go, and this file only ever asks it.

import (
	"maps"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// confirmYes is the one keystroke that answers a question with yes.
//
// It is not translated, and that is deliberate: a keystroke is a key on a
// keyboard rather than a word in a sentence, and a catalogue that moved it
// to "s" for "sí" would move it off the key the prompt names. The prompt
// says which key it is, and that sentence is translated.
const confirmYes = "y"

// key routes one keystroke to whichever of the window's five modes has it.
//
// The order is the order things are on top of each other: a filter being
// typed swallows every letter, a question waiting for an answer takes the
// next key whatever it is, and the two screens below the board have their own
// small maps. Only what is left reaches the list.
func (m Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.filtering:
		return m.filterKey(msg)
	case m.confirm != confirmNone:
		return m.confirmKey(msg)
	case m.screen == screenStart:
		return m.startKey(msg)
	case m.screen == screenDetail:
		return m.detailKey(msg)
	}
	return m.listKey(msg)
}

// listKey is the board's own map.
//
// Every verb here goes through affordance first, so a key that the task
// under the cursor cannot take says why rather than doing nothing. Doing
// nothing is what a reader reads as a broken keyboard.
func (m Model) listKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		return m.move(-1), nil
	case key.Matches(msg, m.keys.Down):
		return m.move(1), nil
	case key.Matches(msg, m.keys.First):
		return m.moveTo(0), nil
	case key.Matches(msg, m.keys.Last):
		return m.moveTo(len(m.rows()) - 1), nil
	case key.Matches(msg, m.keys.PageUp):
		return m.move(-m.frame.Body.H), nil
	case key.Matches(msg, m.keys.PageDown):
		return m.move(m.frame.Body.H), nil
	case key.Matches(msg, m.keys.Open):
		return m.open()
	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		return m, nil
	case key.Matches(msg, m.keys.Autopilot):
		return m.autopilot(), nil
	case key.Matches(msg, m.keys.Pause):
		return m.verb(m.keys.Pause, "pause")
	case key.Matches(msg, m.keys.Resume):
		return m.verb(m.keys.Resume, "resume")
	case key.Matches(msg, m.keys.Hand):
		return m.handBack()
	case key.Matches(msg, m.keys.Cancel):
		return m.ask()
	case key.Matches(msg, m.keys.Take):
		return m.takeKey()
	case key.Matches(msg, m.keys.MarkRead):
		return m.markReadKey()
	case key.Matches(msg, m.keys.Ask):
		// The one verb that is only ever its own reason. Orbit cannot put a
		// question to an engine yet, so gesture refuses it and says so, and
		// nothing here pretends otherwise with a stub.
		_, next, _ := m.gesture(m.keys.Ask)
		return next, nil
	case key.Matches(msg, m.keys.Start):
		return m.openStart()
	case key.Matches(msg, m.keys.Help):
		return m.notBuilt(m.keys.Help), nil
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}
	return m, nil
}

// filterKey feeds the text input, which owns every key it is not given a
// reason to give up.
func (m Model) filterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.filtering, m.filter = false, ""
		return m.clampCursor(), nil
	case key.Matches(msg, m.keys.Open):
		// The filter stays applied and the keyboard goes back to the list.
		// Closing it and clearing it are two different gestures because
		// filtering to one repository and then working in it is the whole
		// point of having filtered.
		m.filtering = false
		return m.clampCursor(), nil
	case msg.Code == tea.KeyBackspace:
		m.filter = trimLastRune(m.filter)
		return m.clampCursor(), nil
	}
	if msg.Text != "" {
		m.filter += msg.Text
	}
	return m.clampCursor(), nil
}

// trimLastRune removes the last character of the filter, counting runes and
// never bytes: backspacing "café" a byte at a time leaves an invalid string
// on screen, which is the same mistake as measuring a column in bytes.
func trimLastRune(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	return string(runes[:len(runes)-1])
}

// confirmKey answers the one question the window ever asks.
//
// Anything that is not the confirming key is a no. A question that only a
// specific "no" closes is a question that traps a reader who has already
// looked away, and the safe answer to "shall I cancel this run" is no.
func (m Model) confirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	id := m.confirmID
	m.confirm, m.confirmID = confirmNone, ""
	if msg.String() != confirmYes {
		return m, nil
	}
	t, ok := m.task(id)
	if !ok {
		return m, nil
	}
	return m, control(m.opts.Control, t, "cancel")
}

// open is one key doing two things, and it is not an overload: on a band
// header it opens the band in place, on a row it opens the task, and the
// cursor is on exactly one of the two.
func (m Model) open() (tea.Model, tea.Cmd) {
	r, ok := m.selected()
	if !ok {
		return m, nil
	}
	if r.head {
		return m.expand(r.band).clampCursor(), nil
	}
	return m.openDetail(r.task)
}

// verb asks the command behind one key to write one word. Whether the key is
// allowed at all is gesture's answer, in gesture.go.
func (m Model) verb(b key.Binding, word string) (Model, tea.Cmd) {
	t, next, ok := m.gesture(b)
	if !ok {
		return next, nil
	}
	return next, control(next.opts.Control, t, word)
}

// ask opens the confirm in front of a cancel.
//
// Cancelling is the one gesture here that cannot be undone by pressing
// something else — a run that was ended did not keep going — so it is the
// one that asks first.
func (m Model) ask() (tea.Model, tea.Cmd) {
	t, next, ok := m.gesture(m.keys.Cancel)
	if !ok {
		return next, nil
	}
	next.confirm, next.confirmID = confirmCancel, t.ID
	return next, nil
}

// autopilot flips the standing switch and says which way it went.
//
// It says what it just did rather than what it undid. The program this
// replaces printed "autopilot was off" after turning it on, and the sentence
// is ambiguous in exactly the moment a reader needs it not to be.
func (m Model) autopilot() Model {
	if m.opts.Settings == nil {
		return m
	}
	on := !m.opts.Settings.Autopilot()
	if err := m.opts.Settings.SetAutopilot(on); err != nil {
		return m.say(err.Error())
	}
	if on {
		return m.say(m.opts.Words.T("msg.autopilot_on", "autopilot is on: every phase runs without asking"))
	}
	return m.say(m.opts.Words.T("msg.autopilot_off", "autopilot is off: every phase stops for you"))
}

// notBuilt answers a key the bar offers and this window does not implement.
//
// One key is in that state — the help overlay is the next task — and the bar
// shows it because the screen this window is specified as shows it. Saying so
// is the honest half of that: a key that silently does nothing is
// indistinguishable from a key that is broken, and this sentence is one
// commit long.
func (m Model) notBuilt(b key.Binding) Model {
	return m.say(m.opts.Words.T("msg.not_built", "{key} is not wired up yet; this window is still being built",
		about("key", b.Help().Key)))
}

// conditions is the standing state the verbs are asked about, for one task.
//
// It takes the task because two of the three answers are about a particular
// one: whether this window handed the terminal to an engine for it, and
// whether the engine that ran it can carry a session on at all. Only the
// autopilot switch is about the whole program.
func (m Model) conditions(t view.Task) Conditions {
	return Conditions{
		Autopilot: m.autopilotOn(),
		CanResume: m.canResume(t.Engine),
		Taken:     m.taken[t.ID],
	}
}

// canResume asks the port about one engine by name, and answers no for a
// window that was handed no way to ask — a rendering test, or a window built
// before the composition root knows what it can run.
func (m Model) canResume(engine string) bool {
	return m.opts.CanResume != nil && m.opts.CanResume(engine)
}

// autopilotOn reads the switch, and answers for a window opened without a
// settings file at all — which is what a rendering test hands it.
func (m Model) autopilotOn() bool {
	return m.opts.Settings != nil && m.opts.Settings.Autopilot()
}

// unreadCap is how many unread finished tasks may stand before nothing new
// starts, and zero when there is no settings file to ask. Whether the brake
// is actually on is atUnreadCap, beside the refusal it produces.
func (m Model) unreadCap() int {
	if m.opts.Settings == nil {
		return 0
	}
	return m.opts.Settings.UnreadCap()
}

// affordance finds one verb's answer for one task, by the glyph its binding
// prints. The glyph is the same in every language, which is what lets this
// match a binding the key map may have rebuilt since.
func (m Model) affordance(t view.Task, b key.Binding) (Affordance, bool) {
	for _, a := range m.keys.Affordances(t, m.conditions(t)) {
		if a.Key.Help().Key == b.Help().Key {
			return a, true
		}
	}
	return Affordance{}, false
}

// task finds one task on the board by id.
func (m Model) task(id string) (view.Task, bool) {
	for _, t := range m.board.Tasks {
		if t.ID == id {
			return t, true
		}
	}
	return view.Task{}, false
}

// expand toggles one band open or shut, on a copy of the map.
func (m Model) expand(b view.Band) Model {
	open := maps.Clone(m.expanded)
	open[b] = !open[b]
	m.expanded = open
	return m
}
