package cli

// The two ways out of `orbit top`: one frame of plain text, or the window
// that redraws itself.

import (
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
)

// drawsOneFrame is the choice between the two ways out of this command: one
// frame of plain text, or the window.
//
// It is three words and a function of its own because the alternative — the
// expression written inline in the if — is the one branch in this command
// that no test can reach both sides of. interactive() is false for every
// writer that is not the process's own os.Stdout, and every test hands the
// command a buffer, so the second term is true in all of them and the first
// decides nothing. Neutralising -once entirely left the whole suite green.
//
// Written as a function over two bools, the table has four rows and a test
// can state all of them, including the one that matters: a terminal, and
// -once typed anyway.
func drawsOneFrame(once, terminal bool) bool {
	return once || !terminal
}

// interactive is whether opening a full-screen program over this writer is
// something a person asked for.
//
// Both halves are load-bearing. A writer that is not the process's own
// stdout is a caller that collected the output — a test, another command —
// and a program that seized the terminal there would take a keyboard nobody
// offered it; every test in this package passes a buffer, so this is the
// line that makes `orbit top` impossible to open by accident from inside one.
// A stdout that is not a character device is a pipe or a file, where a frame
// is what was wanted whether or not -once was typed.
func interactive(out io.Writer) bool {
	if out != io.Writer(os.Stdout) {
		return false
	}

	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

// fullScreen draws a model on the alternate screen.
//
// bubbletea v2 has no WithAltScreen option: whether a frame takes the whole
// terminal is a field on the View a model returns, so asking for one is
// something only a model can do. internal/ui's View does not set it, and it
// is right not to — a window that always seized the screen could not be
// rendered into a golden, and where a frame is drawn is this layer's
// decision rather than the layout's.
//
// It is a wrapper of four lines rather than a flag on ui.Options for the
// same reason: Options is what the window is allowed to reach, and how the
// terminal is put into a mode is not part of what it draws. Re-wrapping in
// Update is what keeps the alt screen through every state the model moves
// to — a bubbletea model answers with the next model, and an unwrapped one
// would drop the terminal back to the scrollback on the first keystroke.
type fullScreen struct{ tea.Model }

// Update passes the message down and keeps the wrapper on what comes back.
func (f fullScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := f.Model.Update(msg)
	return fullScreen{next}, cmd
}

// View is the model's own frame, on the alternate screen.
func (f fullScreen) View() tea.View {
	v := f.Model.View()
	v.AltScreen = true

	return v
}
