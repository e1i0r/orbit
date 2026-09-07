package settings

// Key is every keystroke this screen answers, and the only one it lets
// through is the one that leaves.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
)

// Key answers one keystroke with the screen as it now is, and whatever the
// window has to do about it.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	rows := s.Rows(e)
	if len(rows) == 0 {
		if key.Matches(msg, e.Keys.Back) || key.Matches(msg, e.Keys.Quit) {
			return s, Out{Close: true}
		}

		return s, Out{}
	}

	if s.editing {
		return s.typing(msg, e, rows)
	}

	switch {
	case key.Matches(msg, e.Keys.Back) || key.Matches(msg, e.Keys.Quit):
		return s, Out{Close: true}
	case key.Matches(msg, e.Keys.Up), msg.Text == "k":
		s.sel--
		if s.sel < 0 {
			s.sel = len(rows) - 1
		}

		return s, Out{}
	case key.Matches(msg, e.Keys.Down), msg.Text == "j":
		s.sel++
		if s.sel >= len(rows) {
			s.sel = 0
		}

		return s, Out{}
	case key.Matches(msg, e.Keys.Open), msg.Text == " ", msg.Code == tea.KeyRight, msg.Text == "l":
		return s, s.Cycle(1, e)
	case msg.Code == tea.KeyLeft, msg.Text == "h":
		return s, s.Cycle(-1, e)
	case msg.Text == "e":
		return s.Edit(rows[s.sel].Val), Out{}
	}

	return s, Out{}
}

// typing is the row being written into by hand, which is how a value that is
// on no dial — a number, a model this build has never heard of — is set.
func (s State) typing(msg tea.KeyPressMsg, e Env, rows []Row) (State, Out) {
	switch {
	case key.Matches(msg, e.Keys.Back):
		s.editing, s.typed = false, ""

		return s, Out{}
	case key.Matches(msg, e.Keys.Open):
		if s.sel < 0 || s.sel >= len(rows) {
			s.editing = false

			return s, Out{}
		}

		name, val := rows[s.sel].Key, strings.TrimSpace(s.typed)
		s.editing, s.typed = false, ""

		return s, Apply(name, val, e)
	case msg.Code == tea.KeyBackspace:
		s.typed = cells.TrimLastRune(s.typed)

		return s, Out{}
	}

	if msg.Text != "" {
		s.typed += msg.Text
	}

	return s, Out{}
}

// Cycle moves the chosen row's dial by one, in either direction. It is a
// door because the wheel and a click on the row's name both turn it.
func (s State) Cycle(delta int, e Env) Out {
	rows := s.Rows(e)
	if s.sel < 0 || s.sel >= len(rows) {
		return Out{}
	}

	r := rows[s.sel]
	if len(r.Options) == 0 {
		return Out{}
	}

	at := 0

	for i, opt := range r.Options {
		if opt == r.Val {
			at = i
			break
		}
	}

	next := (at + delta) % len(r.Options)
	if next < 0 {
		next += len(r.Options)
	}

	return Apply(r.Key, r.Options[next], e)
}

// Edit starts writing into the chosen row, with what is already in it. It is
// a door because two gestures reach it: the e key, and a click on the value
// of a row that no dial can hold.
func (s State) Edit(text string) State {
	s.editing, s.typed = true, text

	return s
}

// Typed is what has been written into the row so far, for a window that has
// to measure the line it is drawn on.
func (s State) Typed() string { return s.typed }

// Chosen is which row the cursor is on, for a window drawing a pointer at it
// or deciding what a click landed on.
func (s State) Chosen() int { return s.sel }

// Point puts the cursor on a row, which is what a click does.
func (s State) Point(at int) State {
	s.sel = at

	return s
}

// Editing is whether the chosen row is being typed into.
func (s State) Editing() bool { return s.editing }
