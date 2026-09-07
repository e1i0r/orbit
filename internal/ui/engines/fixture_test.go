package engines

// The world these tests give the knobs screen.

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/words"
)

// world is the Env: the catalogue below, the words, the keys, and a body a
// hundred cells wide.
func world(t *testing.T) Env {
	t.Helper()

	frame, err := layout.Fit(100, 30)
	if err != nil {
		t.Fatalf("a hundred columns is too narrow to draw in: %v", err)
	}

	return Env{
		Words:   words.For("en"),
		Keys:    keymap.New(words.For("en")),
		Frame:   frame,
		Engines: enginesTestList,
		Settled: "claude",
	}
}

// open is the screen as pressing M leaves it, on dials nobody has turned.
func open(t *testing.T) (State, Env) {
	t.Helper()

	e := world(t)

	return Open(0, Knobs{}), e
}

// press is one keystroke as the event loop delivers it.
func press(keystroke string) tea.KeyPressMsg {
	switch keystroke {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	}

	r := []rune(keystroke)[0]

	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// drawn is the screen as rows, at the size the fixture's frame gives it.
func drawn(s State, e Env) []string {
	return s.View(e.Frame.Body.H, e.Frame.Body.W, e)
}
