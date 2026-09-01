package ui

// What the key bar advertises, and whether the pointer can reach it.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestEveryHintDrawnAsAKeyCarriesOne. A hint is drawn "[x] what it does",
// which is read as "press x", and hitBar answers a click on it by sending
// the keystroke it carries — so a hint with no keystroke is a key the
// window offers with the keyboard and refuses with the pointer. Four of the
// task view's were exactly that: [m] tab menu, [v] md / raw, [e] expand and
// [z] fold sections are matched by the letter inside detailKey rather than
// by a binding, and the constructor for a hint without a binding is the one
// that leaves the keystroke empty.
//
// The arrows are the exception and the only one: ↑↓ is two keys with one
// meaning and no single stroke to send.
func TestEveryHintDrawnAsAKeyCarriesOne(t *testing.T) {
	base, _ := testModel(t, 120, 30)
	arrows := base.keys.Up.Help().Key + base.keys.Down.Help().Key

	board := base
	board.cursor = 1

	detail := base
	detail.screen, detail.detail = screenDetail, base.board.Tasks[0].ID

	start := base
	start.screen = screenStart

	for _, c := range []struct {
		what string
		m    Model
	}{
		{"the board", board},
		{"the task view", detail},
		{"the run dialog", start},
	} {
		for _, h := range c.m.hints() {
			glyph, _, found := strings.Cut(strings.TrimPrefix(ansi.Strip(h.text), "["), "]")
			if !found {
				t.Errorf("%s draws a hint that is not a key: %q", c.what, ansi.Strip(h.text))
				continue
			}

			if h.key == "" && glyph != arrows {
				t.Errorf("%s draws [%s] and a click on it sends nothing", c.what, glyph)
			}
		}
	}
}

// TestTheTaskViewsOwnKeysAnswerAClick walks the four hints the fix was
// about the whole way: the keystroke the bar carries goes through sendKey,
// which is the path a clicked hint takes, and the view has to change.
func TestTheTaskViewsOwnKeysAnswerAClick(t *testing.T) {
	m := openOn(t, "ACME-2662")
	m.tab = tabOverview

	for _, c := range []struct {
		key     string
		changed func(Model) bool
	}{
		{"m", func(m Model) bool { return m.menu.open }},
		{"v", func(m Model) bool { return m.rawText }},
		{"e", func(m Model) bool { return m.expandedDetail }},
	} {
		next, _ := m.sendKey(keystroke(c.key))

		if !c.changed(asModel(t, next)) {
			t.Errorf("clicking [%s] in the task view's bar did nothing", c.key)
		}
	}
}

// TestTheTaskViewsCKeyIsTheOneTheBarNames. Two things claimed c on this
// screen: the binding in m.keys, which the bar draws as [c] interactive
// CLI, and the deliver toolbar, which drew it beside "fix checks". The
// letter match came first in detailKey, so the hint the reader clicked
// wrote an instruction note for the next run — an engine's worth of work,
// from the one hint on the bar that says it opens a shell. Fix checks
// keeps a key, upper case, beside the toolbar's other capitals.
func TestTheTaskViewsCKeyIsTheOneTheBarNames(t *testing.T) {
	m := openOn(t, "ACME-2662")

	next, cmd := m.sendKey(keystroke("c"))
	if cmd == nil {
		t.Fatal("c in the task view answered with no command")
	}

	wantBand(t, asModel(t, next), "opening interactive session")

	after, _ := m.sendKey(keystroke("C"))
	wantBand(t, asModel(t, after), "running checks")
}
