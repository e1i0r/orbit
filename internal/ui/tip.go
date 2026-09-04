package ui

// The reader asking what a key does, and getting the answer where they are
// standing.
//
// ? was the cheat sheet and still reaches it, one press further in. The
// overlay is every key at once, which is the wrong shape for the question a
// reader actually has — they are looking at one hint on the bar and want to
// know what pressing it would do. So ? asks which key, and the band answers
// about that one; ? again is the whole sheet, for the reader who does not
// have a key in mind.
//
// The sentences themselves are in meaning.go.

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// tipState is the window being asked what a key does: armed is the ? that
// is waiting to be told which key, and hover is the hint the pointer is
// resting on right now.
type tipState struct {
	armed bool
	hover string
}

// armTip is ?: the next keystroke is a question about itself.
func (m Model) armTip() Model {
	m.tip.armed = true

	return m.say(m.opts.Words.T("tip.armed",
		"which key? press one to read what it does, ? for the full help, esc to leave it"))
}

// tipKey answers that question, and disarms whatever the answer was — a
// reader who asked about one key is not asking about the next one too.
//
// Every key is swallowed here, including q. While the window is holding a
// question, the answer to "what does q do" is a sentence about quitting and
// not the quitting itself; esc is the way out, because esc is the way out of
// everything else this window puts up.
func (m Model) tipKey(k fmt.Stringer) (tea.Model, tea.Cmd) {
	m.tip = tipState{}

	switch {
	case key.Matches(k, m.keys.Help):
		return m.openHelp(), nil
	case key.Matches(k, m.keys.Back):
		m.message = ""
		return m, nil
	}

	return m.say(m.meaning(k)), nil
}

// hover is the pointer resting on the key bar with no button down: the same
// answer as ?, for the reader who is already looking at the hint and has
// their hand on the mouse.
//
// It reads hitBar and nothing else, so a hint is explained exactly where it
// was drawn — including at the widths where the bar drops it, since a hint
// that is not on the line cannot be pointed at. Off the bar the sentence
// goes with the pointer: a tooltip that stayed behind is a sentence about a
// key the reader has stopped asking about.
func (m Model) hover(e tea.Mouse) Model {
	if t := m.hitBar(e.X, e.Y); t.Kind == TargetBarHint {
		m.tip.hover = t.Key
		return m
	}

	m.tip.hover = ""

	return m
}

// hovered is what the band says for it, and nothing at all when the pointer
// is not on a hint. The arrows are drawn as a hint and send no keystroke, so
// they are the one entry of the bar with nothing to say.
func (m Model) hovered() string {
	if m.tip.hover == "" {
		return ""
	}

	return m.meaning(keystroke(m.tip.hover))
}
