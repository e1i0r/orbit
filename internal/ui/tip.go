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

// tipState is the window waiting to be told which key to explain.
type tipState struct {
	armed bool
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
