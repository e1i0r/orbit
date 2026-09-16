package known

// The foot of the screen: the box a rule is written in, the box its history
// is read in, and the line that says what the keys do.
//
// A box, and the fields in a column inside it. The three things a rule is
// made of used to be three sentences run one under another with a label
// inline on each, and the labels were different lengths — so nothing lined
// up, and nothing said where one field ended and the next began. Somebody
// who cannot see the shape of a form cannot fill it in, and this is the form
// the whole of what Orbit knows is typed into.

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/fact"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// listLeast is how many rows of the list are kept whatever is open under it.
// Two, because one is a rule's first line with its sentence cut off, and a
// form with nothing above it has lost what it is about.
const listLeast = 2

// foot is what is under the list: the form while something is being typed,
// the history while a rule is being decided about, and otherwise the one
// line that says what the keys do.
//
// It is given a height because it can want more rows than the terminal has.
// What goes first is the sentence saying what the gesture does — it is there
// to be read once and is the only part somebody already mid-gesture does not
// need — and never the fields or the keys, which are what the reader is
// actually doing.
func (s State) foot(cw, h int, e Env) []string {
	full := s.footAt(cw, e, true)
	if len(full)+len(s.head(cw, e))+listLeast <= h {
		return full
	}

	return s.footAt(cw, e, false)
}

// footAt is the foot with or without the sentence that says what is going on.
func (s State) footAt(cw int, e Env, why bool) []string {
	if s.editing {
		return append([]string{""}, s.form(cw, e, why)...)
	}

	if s.reviewing {
		return append([]string{""}, s.history(cw, e, why)...)
	}

	return append([]string{""}, dimmed(cells.Lines(s.ways(e.Words), cw))...)
}

// form is the box a rule is written in.
func (s State) form(cw int, e Env, sayWhy bool) []string {
	p := e.Words

	title, why := s.formHead(e)
	if !sayWhy {
		why = ""
	}

	fields := s.fields(e)
	wide := 0

	for _, f := range fields {
		wide = max(wide, lipgloss.Width(f.label))
	}

	var body []string

	for _, f := range fields {
		body = append(body, s.formRow(f, wide, cw-boxEdges)...)
	}

	// The hint rides the bottom border, where compose puts the same thing:
	// it is about the field being typed into rather than about the form, so
	// it reads as well from below and costs no row of its own.
	out := box(title, why, body, fields[s.at(fields)].hint, cw)

	ways := p.T("knowledge.editing_ways",
		"[tab] the next field · [↵] save · [esc] leave it as it was")
	if len(fields) == 1 {
		// A pause is one sentence, and a form with one field that offers a
		// key for reaching the next one is a form promising a field that is
		// not there.
		ways = p.T("knowledge.typing_ways", "[↵] save · [esc] leave it as it was")
	}

	return append(out, theme.Paint(theme.Dim).Render(cells.Fit("  "+ways, cw)))
}

// history is the box a rule's story is read in, above the keys that decide
// about it.
//
// The evidence first and the decisions under it, in that order, because the
// whole point of this screen is that nothing is decided before it is read.
func (s State) history(cw int, e Env, sayWhy bool) []string {
	p := e.Words

	lines := s.story(e)
	if len(lines) == 0 {
		lines = []string{p.T("knowledge.nothing_happened", "nothing has happened to this one yet")}
	}

	var body []string

	for _, one := range lines {
		body = append(body, cells.Lines(one, cw-boxEdges)...)
	}

	why := ""
	if sayWhy {
		why = p.T("knowledge.form_review_why",
			"Everything this rule has put you through. There is no score — only the friction.")
	}

	return append(box(s.ruleTitle(e), why, body, "", cw),
		dimmed(cells.Lines("  "+s.ways(p), cw))...)
}

// asking is what the box says it is about: the rule on its top border, and
// under that the one sentence that says what this gesture does to it.
//
// The gesture and not the rule. Somebody who pressed p and somebody who
// pressed e are looking at the same three fields and mean opposite things by
// them, and a form that looked identical either way is a form that gets
// filled in wrong.
func (s State) formHead(e Env) (title, why string) {
	p := e.Words

	if s.pausing {
		return p.T("knowledge.form_pausing", "Pausing {rule}",
				about("rule", s.name(e))),
			p.T("knowledge.form_pausing_why",
				"It stops applying, and goes to be looked at again.")
	}

	if one, waiting := s.onSaid(); waiting {
		return p.T("knowledge.form_keeping", "Keeping what you said · {from}",
				about("from", cells.OrDef(one.From, one.At.Local().Format(saidWhen)))),
			p.T("knowledge.form_keeping_why",
				"Correct the words if they were not quite it, and put it where it is true.")
	}

	f, ok := s.onFact()
	if !ok || f.Phrase == "" {
		return p.T("knowledge.form_new", "A new rule"),
			p.T("knowledge.form_new_why",
				"Say what is true about the code, in the words a run should be told it in.")
	}

	return s.ruleTitle(e),
		p.T("knowledge.form_correcting_why",
			"Correcting a rule. It keeps the name it has had all along.")
}

// ruleTitle is the rule under the cursor named on a border: what it is
// called, how far it reaches, and where it came from.
func (s State) ruleTitle(e Env) string {
	f, ok := s.onFact()
	if !ok {
		return s.name(e)
	}

	return s.name(e) + cells.Dot + fact.Where(f.Scope) + cells.Dot + from(f, e)
}

