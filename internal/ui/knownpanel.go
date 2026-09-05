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
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// The side's shape.
const (
	// sideWidth is how wide the column is: enough for a sentence to wrap
	// twice and still be read, and not so wide that it competes with the
	// conversation for the eye.
	sideWidth = 30
	// sideGap is the empty columns between the thread and the side.
	sideGap = 3
	// sideMinThread is the narrowest the conversation may be left. Below it
	// the thread stops being readable, and a panel that made the thing
	// somebody came here for unreadable would be a bad trade however useful
	// it is.
	sideMinThread = 70
)

// knownSide is the column, and nothing at all when the window is too narrow
// to spare it.
//
// Narrow keeps its conversation. The thread and the line being typed are the
// screen on the terminal most people have; the side is for the width that
// was going to be empty anyway, because the content is capped at 110 columns
// so that a sentence does not stretch across a monitor.
func (m Model) knownSide(h, w int) []string {
	if !m.sideFits(w) {
		return nil
	}

	rules, warns := split(m.supervisor.knows)

	rows := []string{Paint(Dim).Render(m.opts.Words.T("known.side", "What Orbit knows")), ""}
	rows = append(rows, m.sideSection(m.opts.Words.T("known.rules", "Rules"), rules)...)
	rows = append(rows, m.sideSection(m.opts.Words.T("known.aware", "Aware"), warns)...)

	return m.cutSide(rows, h, len(m.supervisor.knows))
}

// cutSide keeps the side inside the screen and says what it left out.
//
// A column that quietly stops listing is worse than a short one: whoever
// reads it believes they have seen everything Orbit knows, and the rules
// past the bottom are exactly the ones nobody finds out about. The rules are
// drawn first, so they are the last thing given up for room.
//
// What is cut is not lost — the Knowledge screen lists all of it, and this
// says how many are waiting there.
func (m Model) cutSide(rows []string, h, facts int) []string {
	if len(rows) <= h {
		return rows
	}

	kept := rows[:max(h-1, 0)]
	shown := 0

	for _, row := range kept {
		if strings.HasPrefix(ansi.Strip(row), "  ") {
			shown++
		}
	}

	return append(kept, Paint(Dim).Render(
		m.opts.Words.T("known.more", "{n} more, in the Knowledge screen",
			about("n", strconv.Itoa(max(facts-shown, 1))))))
}

// sideFits is whether the side is drawn: there is something to say, and the
// conversation still has room to be read.
//
// The width it takes is not width going spare. The content is capped at 110
// columns so that a sentence does not stretch across a monitor, but on a
// terminal narrower than that there is nothing left over — and waiting for a
// screen wide enough meant the panel never appeared on the screen most people
// actually have. So it takes its columns from the thread, down to the point
// where the thread would stop being readable.
func (m Model) sideFits(w int) bool {
	return len(m.supervisor.knows) > 0 && w-sideGap-sideWidth >= sideMinThread
}

// split takes the facts apart by what was asked of them, not by what they
// can do.
//
// The difference matters here and nowhere else. A rule written as a sentence
// brings no check, so it can only warn — and grouping by that put it under
// Aware, where the person who had just typed /rule read it as the gesture
// having been ignored. It goes where they put it, and the mark beside it says
// it cannot fire yet.
//
// The ones that stop come first wherever they are drawn: they are what will
// send work back, and a reader scanning for what is standing over them looks
// for those.
func split(facts []knowledge.Fact) (rules, warns []knowledge.Fact) {
	for _, f := range facts {
		if f.Stops {
			rules = append(rules, f)
			continue
		}

		warns = append(warns, f)
	}

	return rules, warns
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
	where := sideWhere(f.Scope)
	if f.Stops && f.Action() != knowledge.Stops {
		where += " · " + m.opts.Words.T("known.no_check", "no check yet")
	}

	rows := []string{Paint(Dim).Render(where)}
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

		// The thread's rows are wrapped to cw, and the one that carries the
		// scroll rail is a column wider than that: the rail stands past the
		// text. Cutting at cw takes it off and leaves an ellipsis down the
		// seam of every row that had one.
		out = append(out, padRight(fit(left, cw+1), cw+1)+strings.Repeat(" ", sideGap)+right)
	}

	return out
}
