package known

// What the Knowledge screen draws.

import (
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/fact"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// rows is the whole screen: a title, the facts that belong to no
// repository, then each repository's own.
func (s State) rows(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	p := e.Words
	cw := max(min(w-4, 110), 24)

	out := []string{"", theme.Paint(theme.Accent).Render(p.T("knowledge.title", "What Orbit knows")), ""}

	if len(s.facts) == 0 && len(s.waiting) == 0 {
		out = append(out, theme.Paint(theme.Dim).Render(cells.Fit(p.T("knowledge.empty",
			"Nothing written down yet. Say /rule or /aware to the supervisor, or drop a file in .orbit/knowledge/."), cw)))

		return cells.Fill(rowsFit(out, w), h)
	}

	rootless, owned := s.ordered()

	out, at := s.tray(out, cw, e)

	out, at = s.group(out, p.T("knowledge.general",
		"Everywhere · this machine only, these do not travel"), rootless, at, cw, e)

	for _, repo := range byRepo(owned) {
		out, at = s.group(out, repo.name+" · "+p.T("knowledge.travels",
			"travels with the repository"), repo.facts, at, cw, e)
	}

	out = append(out, "")
	out = append(out, s.foot(cw, e)...)

	return cells.Fill(rowsFit(out, w), h)
}

// tray draws what you said that nobody has answered yet, and how many rows
// of it the cursor has to walk before it reaches the facts.
func (s State) tray(out []string, cw int, e Env) ([]string, int) {
	if len(s.waiting) == 0 {
		return out, 0
	}

	out = append(out, theme.Paint(theme.Warn).Render(cells.Fit(e.Words.T("knowledge.said",
		"You said this · keep it and Orbit applies it, or say it was not a rule"), cw)))

	for i, one := range s.waiting {
		out = append(out, s.sentence(one, i == s.sel, cw)...)
	}

	return append(out, ""), len(s.waiting)
}

// sentence is one row of the tray: when and where it was said, and what was
// said under it — the same shape a fact is drawn in, because it is about to
// be one.
func (s State) sentence(one Said, chosen bool, cw int) []string {
	mark := "  "
	if chosen {
		mark = theme.Paint(theme.Accent).Bold(true).Render("▸ ")
	}

	head := one.At.Local().Format(time.DateTime)
	if one.From != "" {
		head += " · " + one.From
	}

	if one.Where != "" {
		head += " · " + one.Where
	}

	rows := []string{mark + theme.Paint(theme.Dim).Render(head)}

	for _, line := range cells.Lines(one.Text, max(cw-4, 8)) {
		rows = append(rows, "    "+theme.Text(theme.Primary).Render(line))
	}

	return rows
}

// group draws one heading and the facts under it, and nothing when
// there are none — an empty heading is a question a reader has to answer for
// themselves.
func (s State) group(
	out []string, head string, facts []knowledge.Fact, at, cw int, e Env,
) ([]string, int) {
	if len(facts) == 0 {
		return out, at
	}

	out = append(out, theme.Paint(theme.Dim).Render(cells.Fit(head, cw)))

	for _, f := range facts {
		out = append(out, s.fact(f, at == s.sel, cw, e)...)
		at++
	}

	return append(out, ""), at
}

// repoFacts is one repository's name and what is known about it.
type repoFacts struct {
	name  string
	facts []knowledge.Fact
}

// byRepo groups the facts by the checkout they belong to, keeping
// the order they arrived in so that two runs of the screen read the same.
func byRepo(facts []knowledge.Fact) []repoFacts {
	var (
		out  []repoFacts
		seen = map[string]int{}
	)

	for _, f := range facts {
		name := fact.Repo(f.Scope.Repo)

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

// fact is one fact: what it does and where it reaches on the first
// line, the sentence under it, and where it came from at the end.
func (s State) fact(f knowledge.Fact, chosen bool, cw int, e Env) []string {
	mark := "  "
	if chosen {
		mark = theme.Paint(theme.Accent).Bold(true).Render("▸ ")
	}

	head := strings.Join(nonEmpty(
		s.does(f, e),
		theme.Paint(theme.Dim).Render(fact.Where(f.Scope)),
		s.from(f, e),
	), theme.Paint(theme.Dim).Render(" · "))

	rows := []string{mark + head}

	ink := theme.Text(theme.Primary)
	if f.Off {
		ink = theme.Paint(theme.Dim)
	}

	for _, line := range cells.Lines(f.Phrase, max(cw-4, 8)) {
		rows = append(rows, "    "+ink.Render(line))
	}

	return rows
}

// does is what a fact does, in the words the screen uses for it: a rule
// that can enforce itself, one that cannot yet, or a sentence that advises.
func (s State) does(f knowledge.Fact, e Env) string {
	p := e.Words

	switch {
	case f.Off:
		return theme.Paint(theme.Dim).Render(p.T("knowledge.is_off", "off"))
	case f.Action() == knowledge.Stops:
		return theme.Paint(theme.Bad).Bold(true).Render(p.T("knowledge.stops", "stops"))
	case f.Stops:
		return theme.Paint(theme.Warn).Render(p.T("knowledge.no_check", "rule, no check yet"))
	default:
		return theme.Paint(theme.Live).Render(p.T("knowledge.warns", "aware"))
	}
}

// from is where a fact came from, and how much use it has had.
//
// The source is the whole of why a fact can be trusted: one read off the code
// regenerates itself, one somebody said is somebody's opinion with a name on
// it, and one the record produced came out of a refusal that actually
// happened. A sentence nobody can trace is one the model may as well have
// made up.
func (s State) from(f knowledge.Fact, e Env) string {
	p := e.Words

	said := map[knowledge.Source]string{
		knowledge.FromCode:       p.T("knowledge.from_code", "from the code"),
		knowledge.Human:          p.T("knowledge.from_human", "you said it"),
		knowledge.FromRecord:     p.T("knowledge.from_record", "from the record"),
		knowledge.FromProduction: p.T("knowledge.from_prod", "from production"),
	}[f.Source]

	if f.Ref != "" {
		said += " · " + f.Ref
	}

	// The date and not "three days ago": a fact is not read in a hurry, and
	// a date can be looked up against a pull request or a task while a
	// relative age cannot.
	if !f.At.IsZero() {
		said += " · " + f.At.Format(time.DateOnly)
	}

	if f.Used > 0 {
		said += " · " + p.T("knowledge.used", "told {n}×", about("n", strconv.Itoa(f.Used)))
	}

	return theme.Paint(theme.Dim).Render(said)
}

// nonEmpty drops the parts that had nothing to say, so the separators do not
// end up next to each other.
func nonEmpty(parts ...string) []string {
	kept := make([]string, 0, len(parts))

	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}

	return kept
}

// rowsFit indents every row and cuts it to the window.
func rowsFit(rows []string, w int) []string {
	for i, row := range rows {
		rows[i] = cells.Fit("  "+row, w)
	}

	return rows
}

// foot is the line being corrected, or the ways out when nothing is.
//
// The two fields are drawn together rather than one at a time: what a rule
// says and what makes it stop are one thought, and somebody adding a check is
// reading the sentence it belongs to while they type it.
func (s State) foot(cw int, e Env) []string {
	p := e.Words

	if !s.editing {
		return []string{theme.Paint(theme.Dim).Render(cells.Fit(s.ways(p), cw))}
	}

	return []string{
		s.line(p.T("knowledge.field_phrase", "what it says"), factPhrase, cw),
		s.line(p.T("knowledge.field_check", "the check that makes it stop"), factCheck, cw),
		s.line(p.T("knowledge.field_where", "the folder or file, if it is about one"), factWhere, cw),
		"",
		theme.Paint(theme.Dim).Render(cells.Fit(p.T("knowledge.editing_ways",
			"[tab] the other field · [↵] save · [esc] leave it as it was"), cw)),
	}
}

// ways is what the keys do, which is not the same sentence in the tray as it
// is under it. The two halves of the screen answer different questions, and
// a bar that listed the keys of both would be a bar nobody reads.
func (s State) ways(p *words.Printer) string {
	if _, waiting := s.onSaid(); waiting {
		return p.T("knowledge.tray_ways",
			"[↑↓] move · [k] keep it · [e] keep it in better words · [d] not a rule · [esc] back")
	}

	return p.T("knowledge.ways",
		"[↑↓] move · [e] edit · [n] new · [←→] wider or narrower · [space] turn off · [esc] back")
}

// line is one field being typed into, with the caret where the next
// character will land.
func (s State) line(label string, field, cw int) string {
	in := s.in[field]

	text := in.Val
	if field == s.field {
		text = withCaret(in.Val, in.At)
	}

	ink := theme.Paint(theme.Dim)
	if field == s.field {
		ink = theme.Paint(theme.Accent)
	}

	return cells.Fit(ink.Render(label+": ")+theme.Text(theme.Primary).Render(text), cw)
}

// withCaret puts the block where the caret is, which is at the end of the line
// as often as not.
func withCaret(s string, at int) string {
	runes := []rune(s)
	at = min(max(at, 0), len(runes))

	if at == len(runes) {
		return s + theme.Paint(theme.Accent).Render("█")
	}

	return string(runes[:at]) + theme.Paint(theme.Accent).Render(string(runes[at])) + string(runes[at+1:])
}
