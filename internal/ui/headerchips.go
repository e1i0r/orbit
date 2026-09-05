package ui

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
// to work on and the other how much has been learned about it. Nothing known
// draws nothing — a zero on the header is a number nobody needs to read, and
// a fresh install has learned nothing yet.
func (m Model) knowledgeChip() []headerField {
	n := m.factCount()
	if n == 0 {
		return nil
	}

	return []headerField{{"knowledge", Chrome().Render("🧩 " +
		m.opts.Words.P("header.knows", n, "{n} fact", "{n} facts"))}}
}

// engineChip is the header's engine, which is the knob when one is set and
// the engine that answers otherwise.
//
// It sits here beside the knowledge chip rather than inside headerFields for
// the reason that one does: a chip is a thought of its own, and the field
// list reads better as a list of them than as the code that builds each.
func (m Model) engineChip() headerField {
	chip, ink := m.knobChip(), Paint(Accent)
	if chip == "" {
		chip, ink = m.dialEngine(""), Chrome()
	}

	return headerField{"engine", ink.Render("🧠 " + chip)}
}
