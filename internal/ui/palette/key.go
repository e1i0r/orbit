package palette

// Every key the line answers, and what running one of its rows means.
//
// ↑↓ are matched against what the terminal sent rather than against the key
// map, on purpose. Up and Down carry k and j as alternates, and both are
// letters a command's name can need typed into this very line; a match
// through the map would move the selection every time the reader typed one.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
)

// Key feeds the line, which owns every key it is not given a reason to give
// up.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	switch {
	case key.Matches(msg, e.Keys.Back):
		return State{}, Out{Leave: true}
	case key.Matches(msg, e.Keys.Open):
		return s.Run(e)
	case msg.String() == "tab":
		return s.complete(e), Out{}
	case msg.String() == "up":
		return s.pick(-1, e), Out{}
	case msg.String() == "down":
		return s.pick(1, e), Out{}
	case msg.Code == tea.KeyBackspace:
		s.typed = cells.TrimLastRune(s.typed)
		return s.reselect(), Out{}
	}

	if msg.Text != "" {
		s.typed += msg.Text
		return s.reselect(), Out{}
	}

	return s, Out{}
}

// Run is the command under the selection going out, which ⏎ and a click both
// do. Nothing is chosen when nothing matches, and the line stays up rather
// than closing on a keystroke that did nothing.
func (s State) Run(e Env) (State, Out) {
	c, ok := s.selected(e.Commands)
	if !ok {
		// Nothing under the selection: an empty list, usually. Staying open
		// says "not yet" more honestly than closing would.
		return s, Out{}
	}

	if c.NeedsArgs && len(argsOf(s.typed)) == 0 {
		// The usage is the table's own, so the sentence is right for every
		// command that reaches it without one being written per command.
		return s, Out{Said: e.Words.T("msg.needs_args", "{name} takes {args}; type them here",
			about("name", c.Name), about("args", c.Args))}
	}

	return State{}, Out{Leave: true, Run: c.Name, Line: s.typed}
}

// Choose is a click on one of the rows: the same two-step a task row takes,
// where the first click selects and the second runs what was selected. Both
// arrive through the same methods the keyboard uses — there is no third path
// to a run.
func (s State) Choose(name string, e Env) (State, Out) {
	i, ok := s.indexOf(name, e)
	if !ok {
		return s, Out{}
	}

	if i == s.sel {
		return s.Run(e)
	}

	s.sel = i

	return s.ensureVisible(e), Out{}
}

// indexOf is where a named command sits in the filtered list, which is what
// the pointer needs and the keyboard never does: the keyboard moves by rows,
// the pointer arrives holding a name.
func (s State) indexOf(name string, e Env) (int, bool) {
	for i, c := range s.candidates(e.Commands) {
		if c.Name == name {
			return i, true
		}
	}

	return 0, false
}

// reselect puts the selection back at the top of whatever the line now
// matches. The alternative — keeping the old index — points at a row the
// reader has never seen choose, and ⏎ would then run a command nobody
// asked for.
func (s State) reselect() State {
	s.sel, s.offset = 0, 0

	return s
}

// pick moves the selection one row, and the window with it.
func (s State) pick(d int, e Env) State {
	if n := len(s.candidates(e.Commands)); n > 0 {
		s.sel += d
		if s.sel < 0 {
			s.sel = 0
		}

		if s.sel >= n {
			s.sel = n - 1
		}
	}

	return s.ensureVisible(e)
}

// ensureVisible moves the offset as little as it can while keeping the
// selection on screen. It reads the frame's body height, which is what the
// renderer counts rows with — one number, two readers, the rule target.go
// exists to keep.
func (s State) ensureVisible(e Env) State {
	h := e.Frame.Body.H
	switch {
	case h <= 0:
		s.offset = 0
	case s.sel < s.offset:
		s.offset = s.sel
	case s.sel >= s.offset+h:
		s.offset = s.sel - h + 1
	}

	if s.offset < 0 {
		s.offset = 0
	}

	return s
}

// complete fills the line with the selection's name. The list is already
// prefix-filtered, so the selection always completes what was typed — tab
// never jumps sideways to a command the reader did not start spelling.
//
// The selection goes back to the top with the line, because completing has
// made the list shorter and an index past its end would leave ⏎ pointing at
// nothing.
func (s State) complete(e Env) State {
	if c, ok := s.selected(e.Commands); ok {
		s.typed = c.Name

		return s.reselect()
	}

	return s
}

// Wheel moves the selection, which is what scrolling a list with one row
// chosen means; the list follows it rather than the reader following the
// list.
func (s State) Wheel(d int, e Env) State { return s.pick(d, e) }

// Type puts text on the line, which is what a paste is.
func (s State) Type(text string) State {
	s.typed += text

	return s.reselect()
}

// argsOf is what was typed after the command's name. The line is split on
// spaces and quoting is not understood: a path with a space in it cannot be
// passed this way yet, and the form that writes a task is the answer for the
// one command that most wants one.
func argsOf(typed string) []string {
	args := strings.Fields(typed)
	if len(args) > 0 {
		args = args[1:]
	}

	return args
}

// Args is what the window runs the chosen command with.
func Args(line string) []string { return argsOf(line) }
