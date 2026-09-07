package ui

// The hints along the bottom bar: which keys are worth naming right now, and
// how each one is drawn.
//
// They are here rather than in header.go because they are the other bar. The
// header says where you are and the bar says what you can do, and the two
// were one file only because both are one line of chrome.

import (
	"charm.land/bubbles/v2/key"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// hints are the bar's entries, in the order they are given up backwards.
//
// Everything about the task under the cursor comes from Affordances, so a
// key the bar offers is a key that will not be refused when it is pressed.
// The bar shows what can be done; the menu, one level down, shows what
// cannot and why.
func (m Model) hints() []barHint {
	switch m.screen {
	case screenDetail:
		return m.detailHints()
	case screenStart:
		return m.startHints()
	}

	var out []barHint

	r, ok := m.selected()
	if ok {
		out = append(out, hint("↑↓", m.opts.Words.T("key.move", "move")), hintFor(m.keys.Open))
	}

	out = append(out, hintFor(m.keys.Start))
	if ok && !r.head {
		for _, a := range m.keys.Affordances(r.task, m.conditions(r.task)) {
			if a.OK && a.Key.Help().Key != m.keys.Open.Help().Key {
				out = append(out, hintFor(a.Key))
			}
		}
	}

	// On the bar and not only in the help overlay: a key a reader never
	// sees is a key they never press.
	return append(out, hintFor(m.keys.Supervisor), hintFor(m.keys.Flows), hintFor(m.keys.Filter))
}

// hintFor is one binding as the bar prints it: the glyph a reader sees, the
// description beside it, and the keystroke a click on it would send.
//
// The keystroke is the binding's own first key rather than the glyph, and
// the two are not the same string — ⏎ is drawn and enter is pressed. Taking
// it from the binding is also what keeps a clicked hint and a pressed key on
// one path: both arrive at the board's map as the same keystroke, so a verb
// cannot be reachable by one and not by the other.
func hintFor(b key.Binding) barHint {
	h := hint(b.Help().Key, b.Help().Desc)
	h.key = string(firstKey(b))

	return h
}

// hint is the same for a pair of keys with one meaning — the arrows — which
// has no binding of its own and so no single keystroke to send. It is drawn
// and it is inert.
func hint(glyph, desc string) barHint {
	return barHint{text: theme.Paint(theme.Accent).Render("["+glyph+"]") + " " + theme.Chrome().Render(desc)}
}

// hintKey is a hint whose glyph is the whole of the keystroke it sends.
//
// Some of what the task view answers is matched by the letter inside
// detailKey rather than by a binding in m.keys, and drawn with hint those
// were drawn as keys and clicked as nothing: [m] tab menu, [v] md / raw and
// [e] expand did what pressing them does and nothing at all from the
// pointer. A hint that names a key a reader can press is a hint they can
// click, and the click sends that key.
func hintKey(glyph, desc string) barHint {
	h := hint(glyph, desc)
	h.key = glyph

	return h
}
