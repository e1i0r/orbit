package flows

// The world these tests give the designer.
//
// It is the whole of what the screen needs — words, keys, room, a place to
// keep flows and a catalogue of engines — which is the point of the package
// having an Env at all: a test builds one in six lines and never opens a
// window.

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// notch is how far one turn of the wheel moves a list. The window decides
// it and hands it down, so a test that turns the wheel says the same number
// the window would.
const notch = 3

// fixtureNow is a fixed clock: what a test asserts about a count of seconds
// is a fact about the fixture and not about when it ran.
var fixtureNow = time.Date(2026, 9, 6, 4, 0, 0, 0, time.UTC)

// zeta is the one engine this build has, invented on purpose: a made-up name
// is what proves a dial reads the catalogue rather than a list written out
// in this package.
func zeta() []string { return []string{"zeta"} }

func zetaModels(string) (ids, labels []string) {
	return []string{"zeta/one", "zeta/two"}, []string{"one", "two"}
}

func zetaEfforts(string) (ids, labels []string) {
	return []string{"brisk"}, []string{"brisk"}
}

// world is the Env, with the flows kept in a directory of the test's own.
func world(t *testing.T) Env {
	t.Helper()
	t.Setenv("ORBIT_HOME", t.TempDir())

	// Tall on purpose: these tests walk every row the screen can draw, and
	// a body that cuts the list would have them asserting about a window's
	// height rather than about the screen.
	frame, err := layout.Fit(100, 60)
	if err != nil {
		t.Fatalf("a hundred columns is too narrow to draw in: %v", err)
	}

	return Env{
		Words:   words.For("en"),
		Keys:    keymap.New(words.For("en")),
		Frame:   frame,
		Flows:   flowsTestDir(t.TempDir()),
		Now:     fixtureNow,
		Engine:  "zeta",
		Spinner: func(theme.Role) string { return "⠋ " },
		Engines: zeta,
		Models:  zetaModels,
		Efforts: zetaEfforts,
	}
}

// designer is the screen open on the list, as pressing F leaves it.
func designer(t *testing.T) (State, Env) {
	t.Helper()

	e := world(t)

	return Open(FromBoard, e), e
}

// editing is the designer with a new flow in it, on the tab where fields are
// changed by hand rather than described in words.
func editing(t *testing.T) (State, Env) {
	t.Helper()

	s, e := designer(t)
	s = s.startCreateFlow(e)

	return s.OnFields(), e
}

// linesOf is what the builder drew, as plain rows: the second answer is
// where the window started drawing, which most tests do not care about.
func linesOf(lines []builderLine, _ int) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.text)
	}

	return out
}

// press is one keystroke as the event loop delivers it.
func press(keystroke string) tea.KeyPressMsg {
	switch keystroke {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	}

	r := []rune(keystroke)[0]

	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// discriminatingWant is how much of a sentence a test has to name for the
// assertion to mean anything: three characters would pass on almost any
// message the screen could produce.
const discriminatingWant = 6

// wantBand is what the screen asked the window to say.
func wantBand(t *testing.T, out Out, want string) {
	t.Helper()

	if utf8.RuneCountInString(want) < discriminatingWant {
		t.Fatalf("wantBand(%q) would pass on almost any sentence; assert a phrase only the right message contains", want)
	}

	if !strings.Contains(out.Said, want) {
		t.Errorf("the screen said %q, want it to mention %q", out.Said, want)
	}
}
