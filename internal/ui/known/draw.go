package known

// What the Knowledge screen draws.
//
// Three parts, and the middle one is the only one that moves: a head that
// says what is here, a list that scrolls, and a foot that says what the keys
// do or holds the form. The head and the foot are pinned because a reader
// going down forty rules is exactly the reader who needs to be told what
// [r] does, and a hint that scrolled off the bottom is a hint nobody has.

import (
	"strconv"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// rows is the whole screen.
func (s State) rows(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	cw := content(w)

	if len(s.facts) == 0 && len(s.waiting) == 0 {
		return cells.Fill(rowsFit(s.nothing(cw, e), w), h)
	}

	above, below := s.head(cw, e), s.foot(cw, h, e)
	body, _ := s.body(cw, e)

	room := max(1, h-len(above)-len(below))
	off := s.window(len(body), room)

	out := append([]string{}, above...)
	for i := off; i < len(body) && i < off+room; i++ {
		out = append(out, body[i])
	}

	// The foot is held against the bottom of the screen rather than run on
	// after the last rule, so the line that says what the keys do is in the
	// same place on a screen of three rules and a screen of forty.
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

// head is the title, and what the screen holds: how many rules, and how many
// of them are waiting on somebody.
//
// The second half is why the title row exists at all. A reader opens this
// screen with one question — is there anything here for me — and the answer
// is a number they should not have to count.
func (s State) head(cw int, e Env) []string {
	p := e.Words

	title := theme.Paint(theme.Accent).Render(p.T("knowledge.title", "What Orbit knows"))

	tally := theme.Paint(theme.Dim).Render(
		p.P("knowledge.tally", len(s.facts), "{n} rule", "{n} rules",
			about("n", strconv.Itoa(len(s.facts)))))

	if n := s.asking(); n > 0 {
		tally += theme.Paint(theme.Dim).Render(cells.Dot) + theme.Paint(theme.Warn).Render(
			p.P("knowledge.tally_waiting", n, "{n} waiting on you", "{n} waiting on you",
				about("n", strconv.Itoa(n))))
	}

	return []string{"", cells.Spread(title, tally, cw), ""}
}

// asking is how many things on this screen want an answer: the sentences
// nobody has decided about, and the rules sent to be looked at again.
func (s State) asking() int {
	n := len(s.waiting)

	for _, f := range s.facts {
		if f.Review {
			n++
		}
	}

	return n
}

// body is everything that scrolls, and the line each row the cursor can be
// on begins at — the tray first, then the rules, in the order the cursor
// walks them.
func (s State) body(cw int, e Env) ([]string, []span) {
	p := e.Words
	w := plan(cw)

	out, at := s.tray(nil, nil, cw, e)

	rootless, owned := s.ordered()
	if len(rootless)+len(owned) > 0 {
		out = append(out, heading(w, e))
	}

	out, at = s.group(out, at, p.T("knowledge.general",
		"Everywhere · this machine only, these do not travel"), rootless, w, e)

	for _, repo := range byRepo(owned) {
		out, at = s.group(out, at, repo.name+cells.Dot+p.T("knowledge.travels",
			"travels with the repository"), repo.facts, w, e)
	}

	return out, at
}

// tray draws what you said that nobody has answered yet.
//
// Above the rules and not below them. This is the one part of the screen
// with a question in it, and a question under two pages of facts is a
// question nobody answers.
func (s State) tray(out []string, at []span, cw int, e Env) ([]string, []span) {
	if len(s.waiting) == 0 {
		return out, at
	}

	for _, line := range cells.Lines(e.Words.T("knowledge.said",
		"You said this · keep it and Orbit applies it, or say it was not a rule"), cw) {
		out = append(out, theme.Paint(theme.Warn).Render(line))
	}

	for i, one := range s.waiting {
		row := s.sentence(one, i == s.sel, cw)
		at = append(at, span{from: len(out), rows: len(row)})
		out = append(out, row...)
	}

	return append(out, ""), at
}

// saidWhen is when a sentence was said, to the minute. The seconds were what
// the record needed to name it by and nothing a reader wants.
const saidWhen = "2006-01-02 15:04"

// sentence is one row of the tray: when and where it was said, and what was
// said under it. It is not drawn in the table's columns because it has none
// of them yet — what it does, where it stands and where it reaches are all
// decided by keeping it.
func (s State) sentence(one Said, chosen bool, cw int) []string {
	head := one.At.Local().Format(saidWhen)
	if one.From != "" {
		head += cells.Dot + one.From
	}

	if one.Where != "" {
		head += cells.Dot + one.Where
	}

	rows := []string{saidMark(chosen) + theme.Paint(theme.Dim).Render(head)}

	for _, line := range cells.Lines(one.Text, max(cw-4, 8)) {
		rows = append(rows, "    "+theme.Text(theme.Primary).Render(line))
	}

	return rows
}

// saidMark is the gutter of a tray row: the cursor, or the dot that says
// this one is waiting — which every sentence in the tray is.
func saidMark(chosen bool) string {
	if chosen {
		return theme.Paint(theme.Accent).Bold(true).Render(cells.Mark + " ")
	}

	return theme.Paint(theme.Warn).Render("• ")
}

// group draws one heading and the rules under it, and nothing when there are
// none — an empty heading is a question a reader has to answer for
// themselves.
func (s State) group(
	out []string, at []span, head string, facts []knowledge.Fact, w widths, e Env,
) ([]string, []span) {
	if len(facts) == 0 {
		return out, at
	}

	out = append(out, theme.Paint(theme.Dim).Render(cells.Fit(head, w.left()+w.says)))

	for _, f := range facts {
		row := s.factRow(f, len(at) == s.sel, w, e)
		at = append(at, span{from: len(out), rows: len(row)})
		out = append(out, row...)
	}

	return append(out, ""), at
}

// repoFacts is one repository's name and what is known about it.
type repoFacts struct {
	name  string
	facts []knowledge.Fact
}

// byRepo groups the facts by the checkout they belong to, keeping the order
// they arrived in so that two runs of the screen read the same.
func byRepo(facts []knowledge.Fact) []repoFacts {
	var (
		out  []repoFacts
		seen = map[string]int{}
	)

	for _, f := range facts {
		name := factRepo(f)

		i, held := seen[name]
		if !held {
			i = len(out)
			seen[name] = i
			out = append(out, repoFacts{name: name})
		}

		out[i].facts = append(out[i].facts, f)
	}

	return out
}

// rowsFit indents every row and cuts it to the window.
func rowsFit(rows []string, w int) []string {
	for i, row := range rows {
		rows[i] = cells.Fit("  "+row, w)
	}

	return rows
}
