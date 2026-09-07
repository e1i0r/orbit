package engines

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

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/roster"
)

// knobFilter is what has been typed, trimmed and folded for comparing.
func (s State) knobFilter() string {
	return strings.ToLower(strings.TrimSpace(s.filter))
}

// matchingModels is an engine's catalogue as the filter leaves it: all of it
// when nothing is typed, and all of it again when the engine's own name is
// what matched — "codex" is a reasonable thing to type when what you want is
// everything codex runs.
func (s State) matchingModels(eng roster.Engine) []roster.Choice {
	want := s.knobFilter()
	if want == "" || strings.Contains(strings.ToLower(eng.Name), want) {
		return eng.Models
	}

	out := make([]roster.Choice, 0, len(eng.Models))

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
func (s State) engineFilterKey(msg tea.KeyPressMsg, e Env) State {
	switch {
	case key.Matches(msg, e.Keys.Back):
		s.typing, s.filter = false, ""
	case key.Matches(msg, e.Keys.Open):
		s.typing = false
	case msg.Code == tea.KeyBackspace:
		s.filter = cells.TrimLastRune(s.filter)
	case key.Matches(msg, e.Keys.Up), key.Matches(msg, e.Keys.Down):
		// The arrows still walk the list while the line is open: typing
		// three letters and then choosing what they found is one gesture,
		// and a mode that had to be left first would break it in half.
		return s.walkKnobs(key.Matches(msg, e.Keys.Down), e)
	case msg.Text != "":
		s.filter += msg.Text
	}

	return s.keepKnobChoiceReal(e)
}

// keepKnobChoiceReal puts the cursor back on a row that exists. The list is
// recomputed from what is typed, so the row the cursor was on can be gone by
// the next keystroke — and a cursor past the end of the list is a ⏎ that
// chooses nothing.
func (s State) keepKnobChoiceReal(e Env) State {
	n := len(selectableEngineIndices(s.collectEngineRows(e)))
	if n == 0 {
		s.sel = 0
		return s
	}

	s.sel = min(max(s.sel, 0), n-1)

	return s.keepEngineRowSeen(e)
}

// walkKnobs moves the cursor one row, wrapping at the ends as the arrows do
// everywhere on this screen.
func (s State) walkKnobs(down bool, e Env) State {
	n := len(selectableEngineIndices(s.collectEngineRows(e)))
	if n == 0 {
		return s
	}

	if down {
		s.sel = (s.sel + 1) % n
	} else {
		s.sel = (s.sel + n - 1) % n
	}

	return s.keepEngineRowSeen(e)
}

// knobFilterLine is what the chrome says in place of the advice while the
// list is cut down: what was typed, and — when it matched nothing — that it
// did, rather than an empty screen the reader has to explain to themselves.
func (s State) knobFilterLine(models int, e Env) string {
	p := e.Words

	typed := s.filter
	if s.typing {
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
func (s State) shownModels(e Env) int {
	var n int

	for _, r := range s.collectEngineRows(e) {
		if r.kind == rowModel {
			n++
		}
	}

	return n
}
