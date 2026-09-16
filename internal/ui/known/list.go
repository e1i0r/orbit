package known

// The list of rules, in the shape the board already has: bands that say what
// you have to do about what is in them, and one line a rule under them.
//
// Bands and not a column of states. A reader opens this screen with one
// question — is there anything here for me — and the board answers that with
// where a row is rather than with a word inside it. Two columns saying
// "paused" and "no check" made the reader do the sorting themselves.

import (
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/fact"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// The columns that are not the rule itself. The rule takes what is left,
// because it is the only one that is prose and the only one anybody came to
// read.
const (
	colID    = 8
	colWhere = 16
	// colWheres is the same column on a board of several checkouts, where
	// every path carries the name of the one it is in.
	colWheres = 21
	colWhen   = 10
	colGap    = 2
	// gutter is the two cells every row opens with: the cursor's mark.
	gutter = 2
	// ruleLeast is the narrowest the sentence is allowed to get before the
	// other columns start being dropped.
	ruleLeast = 30
)

// widths is how many cells each column got at the width the screen has.
type widths struct{ id, rule, where, when int }

// plan shares the width out, dropping columns a narrow terminal has no room
// for. What goes is what a reader can find another way: the date and the
// name are both on the rule's own screen.
func plan(cw int, many bool) widths {
	w := widths{id: colID, where: colWhere, when: colWhen}
	if many {
		w.where = colWheres
	}

	for range 2 {
		if cw-gutter-w.id-w.where-w.when-4*colGap >= ruleLeast {
			break
		}

		if w.when > 0 {
			w.when = 0
			continue
		}

		w.id = 0
	}

	w.rule = max(cw-gutter-w.id-w.where-w.when-4*colGap, 12)

	return w
}

// rows is the whole screen: the list, or the one rule being read or written.
func (s State) rows(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	if s.editing {
		return s.formRows(h, w, e)
	}

	if s.reading {
		return s.detailRows(h, w, e)
	}

	cw := content(w)

	if len(s.facts) == 0 && len(s.waiting) == 0 {
		return cells.Fill(rowsFit(s.nothing(cw, e), w), h)
	}

	above, below := s.head(cw, e), s.foot(cw, e)
	body, _ := s.body(cw, e)

	room := max(1, h-len(above)-len(below))
	off := s.window(len(body), room)

	out := append([]string{}, above...)
	for i := off; i < len(body) && i < off+room; i++ {
		out = append(out, body[i])
	}

	// The foot is held against the bottom rather than run on after the last
	// rule, so the line that says what the keys do is in the same place on a
	// screen of three rules and a screen of forty.
	for len(out) < max(h-len(below), 0) {
		out = append(out, "")
	}

	return cells.Fill(rowsFit(append(out, below...), w), h)
}

// nothing is the screen before anything has been written down, which is the
// screen most people meet first: it says how to put something in it.
func (s State) nothing(cw int, e Env) []string {
	p := e.Words

	return []string{
		"",
		theme.Paint(theme.Accent).Render(p.T("knowledge.title", "What Orbit knows")),
		"",
		theme.Paint(theme.Dim).Render(cells.Fit(p.T("knowledge.empty",
			"Nothing written down yet. Say /rule or /aware to the supervisor, "+
				"or drop a file in .orbit/knowledge/."), cw)),
	}
}

// head is the title, what the screen holds, and the row of column names.
func (s State) head(cw int, e Env) []string {
	p := e.Words

	title := theme.Paint(theme.Accent).Render(p.T("knowledge.title", "What Orbit knows"))

	tally := theme.Paint(theme.Dim).Render(
		p.P("knowledge.tally", len(s.facts), "{n} rule", "{n} rules",
			about("n", strconv.Itoa(len(s.facts)))))

	return []string{"", cells.Spread(title, tally, cw), "", s.columns(plan(cw, many(e)), cw, e)}
}

// columns is the row of column names, which is drawn once at the top because
// it is part of the frame rather than of the list — the board's own is too.
func (s State) columns(w widths, cw int, e Env) string {
	p := e.Words
	dim := theme.Paint(theme.Dim).Render

	left := strings.Repeat(" ", gutter) +
		pad(p.T("knowledge.col_id", "ID"), w.id, dim) +
		dim(cells.Pad(p.T("knowledge.col_rule", "THE RULE"), w.rule, false))

	return cells.Spread(left,
		pad(p.T("knowledge.col_where", "WHERE"), w.where, dim)+
			dim(cells.Pad(p.T("knowledge.col_when", "WHEN"), w.when, false)), cw)
}

// pad is one cell of a row: cut to its budget, filled out to it, coloured,
// and followed by the gap. A column planned at nothing is not drawn at all.
func pad(text string, n int, ink func(...string) string) string {
	if n <= 0 {
		return ""
	}

	return ink(cells.Pad(text, n, false)) + strings.Repeat(" ", colGap)
}

// body is everything that scrolls, and where each row the cursor can be on
// begins.
func (s State) body(cw int, e Env) ([]string, []span) {
	var (
		out []string
		at  []span
	)

	w := plan(cw, many(e))

	for _, b := range bands {
		held := s.inBand(b)
		if len(held) == 0 {
			continue
		}

		out = append(out, bandHead(b, len(held), cw, e))

		for _, one := range held {
			row := s.entryRow(one, len(at) == s.sel, w, cw, e)
			at = append(at, span{from: len(out), rows: len(row)})
			out = append(out, row...)
		}

		out = append(out, "")
	}

	return out, at
}

// bandHead is one band's name, how many are in it, and the rule out to the
// edge — the board's own heading, so the two screens read as one window.
func bandHead(b band, n, cw int, e Env) string {
	name := theme.Paint(theme.Accent).Bold(true).Render(bandName(b, e)) +
		" " + theme.Paint(bandCount(b)).Render("("+strconv.Itoa(n)+")") + " "

	rule := max(cw-lipgloss.Width(name), 0)

	return name + theme.Paint(theme.Dim).Render(strings.Repeat("─", rule))
}

// entryRow is one line of the list: a sentence nobody has answered, or a
// rule.
func (s State) entryRow(one entry, chosen bool, w widths, cw int, e Env) []string {
	if one.said {
		return s.saidRow(s.waiting[one.at], chosen, w, cw)
	}

	return s.ruleRow(s.facts[one.at], chosen, w, cw, e)
}

// ruleRow is a rule: its name, its sentence, how far it reaches and since
// when. What it does and where it stands are the band it is under.
func (s State) ruleRow(f knowledge.Fact, chosen bool, w widths, cw int, e Env) []string {
	ink := theme.Text(theme.Primary)
	if !f.Tells() {
		ink = theme.Paint(theme.Dim)
	}

	said := cells.Lines(f.Phrase, w.rule)
	if len(said) == 0 {
		said = []string{""}
	}

	rows := []string{cells.Spread(
		mark(chosen)+
			pad(cells.OrDef(f.ID, "—"), w.id, theme.Paint(theme.Dim).Render)+
			ink.Render(cells.Pad(said[0], w.rule, false)),
		pad(cells.Tail(where(f, e), w.where), w.where, theme.Text(theme.Secondary).Render)+
			theme.Paint(theme.Dim).Render(cells.Pad(when(f.At), w.when, false)), cw)}

	lead := strings.Repeat(" ", gutter+w.id+colGap)
	for _, line := range said[1:] {
		rows = append(rows, lead+ink.Render(line))
	}

	return rows
}

// saidRow is a sentence in the tray, drawn in the same columns as a rule
// because it is one question away from being one.
func (s State) saidRow(one Said, chosen bool, w widths, cw int) []string {
	said := cells.Lines(one.Text, w.rule)
	if len(said) == 0 {
		said = []string{""}
	}

	rows := []string{cells.Spread(
		mark(chosen)+
			pad("—", w.id, theme.Paint(theme.Dim).Render)+
			theme.Text(theme.Primary).Render(cells.Pad(said[0], w.rule, false)),
		pad(cells.Tail(cells.OrDef(one.Where, "—"), w.where), w.where, theme.Text(theme.Secondary).Render)+
			theme.Paint(theme.Dim).Render(cells.Pad(when(one.At), w.when, false)), cw)}

	lead := strings.Repeat(" ", gutter+w.id+colGap)
	for _, line := range said[1:] {
		rows = append(rows, lead+theme.Text(theme.Primary).Render(line))
	}

	return rows
}

// many is whether the board has more than one checkout on it, which is the
// question that decides how a place has to be named.
func many(e Env) bool { return len(e.Repos) > 1 }

// where is how far a rule reaches, named so that it cannot be mistaken on
// the board it is being read on.
//
// With one checkout the path is the whole answer. With several, the same
// path exists in all of them — a rule about migrations says nothing about
// which project's migrations — so the checkout's name goes in front of it.
func where(f knowledge.Fact, e Env) string {
	// Everywhere is the one scope with no path behind it, so it is the one
	// whose name is a word rather than a string off the record — and a word
	// is said in the reader's own language.
	if f.Scope.Kind == knowledge.General {
		return e.Words.T("knowledge.place_all", "everywhere")
	}

	named := fact.Where(f.Scope)
	if !many(e) || f.Scope.Repo == "" || f.Scope.Kind == knowledge.Repo {
		return named
	}

	return fact.Repo(f.Scope.Repo) + "/" + named
}

// mark is the two cells a row opens with.
func mark(chosen bool) string {
	if chosen {
		return theme.Paint(theme.Accent).Bold(true).Render(cells.Mark + " ")
	}

	return strings.Repeat(" ", gutter)
}

// when is a day, and nothing at all for a rule that has no date on it.
//
// The date and not "three days ago": a rule is not read in a hurry, and a
// date can be looked up against a pull request or a task while a relative
// age cannot. It is the day it was written down as the record has it, and
// not the reader's own — two people reading one repository's rules should
// see the same column.
func when(at time.Time) string {
	if at.IsZero() {
		return ""
	}

	return at.UTC().Format(time.DateOnly)
}

// gutterAll is rowsFit under the name the screens that are not the list call
// it by: every row indented into the margin the window keeps, and cut to it.
func gutterAll(rows []string, w int) []string { return rowsFit(rows, w) }

// rowsFit indents every row and cuts it to the window.
func rowsFit(rows []string, w int) []string {
	for i, row := range rows {
		rows[i] = cells.Fit("  "+row, w)
	}

	return rows
}

// foot is the line that says what the keys do, held against the bottom.
func (s State) foot(cw int, e Env) []string {
	return append([]string{""}, dimmed(cells.Lines(s.ways(e.Words), cw))...)
}
