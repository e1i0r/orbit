package menu

// Every gesture the menu answers: the keyboard, the pointer's choice, the
// wheel.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Key answers the keyboard while the menu is up: pick, choose, leave. Every
// other key does nothing rather than reaching past a menu the reader is
// looking at.
//
// Leaving from a submenu goes back to the top instead: families are one
// level deep, so back always knows where it goes.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	// Inside a task, the key a pane is on opens it from here too — the
	// menu is where a reader goes to find out which key that is.
	if e.Detail {
		if p, ok := e.pane(msg.String()); ok {
			return s, Out{Leave: true, Pane: p}
		}
	}

	switch {
	case key.Matches(msg, e.Keys.Back):
		if s.sub != "" {
			return s.moved("", e), Out{}
		}

		return s, Out{Leave: true}
	case key.Matches(msg, e.Keys.Open):
		return s.choose(e)
	case msg.String() == "up" || key.Matches(msg, e.Keys.Up):
		return s.pick(-1, e), Out{}
	case msg.String() == "down" || key.Matches(msg, e.Keys.Down):
		return s.pick(1, e), Out{}
	}

	return s, Out{}
}

// Choose acts on the entry a click landed on, or moves the cursor to it
// first.
//
// The entry is found by what identifies it — glyph for a verb, name for a
// command — never by where it sat when the button went down: the list is
// recomputed between press and release, and an index could point at a
// different row by then.
func (s State) Choose(id string, e Env) (State, Out) {
	for i, entry := range s.entries(e) {
		if ident(entry) != id {
			continue
		}

		if i == s.sel {
			return s.choose(e)
		}

		return s.Point(i), Out{}
	}

	return s, Out{}
}

// Enter acts on the selection, which is what the key that opens a thing
// does here and what a second click on a row already chosen does.
func (s State) Enter(e Env) (State, Out) { return s.choose(e) }

// Wheel moves the selection, which is what scrolling a list with one row
// chosen means. It moves several rows at a time, and one of those jumps
// lands on the line between two blocks.
func (s State) Wheel(d int, e Env) State { return s.pick(d, e) }

// choose acts on the selection: an entry answers with the pane it opens,
// the command it runs, or the keystroke its binding names — which the
// window puts through the same map a pressed key goes through, so a refused
// verb refuses here exactly as it does from the keyboard.
func (s State) choose(e Env) (State, Out) {
	es := s.entries(e)
	if s.sel < 0 || s.sel >= len(es) {
		return s, Out{}
	}

	entry := es[s.sel]
	if entry.Head {
		return s, Out{}
	}

	switch {
	case entry.Pane != "":
		return s, Out{Leave: true, Pane: entry.Pane}
	case entry.Family != "":
		return s.moved(entry.Family, e), Out{}
	case entry.Command != "":
		// A command that takes a message is handed the box rather than
		// run: the menu has nothing to fill the sentence in with.
		if entry.Says {
			return s, Out{
				Leave: true, Run: entry.Command, Child: entry.Child,
				Args: entry.Args, Ask: true,
			}
		}

		// One that takes arguments is handed the line with its name on
		// it: the menu chooses with no arguments at all, so running one
		// of these bare came back with the refusal, which reads as an
		// entry that is broken rather than as one in the wrong place.
		if entry.NeedsArgs && len(entry.Args) <= 1 {
			return s, Out{Leave: true, Palette: entry.Command + " " + entry.Child + " "}
		}

		return s, Out{
			Leave: true, Run: entry.Command, Child: entry.Child,
			Args: entry.Args,
		}
	}

	return s, Out{Leave: true, Send: entry.Glyph}
}

// moved changes the level the menu is on: the cursor on the first entry
// there is to choose, the way opening puts it.
func (s State) moved(sub string, e Env) State {
	s.sub = sub
	s.sel = max(0, choice(s.entries(e), 0, 1))
	s.offset = 0

	return s.keepSeen(e)
}

// pick moves the selection within the entries there are.
func (s State) pick(d int, e Env) State {
	es := s.entries(e)

	n := len(es)
	if n == 0 {
		s.sel = 0
		return s
	}

	// The step is a distance — the wheel moves several rows at once — and
	// the search for a row that can be chosen is one entry at a time in
	// the direction it was going.
	step := 1
	if d < 0 {
		step = -1
	}

	next := min(max(s.sel+d, 0), n-1)
	if at := choice(es, next, step); at >= 0 {
		next = at
	}

	s.sel = next

	return s.keepSeen(e)
}

// pane is the pane that keystroke opens, if it is one of them. The letters
// are matched either way round, because a reader with caps lock on is still
// asking for the same pane.
func (e Env) pane(k string) (string, bool) {
	for _, p := range e.Panes {
		if p.Key != "" && strings.EqualFold(p.Key, k) {
			return p.Key, true
		}
	}

	return "", false
}
