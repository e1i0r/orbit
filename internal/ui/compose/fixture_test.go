package compose

// The world these tests give the form.

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/words"
)

// fixtureNow is a fixed clock: what a test asserts about a count of seconds
// is a fact about the fixture and not about when it ran.
var fixtureNow = time.Date(2026, 9, 6, 4, 0, 0, 0, time.UTC)

// world is the Env: the words, the keys, a hundred columns of room, and two
// checkouts to start work in.
func world(t *testing.T) Env {
	t.Helper()
	t.Setenv("ORBIT_HOME", t.TempDir())

	// Wide on purpose: these tests point at cells, and the row of flow
	// pills fills a hundred columns, so there is nowhere on it that is the
	// row and not one of the pills.
	frame, err := layout.Fit(140, 40)
	if err != nil {
		t.Fatalf("a hundred and forty columns is too narrow to draw in: %v", err)
	}

	return Env{
		Words: words.For("en"),
		Keys:  keymap.New(words.For("en")),
		Frame: frame,
		Now:   fixtureNow,
		Flows: flowsIn(t.TempDir()),
		Places: []Place{
			{Name: "payments", Path: "/r/payments"},
			{Name: "app", Path: "/r/app"},
		},
	}
}

// flowsIn is a reader's own flow directory, as flow.Source asks for it.
type flowsIn string

func (d flowsIn) FlowDir() string { return string(d) }

// form is the screen as pressing N leaves it, on no repository in
// particular.
func form(t *testing.T) (State, Env) {
	t.Helper()

	e := world(t)

	return Open("", e), e
}

// press is one keystroke as the event loop delivers it.
func press(keystroke string) tea.KeyPressMsg {
	switch keystroke {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
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
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	}

	r := []rune(keystroke)[0]

	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// typed is a whole word going into the field the caret is in.
func typed(s State, e Env, word string) State {
	for _, r := range word {
		s, _ = s.Key(press(string(r)), e)
	}

	return s
}

// discriminatingWant is how much of a sentence a test has to name for the
// assertion to mean anything.
const discriminatingWant = 6

// wantBand is what the form asked the window to say.
func wantBand(t *testing.T, out Out, want string) {
	t.Helper()

	if utf8.RuneCountInString(want) < discriminatingWant {
		t.Fatalf("wantBand(%q) would pass on almost any sentence; assert a phrase only the right message contains", want)
	}

	if !strings.Contains(out.Said, want) {
		t.Errorf("the form said %q, want it to mention %q", out.Said, want)
	}
}

// keyed is one keystroke, for the tests that only want what it left behind.
func keyed(s State, msg tea.KeyPressMsg, e Env) State {
	next, _ := s.Key(msg, e)

	return next
}

// screenRows is the form drawn, in the window's own rows: the body starts
// where the frame says it does, and the pointer answers about the terminal
// and not about the block. The rows above it are the header the window
// draws, which this package does not.
func screenRows(s State, e Env) []string {
	rows := make([]string, e.Frame.Body.Y)

	drawn := ansi.Strip(strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n"))

	return append(rows, strings.Split(drawn, "\n")...)
}

// rowOf is the first row of lines that mentions want, or -1.
func rowOf(lines []string, want string) int {
	for i, l := range lines {
		if strings.Contains(l, want) {
			return i
		}
	}

	return -1
}
