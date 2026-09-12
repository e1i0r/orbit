package ui

import "github.com/e1i0r/orbit/internal/ui/theme"

// The header's chips: the standing facts about the workspace, drawn along the
// top and each one a thing a pointer can press.
//
// They live apart from header.go because a chip is a thought of its own — what
// it counts, when it is worth drawing at all, and where pressing it goes — and
// the field list reads better as a list of them than as the code that builds
// each one in place.

// knowledgeChip is the header's count of what Orbit has learned.
//
// Beside the repositories and not instead of them: one says how much there is
// to work on and the other how much has been learned about it.
//
// It is drawn at zero as well, dimmed. The chip is the door to the screen
// where facts are written and turned back on, and a door that appears only
// once you are through it is one nobody finds: an install that has learned
// nothing is exactly the one whose reader needs to be told the screen exists.
func (m Model) knowledgeChip() []headerField {
	// Chrome at zero as well as at forty: the header shares one ink, and
	// faint text on it is the thing theme_test.go refuses — a chip nobody
	// can read is not a gentler way of saying nothing.
	chip := "🧩 " + m.opts.Words.P("header.knows", m.factCount(), "{n} fact", "{n} facts")

	// And what is waiting to be answered, when anything is. A tray nobody is
	// told about is a tray nobody opens, and this is the only place a reader
	// who is not already on that screen would learn there is one.
	if n := m.knowledge.Unanswered(); n > 0 {
		chip += " · " + m.opts.Words.P("header.waiting", n, "{n} waiting", "{n} waiting")
	}

	return []headerField{{"knowledge", theme.Chrome().Render(chip)}}
}

// engineChip is the header's engine, which is the knob when one is set and
// the engine that answers otherwise.
//
// It sits here beside the knowledge chip rather than inside headerFields for
// the reason that one does: a chip is a thought of its own, and the field
// list reads better as a list of them than as the code that builds each.
func (m Model) engineChip() headerField {
	chip, ink := m.knobChip(), theme.Paint(theme.Accent)
	if chip == "" {
		chip, ink = m.dialEngine(""), theme.Chrome()
	}

	return headerField{"engine", ink.Render("🧠 " + chip)}
}
