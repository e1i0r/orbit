package known

// Writing a rule down, in the shape the flow designer already uses: groups
// under a heading and a rule, a column of labels, and the options of each row
// beside it with the one it is holding lit.
//
// The same shape and not a second one. Somebody who has filled in one form in
// this window has learnt where the labels sit, that the mark on the left is
// where they are, that a lit pill is the choice and the grey ones beside it
// are the others, and that the green line at the bottom is about the row they
// are on. A screen that answered those differently would teach them twice for
// nothing.

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/ui/typing"
	"github.com/e1i0r/orbit/internal/words"
)

// labelWidth is how wide the left column is, and boxRows how tall the well
// the sentence is written in stands. A rule is a sentence and sometimes two,
// and a line that scrolled sideways is a sentence nobody can re-read.
const (
	labelWidth = 22
	boxRows    = 2
)

// formRows is the whole screen while a rule is being written: the heading,
// the rows between, and the sentence about the row somebody is on.
func (s State) formRows(h, w int, e Env) []string {
	cw := content(w)
	rows := s.rows2(e)

	above := s.formAbove(cw, e)
	below := s.beneath(rows, cw, e)

	var (
		body []string
		here span
	)

	if r, up := s.picked(e); up {
		return cells.Fill(gutterAll(framed(above, s.pickBox(r, cw, e), below, h, 0), w), h)
	}

	for _, g := range s.groups(rows, e) {
		if g.head != "" {
			body = append(body, group(g.head, cw))
		}

		if g.buttons {
			// Side by side, the way every other form in the window ends.
			// Stacked, two answers to one question read as two questions.
			if s.at(g.rows) >= 0 {
				here = span{from: len(body), rows: 1}
			}

			body = append(body, cells.Fit(s.buttonRow(g.rows), cw))
		}

		for _, r := range g.rows {
			if g.buttons {
				break
			}

			drawn := s.formRow(r, cw, e)
			if r.which == s.field {
				here = span{from: len(body), rows: len(drawn)}
			}

			body = append(body, drawn...)
		}

		body = append(body, "")
	}

	off := deepEnough(s.deep, here.from, here.rows,
		max(1, h-len(above)-len(below)), len(body))

	return cells.Fill(gutterAll(framed(above, body, below, h, off), w), h)
}

// formAbove is the two rows every form opens with, and the air under them.
func (s State) formAbove(cw int, e Env) []string {
	return []string{"", s.formHead(cw, e), ""}
}

// formHead is what the form says it is doing, with what that means beside it.
//
// The gesture and not the rule. Somebody who pressed p and somebody who
// pressed e are looking at the same rows and mean opposite things by them,
// and a form that looked identical either way is a form filled in wrong.
func (s State) formHead(cw int, e Env) string {
	p := e.Words

	title, why := p.T("knowledge.form_new", "A new rule"),
		p.T("knowledge.form_new_why", "a sentence about your code, told to every run before it works")

	switch one, waiting := s.onSaid(); {
	case s.pausing:
		title = p.T("knowledge.form_pausing", "Pausing {rule}", about("rule", s.name(e)))
		why = p.T("knowledge.form_pausing_why", "it stops applying, and goes to be looked at again")
	case waiting:
		title = p.T("knowledge.form_keeping", "Keeping what you said")
		why = p.T("knowledge.form_keeping_why", "said at {where} · correct the words if they were not quite it",
			about("where", cells.OrDef(one.From, when(one.At))))
	case s.held(aRow{which: rowPhrase, typed: factPhrase}) != "" && !s.fresh:
		title = p.T("knowledge.form_correcting", "Correcting {rule}", about("rule", s.name(e)))
		why = p.T("knowledge.form_correcting_why", "it keeps the name it has had all along")
	}

	return theme.Paint(theme.Accent).Bold(true).Render(title) + "  " +
		theme.Text(theme.Tertiary).Render(cells.Fit(why, max(cw-lipgloss.Width(title)-2, 0)))
}

// aGroup is a heading and the rows under it.
type aGroup struct {
	head string
	rows []aRow
	// buttons says the rows are answers rather than questions, and belong
	// on one line.
	buttons bool
}

// groups is the form's rows under their headings. A pause is one sentence
// and needs none of them: it has one question and a heading over one question
// is a heading nobody reads.
func (s State) groups(rows []aRow, e Env) []aGroup {
	p := e.Words

	if s.pausing {
		var ask, done []aRow

		for _, r := range rows {
			if r.button != "" {
				done = append(done, r)
				continue
			}

			ask = append(ask, r)
		}

		return []aGroup{{rows: ask}, {rows: done, buttons: true}}
	}

	var said, gate, done []aRow

	for _, r := range rows {
		switch {
		case r.button != "":
			done = append(done, r)
		case r.which == rowDoes || r.which == rowCheck:
			gate = append(gate, r)
		default:
			said = append(said, r)
		}
	}

	return []aGroup{
		{head: p.T("knowledge.group_rule", "THE RULE · what it says, and what it is about"), rows: said},
		{head: p.T("knowledge.group_gate",
			"THE GATE · whether it also blocks the work, or only says it"), rows: gate},
		{rows: done, buttons: true},
	}
}

