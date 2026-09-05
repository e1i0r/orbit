package ui

// What Orbit knows, down the side of the supervisor screen.
//
// The same facts a phase started in this repository would be told, in the
// place the operator is already looking when they write another one. It is
// the argument the Knowledge view is built on, applied to the one screen
// where somebody is writing rules: if a rule cannot be seen it is not
// trusted, and what is not trusted gets turned off.
//
// It reads and does not touch. Turning a fact off is the thing somebody will
// want to do from here, and it would put a second cursor on a screen that
// already has the thread and the line being typed; the Knowledge screen is
// where a fact is edited.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// The side's shape.
const (
	// sideWidth is how wide the column is: enough for a sentence to wrap
	// twice and still be read, and not so wide that it competes with the
	// conversation for the eye.
	sideWidth = 34
	// sideGap is the empty columns between the thread and the side.
	sideGap = 4
)

// knownSide is the column, and nothing at all when the window is too narrow
// to spare it.
//
// Narrow keeps its conversation. The thread and the line being typed are the
// screen on the terminal most people have; the side is for the width that
// was going to be empty anyway, because the content is capped at 110 columns
// so that a sentence does not stretch across a monitor.
func (m Model) knownSide(h, w, cw int) []string {
	if m.opts.Knows == nil || w-cw-sideGap < sideWidth {
		return nil
	}

	facts := m.opts.Knows()
	if len(facts) == 0 {
		return nil
	}

	stops, warns := split(facts)

	rows := make([]string, 0, h)
	rows = append(rows, Paint(Dim).Render(m.opts.Words.T("known.side", "What Orbit knows")), "")
	rows = append(rows, m.sideSection(m.opts.Words.T("known.rules", "Rules"), stops)...)
	rows = append(rows, m.sideSection(m.opts.Words.T("known.aware", "Aware"), warns)...)

	return rows
}

// split takes the facts apart by what they do. The ones that stop come
// first wherever they are drawn: they are what will send work back, and a
// reader scanning for what is standing over them looks for those.
func split(facts []knowledge.Fact) (stops, warns []knowledge.Fact) {
	for _, f := range facts {
		if f.Action() == knowledge.Stops {
			stops = append(stops, f)
			continue
		}

		warns = append(warns, f)
	}

	return stops, warns
}

// sideSection is one heading and what is under it, and nothing when there is
// nothing under it — an empty heading is a question a reader has to answer
// for themselves.
func (m Model) sideSection(head string, facts []knowledge.Fact) []string {
	if len(facts) == 0 {
		return nil
	}

	rows := []string{Paint(Accent).Bold(true).Render(head)}

	for _, f := range facts {
		rows = append(rows, m.sideFact(f)...)
	}

	return append(rows, "")
}

// sideFact is one fact: where it reaches, and what it says under it.
//
// The scope is on its own line above the sentence rather than in front of
// it, because a general fact and the repository's own sit side by side here
// and a line that did not say which is which reads as a rule about
// everything.
func (m Model) sideFact(f knowledge.Fact) []string {
	rows := []string{Paint(Dim).Render(sideWhere(f.Scope))}
	for _, line := range splitIntoLines(f.Phrase, sideWidth-2) {
		rows = append(rows, Text(Primary).Render("  "+line))
	}

	return rows
}

// sideWhere is how far a fact reaches, in as few words as the column has.
func sideWhere(s knowledge.Scope) string {
	switch s.Kind {
	case knowledge.General:
		return "everywhere"
	case knowledge.Language:
		return s.Lang
	case knowledge.Repo:
		return shortRepo(s.Repo)
	case knowledge.Symbol:
		return s.Path + "#" + s.Symbol
	default:
		return s.Path
	}
}

// besideThread puts the side beside the rows already drawn, padding each one
// out to the content width first so that the column stands still instead of
// zigzagging down the ragged right edge of the conversation.
func besideThread(rows, side []string, cw int) []string {
	if len(side) == 0 {
		return rows
	}

	out := make([]string, 0, max(len(rows), len(side)))

	for i := range max(len(rows), len(side)) {
		left, right := "", ""
		if i < len(rows) {
			left = rows[i]
		}

		if i < len(side) {
			right = side[i]
		}

		if right == "" {
			out = append(out, left)
			continue
		}

		out = append(out, padRight(fit(left, cw), cw)+strings.Repeat(" ", sideGap)+right)
	}

	return out
}
