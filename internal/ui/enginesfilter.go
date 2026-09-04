package ui

// Typing to cut the model catalogue down.
//
// opencode alone offers sixty-five models. Folding made that list possible
// to walk past; it did not make one model possible to find, and a reader who
// knows they want sonnet should not have to go down sixty-five rows reading
// for it. The key is the board's — "/" filters what is in front of you — so
// the same gesture means the same thing on both screens.
//
// What is cut is the catalogue and nothing else. Effort and thinking are two
// short dials rather than a list to search, and a filter that emptied them
// too would answer "sonnet" by hiding the effort the reader was about to
// set.

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// knobFilter is what has been typed, trimmed and folded for comparing.
func (m Model) knobFilter() string {
	return strings.ToLower(strings.TrimSpace(m.engines.filter))
}

// matchingModels is an engine's catalogue as the filter leaves it: all of it
// when nothing is typed, and all of it again when the engine's own name is
// what matched — "codex" is a reasonable thing to type when what you want is
// everything codex runs.
func (m Model) matchingModels(eng EngineInfo) []ChoiceInfo {
	want := m.knobFilter()
	if want == "" || strings.Contains(strings.ToLower(eng.Name), want) {
		return eng.Models
	}

	out := make([]ChoiceInfo, 0, len(eng.Models))

	for _, mdl := range eng.Models {
		if strings.Contains(strings.ToLower(mdl.ID+" "+mdl.Label), want) {
			out = append(out, mdl)
		}
	}

	return out
}

// engineFilterKey feeds the line while it is being typed.
//
// The two ways out are the board's, and they are different gestures on
// purpose: ⏎ leaves the line with the list still cut down, which is what
// filtering was for, and esc gives back the whole catalogue.
func (m Model) engineFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.engines.typing, m.engines.filter = false, ""
	case key.Matches(msg, m.keys.Open):
		m.engines.typing = false
	case msg.Code == tea.KeyBackspace:
		m.engines.filter = trimLastRune(m.engines.filter)
	case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down):
		// The arrows still walk the list while the line is open: typing
		// three letters and then choosing what they found is one gesture,
		// and a mode that had to be left first would break it in half.
		return m.walkKnobs(key.Matches(msg, m.keys.Down)), nil
	case msg.Text != "":
		m.engines.filter += msg.Text
	}

	return m.keepKnobChoiceReal(), nil
}

// keepKnobChoiceReal puts the cursor back on a row that exists. The list is
// recomputed from what is typed, so the row the cursor was on can be gone by
// the next keystroke — and a cursor past the end of the list is a ⏎ that
// chooses nothing.
func (m Model) keepKnobChoiceReal() Model {
	n := len(m.selectableEngineIndices(m.collectEngineRows()))
	if n == 0 {
		m.engines.sel = 0
		return m
	}

	m.engines.sel = min(max(m.engines.sel, 0), n-1)

	return m.keepEngineRowSeen()
}

// walkKnobs moves the cursor one row, wrapping at the ends as the arrows do
// everywhere on this screen.
func (m Model) walkKnobs(down bool) Model {
	n := len(m.selectableEngineIndices(m.collectEngineRows()))
	if n == 0 {
		return m
	}

	if down {
		m.engines.sel = (m.engines.sel + 1) % n
	} else {
		m.engines.sel = (m.engines.sel + n - 1) % n
	}

	return m.keepEngineRowSeen()
}

// knobFilterLine is what the chrome says in place of the advice while the
// list is cut down: what was typed, and — when it matched nothing — that it
// did, rather than an empty screen the reader has to explain to themselves.
func (m Model) knobFilterLine(models int) string {
	p := m.opts.Words

	typed := m.engines.filter
	if m.engines.typing {
		typed += "▌"
	}

	if models == 0 {
		return p.T("engines.filter_none", "no model matches {typed}", about("typed", typed))
	}

	return p.T("engines.filter_line", "filter: {typed} · {count} models",
		about("typed", typed), about("count", fmt.Sprint(models)))
}

// shownModels is how many models are on the list as it stands, which is what
// the filter line reports. It is counted off the rows rather than added up
// per engine, so what it says is what is drawn.
func (m Model) shownModels() int {
	var n int

	for _, r := range m.collectEngineRows() {
		if r.kind == rowModel {
			n++
		}
	}

	return n
}