// buttonRow is the form's answers, side by side, with the one under the
// cursor lit.
func (s State) buttonRow(rows []aRow) string {
	var out []string

	for _, r := range rows {
		out = append(out, s.mark(r)+button(r.button, r.which == s.field))
	}

	return "  " + strings.Join(out, "  ")
}

// group is one heading with a rule after it, so the eye finds the next group
// without reading the words again.
func group(head string, cw int) string {
	line := theme.Paint(theme.Live).Bold(true).Render(head) + " "

	if rule := cw - lipgloss.Width(line); rule > 0 {
		line += theme.Paint(theme.Dim).Render(strings.Repeat("─", rule))
	}

	return cells.Fit(line, cw)
}

// formRow is one line of the form: the mark, the label in its column, and
// what the row holds — its options, or a button.
func (s State) formRow(r aRow, cw int, e Env) []string {
	if r.button != "" {
		return []string{s.mark(r) + button(r.button, r.which == s.field)}
	}

	lead := s.mark(r) + label(r.label, r.which == s.field)

	if r.which == rowPhrase {
		// The sentence is written in a well of its own under its label. It
		// is the one row that is prose, and prose on a line that scrolls
		// sideways is prose nobody can re-read while they write it.
		return append([]string{cells.Fit(lead, cw)}, s.well(cw, e)...)
	}

	held := s.value(r, cw, e)
	out := []string{cells.Fit(lead+held[0], cw)}

	// A row whose options ran to a second line keeps them in their own
	// column, so the label column stays a column.
	for _, line := range held[1:] {
		out = append(out, cells.Fit(strings.Repeat(" ", labelWidth+3)+line, cw))
	}

	return out
}

// beneath holds the foot against the bottom of the screen: the sentence
// about the row the reader is on, and the keys.
//
// Green for the one and grey for the other, which is the designer's rule:
// one of the two is about what is under the cursor right now and the other is
// always true, and a reader learns in one screen which is which.
func (s State) beneath(rows []aRow, cw int, e Env) []string {
	foot := []string{""}

	if at := s.at(rows); at >= 0 && rows[at].hint != "" {
		for _, line := range cells.Lines("  "+rows[at].hint, cw) {
			foot = append(foot, theme.Paint(theme.Live).Render(line))
		}
	}

	return append(foot, dimmed(cells.Lines("  "+e.Words.T("knowledge.editing_ways",
		"[tab] next row · [↑↓] move · [←→] change · [↵] do it · [esc] back"), cw))...)
}

// at is which row the cursor is on, and -1 when it is on none of them.
func (s State) at(rows []aRow) int {
	for i, r := range rows {
		if r.which == s.field {
			return i
		}
	}

	return -1
}

// name is what the rule under the cursor is called.
//
// A rule Orbit has not written down yet has none, so it is called by what it
// says: a screen headed "this rule" tells a reader nothing, and the sentence
// is the one thing every rule always has.
func (s State) name(e Env) string {
	f, ok := s.onFact()

	switch {
	case ok && f.ID != "":
		return f.ID
	case ok && f.Phrase != "":
		return "“" + f.Phrase + "”"
	}

	return e.Words.T("knowledge.no_name_yet", "no name yet")
}

// ways is what the keys do, which is not the same sentence in the tray as it
// is under it. The two halves of the list answer different questions, and a
// bar that listed the keys of both would be a bar nobody reads.
func (s State) ways(p *words.Printer) string {
	if _, waiting := s.onSaid(); waiting {
		return p.T("knowledge.tray_ways",
			"[↑↓] move · [↵] look at it · [k] keep it as said · [d] not a rule · [esc] back")
	}

	return p.T("knowledge.ways",
		"[↑↓] move · [↵] open it · [p] pause it · [n] a new one · [esc] back")
}

// dimmed paints every line of a hint, which wraps rather than being cut: a
// list of keys with its end trimmed off is a list missing the key somebody
// was looking for.
func dimmed(lines []string) []string {
	for i, line := range lines {
		lines[i] = theme.Paint(theme.Dim).Render(line)
	}

	return lines
}

// oneLine is a field holding one value, with the caret after it.
func oneLine(val string) typing.Field { return typing.New(val) }

// withCaret puts the block where the caret is, which is at the end of the
// line as often as not.
func withCaret(s string, at int) string {
	runes := []rune(s)
	at = min(max(at, 0), len(runes))

	if at == len(runes) {
		return s + theme.Paint(theme.Accent).Render("█")
	}

	return string(runes[:at]) + theme.Paint(theme.Accent).Render(string(runes[at])) + string(runes[at+1:])
}
