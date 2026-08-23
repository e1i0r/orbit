package ui

// The two gestures that are not one of the five control words, and the fact
// that makes a third of them honest.
//
// d moves the unread brake back by one; t hands the terminal to the engine
// and takes it back again. Neither goes through internal/task's control file
// — "read" is not a word a run understands, and an interactive session is
// not a thing a run is told about at all — so each has its own port and its
// own message, rather than a sixth and seventh word smuggled onto a wire
// that has five.
//
// *h releases the pause and nothing more.* A reader who took the keyboard,
// talked to the engine and pressed h has left a conversation behind, and the
// obvious next thought is that Orbit should file it back into the task's
// record as a note. It does not, and this is the place to say so: internal/
// task has no note mechanism, internal/ui may not append to a record at all,
// and inventing one here would be a third feature wearing a fourth's
// clothes. What h does is exactly what r does — write "continue" — with the
// window's own memory of the session cleared. The transcript of a forked
// session lives wherever the engine puts it, and finding it is the reader's.

import (
	"maps"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// gesture is the check every key that acts on a task makes before it acts:
// there is a task under the cursor, this verb knows about it, and the verb is
// not refused.
//
// The refusal is the affordance's own sentence, said verbatim. This is where
// the window is most tempted to have an opinion — it knows the task, it knows
// the rule, it could phrase it better — and the rule it knows is a copy of
// the one internal/task enforces, which is the copy that goes stale.
func (m Model) gesture(b key.Binding) (view.Task, Model, bool) {
	r, ok := m.selected()
	if !ok || r.head {
		return view.Task{}, m, false
	}
	a, ok := m.affordance(r.task, b)
	if !ok {
		return view.Task{}, m, false
	}
	if !a.OK {
		return view.Task{}, m.say(a.Why(m.opts.Words)), false
	}
	return r.task, m, true
}

// markReadKey is d: one finished task read, and the brake one notch looser.
func (m Model) markReadKey() (tea.Model, tea.Cmd) {
	t, next, ok := m.gesture(m.keys.MarkRead)
	if !ok {
		return next, nil
	}
	return next, markRead(next.opts.MarkRead, t)
}

// takeKey is t: build the session, and do not run it.
//
// Nothing is remembered here. The window learns that the keyboard was taken
// when the command comes back and the suspend is actually issued, which is
// the only moment the fact is true — a port that failed to find a session id
// must not leave a task the window believes somebody is sitting in front of.
func (m Model) takeKey() (tea.Model, tea.Cmd) {
	t, next, ok := m.gesture(m.keys.Take)
	if !ok {
		return next, nil
	}
	return next, takeSession(next.opts.Take, t)
}

// handBack is h: the pause released, and the window's memory of the session
// cleared so that h stops being offered.
func (m Model) handBack() (tea.Model, tea.Cmd) {
	t, next, ok := m.gesture(m.keys.Hand)
	if !ok {
		return next, nil
	}
	return next.took(t.ID, false), control(next.opts.Control, t, "continue")
}

// session is what the window does with a command line it asked for: suspend
// itself and hand the terminal over.
//
// tea.ExecProcess is the whole of the mechanism. The alternative — running
// the engine with its output piped into a pane — is not the same gesture: a
// session is a conversation, and a conversation needs the terminal, not a
// viewport with somebody else's key map in front of it.
func (m Model) session(msg sessionMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return m.say(msg.Err.Error()), nil
	}
	if msg.Cmd == nil {
		return m.say(m.opts.Words.T("msg.no_session", "{id} has no session to carry on",
			about("id", msg.ID))), nil
	}
	id := msg.ID
	return m.took(id, true), tea.ExecProcess(msg.Cmd, func(err error) tea.Msg {
		return sessionEndedMsg{ID: id, Err: err}
	})
}

// sessionEnded is the reader back from one, and it says what is still true:
// the run is still parked, and it is theirs until they hand it back.
func (m Model) sessionEnded(msg sessionEndedMsg) Model {
	if msg.Err != nil {
		return m.say(msg.Err.Error())
	}
	return m.say(m.opts.Words.T("msg.session_ended", "{id} is still stopped and still yours; press h to hand it back",
		about("id", msg.ID)))
}

// readSaid is what the band says about a task that was marked read.
func (m Model) readSaid(msg readMsg) string {
	if msg.Err != nil {
		return msg.Err.Error()
	}
	return m.opts.Words.T("msg.marked_read", "{id} is marked read", about("id", msg.ID))
}

// took records, or forgets, that this window handed the terminal to an
// engine for one task.
//
// *The window tracks this, and view.Task does not.* The fact has to live
// somewhere, because h without it is offered on every parked task — including
// one that was merely paused, where handing back a keyboard nobody took is a
// sentence that means nothing. The two places it could live are the record
// and this map, and the record is not available: appending to it means
// reaching internal/record, which arch.layers does not let this package do,
// and adding an event kind for "a person opened a session" would be a third
// feature in a task that has two.
//
// The consequence is stated rather than hidden: this fact dies with the
// window. Restart Orbit while a session is open and h is refused where it
// would have worked, while r — which releases the same pause — still does.
// That is the safe direction to be wrong in. The day the record carries the
// fact, this map is deleted and Conditions.Taken is filled from view.Task.
//
// It is cloned rather than written in place, for the reason Model's other
// maps are: a map field survives the copy Update makes, so mutating one
// changes the model the renderer was already handed.
func (m Model) took(id string, yes bool) Model {
	held := maps.Clone(m.taken)
	if held == nil {
		held = map[string]bool{}
	}
	if yes {
		held[id] = true
	} else {
		delete(held, id)
	}
	m.taken = held
	return m
}
