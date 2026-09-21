package ui

// A command that failed says so where the reader is looking.

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestAFailedCommandSaysSoInThePane.
//
// The pane opens when a command starts and stays up until it is closed. A
// command that failed having printed nothing left it reading "no output
// yet… finished", and the only account of what went wrong went to the
// band — where it sat under whatever had been said before, after a form
// that had just closed over it. That is how "no such command: new" was on
// screen for a week without anybody reading it as the reason nothing was
// ever written.
func TestAFailedCommandSaysSoInThePane(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m.watching = &commandWatch{name: "board"}
	m.watchUp = true

	broke := errors.New("no such command: new")

	answered, _ := m.Update(commandMsg{Name: "board", Err: broke})
	next := asModel(t, answered)

	drawn := ansi.Strip(strings.Join(next.watchRows(next.frame.Body.H, next.frame.Body.W), "\n"))
	if !strings.Contains(drawn, "no such command: new") {
		t.Errorf("the pane says nothing about the failure:\n%s", drawn)
	}

	// And what the command did print is still what the pane shows when it
	// printed something: the error is the fallback, not a replacement.
	m2, _ := testModel(t, 100, 30)
	m2.watching = &commandWatch{name: "board"}
	m2.watchUp = true

	told, _ := m2.Update(commandMsg{Name: "board", Text: "wrote it down", Err: broke})
	spoke := asModel(t, told)

	said := ansi.Strip(strings.Join(spoke.watchRows(spoke.frame.Body.H, spoke.frame.Body.W), "\n"))
	if !strings.Contains(said, "wrote it down") {
		t.Errorf("the pane lost what the command printed:\n%s", said)
	}
}