// name is what the rule under the cursor is called, and a placeholder for
// one Orbit has not written down yet.
func (s State) name(e Env) string {
	f, ok := s.onFact()
	if !ok || f.ID == "" {
		return e.Words.T("knowledge.no_name_yet", "this rule")
	}

	return f.ID
}

// aField is one line of the form: what it is called, which of the typed
// fields it is, and the sentence that says what belongs in it.
type aField struct {
	label, hint string
	which       int
}

// fields is the form's rows, which are not the same rows for every gesture.
// A pause is one sentence and has no check and no place — showing the other
// two would be offering to change things this gesture does not touch.
func (s State) fields(e Env) []aField {
	p := e.Words

	if s.pausing {
		return []aField{{
			label: p.T("knowledge.field_why", "what for"),
			hint:  p.T("knowledge.hint_why", "a pause with no reason is a switch under another name"),
			which: factPhrase,
		}}
	}

	return []aField{
		{
			label: p.T("knowledge.field_phrase", "what it says"),
			hint:  p.T("knowledge.hint_phrase", "the sentence every run is told before it starts work"),
			which: factPhrase,
		},
		{
			label: p.T("knowledge.field_check", "the check"),
			hint: p.T("knowledge.hint_check", "a command that answers yes or no. "+
				"Leave it empty and the rule only says its sentence."),
			which: factCheck,
		},
		{
			label: p.T("knowledge.field_where", "where"),
			hint: p.T("knowledge.hint_where",
				"the folder or file it is about; empty is the whole checkout"),
			which: factWhere,
		},
	}
}

// field is one row of the form: the label in its column, and the value in
// the rest of the width, wrapped under itself rather than cut.
func (s State) formRow(f aField, wide, cw int) []string {
	in := s.in[f.which]
	on := f.which == s.field

	ink := theme.Paint(theme.Dim)
	if on {
		ink = theme.Paint(theme.Accent)
	}

	room := max(cw-wide-labelGap, 8)

	text := in.Val
	if on {
		text = withCaret(in.Val, in.At)
	}

	lines := cells.Lines(text, room)
	if len(lines) == 0 {
		lines = []string{""}
	}

	// The caret is an escape sequence in the middle of the value, so the
	// first line is wrapped before it is painted and the caret put back on
	// whichever line it landed on.
	out := []string{
		ink.Render(cells.Pad(f.label, wide, false)) +
			strings.Repeat(" ", labelGap) + theme.Text(theme.Primary).Render(lines[0]),
	}

	for _, line := range lines[1:] {
		out = append(out, strings.Repeat(" ", wide+labelGap)+theme.Text(theme.Primary).Render(line))
	}

	return out
}

// The box the form is drawn in: two cells of border and one of air each
// side, and the space between a label and its value.
const (
	boxEdges = 4
	labelGap = 2
)

// box draws one: the title on the top border, the sentence that says what it
// is for under it, the body below that, and a hint on the bottom border.
func box(title, why string, body []string, hint string, cw int) []string {
	line := theme.Paint(theme.Dim)
	inner := max(cw-boxEdges, 8)

	out := []string{edge("┌", "┐", theme.Text(theme.Primary).Render(title), cw)}

	if why != "" {
		for _, one := range cells.Lines(why, inner) {
			out = append(out, boxRow(theme.Paint(theme.Dim).Render(one), inner, line))
		}

		out = append(out, boxRow("", inner, line))
	}

	for _, one := range body {
		out = append(out, boxRow(one, inner, line))
	}

	return append(out, edge("└", "┘", hint, cw))
}

// edge is a border with a word set into it: "┌─ what this is ─────┐". The
// title goes on the top and the hint on the bottom, so neither costs a row
// of a terminal that has none to spare — and the box stays a rectangle,
// which is the whole of why it is drawn at all.
func edge(left, right, set string, cw int) string {
	line := theme.Paint(theme.Dim)
	if set == "" {
		return line.Render(left + strings.Repeat("─", max(cw-2, 0)) + right)
	}

	set = cells.Fit(set, max(cw-8, 4))
	rule := max(cw-5-lipgloss.Width(set), 0)

	return line.Render(left+"─ ") + set + line.Render(" "+strings.Repeat("─", rule)+right)
}

// boxRow is one line between the borders, filled out so that the right edge
// is a straight line down the screen.
func boxRow(text string, inner int, line lipgloss.Style) string {
	return line.Render("│ ") + cells.PadRight(text, inner) + line.Render(" │")
}

// ways is what the keys do, which is not the same sentence in the tray as it
// is under it. The two halves of the screen answer different questions, and
// a bar that listed the keys of both would be a bar nobody reads.
func (s State) ways(p *words.Printer) string {
	if s.reviewing {
		return p.T("knowledge.review_ways",
			"[c] say it better or move it · [o] decide against it · [u] have it apply again · [esc] back")
	}

	if _, waiting := s.onSaid(); waiting {
		return p.T("knowledge.tray_ways",
			"[↑↓] move · [k] keep it · [e] keep it in better words · [d] not a rule · [esc] back")
	}

	return p.T("knowledge.ways",
		"[↑↓] move · [r] decide about it · [p] pause it · [n] new · [esc] back")
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
