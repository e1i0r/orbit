package supervisor

// The world these tests give the screen.
//
// It is the whole of what the screen needs — words, keys, room, and the
// doors of a record kept in a variable — which is the point of the package
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
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// fixtureNow is a fixed clock: what a test asserts about a count of seconds
// is a fact about the fixture and not about when it ran.
var fixtureNow = time.Date(2026, 9, 6, 4, 0, 0, 0, time.UTC)

// held is the record these tests write into and read back, so that what a
// gesture wrote is asserted from the thread rather than guessed at.
type held struct {
	lines []view.SupervisorLine
	wrote []view.SupervisorLine
	back  []time.Time
	gone  []string
}

// log is the Log door: every line of every conversation.
func (h *held) log() ([]view.SupervisorLine, error) { return h.lines, nil }

// record is the Record door, which both keeps what was written and puts it
// in the thread the next read returns.
func (h *held) record(conversation, by, channel, text string) error {
	l := view.SupervisorLine{
		At: fixtureNow, Kind: "supervisor.message", By: by,
		Channel: channel, Text: text, Conversation: conversation,
	}
	h.wrote = append(h.wrote, l)
	h.lines = append(h.lines, l)

	return nil
}

func (h *held) retract(at time.Time) error { h.back = append(h.back, at); return nil }
func (h *held) forget(id string) error     { h.gone = append(h.gone, id); return nil }

// world is the Env, with a record of its own and nothing else lent to it.
func world(t *testing.T, kept *held) Env {
	t.Helper()

	frame, err := layout.Fit(100, 30)
	if err != nil {
		t.Fatalf("a hundred columns is too narrow to draw in: %v", err)
	}

	e := Env{
		Words:   words.For("en"),
		Keys:    keymap.New(words.For("en")),
		Frame:   frame,
		Now:     fixtureNow,
		Spinner: func(theme.Role) string { return "⠋ " },
		NewID:   func() string { return "c-new" },
		// zeta is invented on purpose: a made-up name is what proves the
		// screen asks who the engines are rather than reading a list
		// written out in this package.
		Engine:   "zeta",
		IsEngine: func(name string) bool { return name == "zeta" },
		Ask:      func(string, string) tea.Cmd { return func() tea.Msg { return nil } },
	}

	if kept != nil {
		e.Log, e.Record, e.Retract, e.Forget = kept.log, kept.record, kept.retract, kept.forget
	}

	return e
}

// open is the screen as pressing S leaves it, on an empty record.
func open(t *testing.T) (State, Env) {
	t.Helper()

	e := world(t, &held{})

	return Open(0, e), e
}

// opened is the screen on a record that already has something in it.
func opened(t *testing.T, kept *held) (State, Env) {
	t.Helper()

	e := world(t, kept)

	return Open(0, e), e
}

// spoke is one line of a conversation.
func spoke(at time.Time, conv, by, text string) view.SupervisorLine {
	return view.SupervisorLine{At: at, Kind: "supervisor.message", By: by, Text: text, Conversation: conv}
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
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	}

	r := []rune(keystroke)[0]

	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// ctrl is a key held with control, which is how this screen's three modes
// are reached.
func ctrl(letter rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: letter, Mod: tea.ModCtrl}
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
