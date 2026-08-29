package ui

import (
	"fmt"
	"io"
	"time"

	tea "charm.land/bubbletea/v2"
)

// commandMsg is a palette command having come back, with whatever it
// printed in its last moments and its error verbatim — the window is
// a keyboard in front of the commands, not a second copy of their
// rules, so the error is said exactly as the command phrased it.
type commandMsg struct {
	Name string
	Text string // the final snapshot of the watch's buffer
	Err  error
}

// outputMsg is one tick of a running command's output, read off the
// watch's buffer. It carries no verdict; only commandMsg ends a run.
type outputMsg struct {
	Name string
	Text string
}

// runCommand runs one palette command through the port, off the event
// loop, writing everything it prints into the watch. The verdict comes
// back once, with the final snapshot riding along so the last frame of
// output and the sentence about the run land in the same message.
//
// A nil port is a window opened without a way to run commands, and it is
// said here rather than at the keystroke because the keystroke cannot know
// what the port will answer.
func runCommand(port func(string, []string, io.Writer) error, w *commandWatch, args []string) tea.Cmd {
	return func() tea.Msg {
		err := fmt.Errorf("%w: %s", errNoCommandPort, w.name)
		if port != nil {
			err = port(w.name, args, w)
		}

		w.finish()
		text, _ := w.snapshot()

		return commandMsg{Name: w.name, Text: text, Err: err}
	}
}

// outputPump reads the watch's buffer onto the screen while its command
// runs. It is re-armed by Update on every tick it delivers and stops being
// re-armed the moment the watch is no longer the running one — which is how
// the pump ends without anybody keeping count of it.
func outputPump(w *commandWatch) tea.Cmd {
	return tea.Tick(outputTick, func(time.Time) tea.Msg {
		// The verdict may already be in flight; this tick carries only
		// bytes, and one arriving after commandMsg is dropped by name.
		text, _ := w.snapshot()
		return outputMsg{Name: w.name, Text: text}
	})
}
