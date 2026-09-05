package ui

// What the Knowledge screen draws.

import (
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// knowledgeRows is the whole screen: a title, the facts that belong to no
// repository, then each repository's own.
func (m Model) knowledgeRows(h, w int) []string {
	if h <= 0 {
		return nil
	}

	p := m.opts.Words
	cw := max(min(w-4, 110), 24)

	out := []string{"", Paint(Accent).Render(p.T("knowledge.title", "What Orbit knows")), ""}

	if len(m.knowledge.facts) == 0 {
		out = append(out, Paint(Dim).Render(fit(p.T("knowledge.empty",
			"Nothing written down yet. Say /rule or /aware to the supervisor, or drop a file in .orbit/knowledge/."), cw)))

		return fill(rowsFit(out, w), h)
	}

	rootless, owned := m.orderedKnowledge()

	at := 0
	out, at = m.knowledgeGroup(out, p.T("knowledge.general",
		"Everywhere · this machine only, these do not travel"), rootless, at, cw)

	for _, repo := range knowledgeRepos(owned) {
		out, at = m.knowledgeGroup(out, repo.name+" · "+p.T("knowledge.travels",
			"travels with the repository"), repo.facts, at, cw)
	}

	out = append(out, "")
	out = append(out, m.knowledgeFoot(cw)...)

	return fill(rowsFit(out, w), h)
}

// knowledgeGroup draws one heading and the facts under it, and nothing when
// there are none — an empty heading is a question a reader has to answer for
// themselves.
func (m Model) knowledgeGroup(out []string, head string, facts []knowledge.Fact, at, cw int) ([]string, int) {
	if len(facts) == 0 {
		return out, at
	}

	out = append(out, Paint(Dim).Render(fit(head, cw)))

	for _, f := range facts {
		out = append(out, m.knowledgeFact(f, at == m.knowledge.sel, cw)...)
		at++
	}

	return append(out, ""), at
}

// repoFacts is one repository's name and what is known about it.
type repoFacts struct {
	name  string
	facts []knowledge.Fact
}

// knowledgeRepos groups the facts by the checkout they belong to, keeping
// the order they arrived in so that two runs of the screen read the same.
func knowledgeRepos(facts []knowledge.Fact) []repoFacts {
	var (
		out  []repoFacts
		seen = map[string]int{}
	)

	for _, f := range facts {
		name := shortRepo(f.Scope.Repo)

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

// knowledgeFact is one fact: what it does and where it reaches on the first
// line, the sentence under it, and where it came from at the end.
func (m Model) knowledgeFact(f knowledge.Fact, chosen bool, cw int) []string {
	mark := "  "
	if chosen {
		mark = Paint(Accent).Bold(true).Render("▸ ")
	}

	head := strings.Join(nonEmpty(
		m.factDoes(f),
		Paint(Dim).Render(sideWhere(f.Scope)),
		m.factFrom(f),
	), Paint(Dim).Render(" · "))

	rows := []string{mark + head}

	ink := Text(Primary)
	if f.Off {
		ink = Paint(Dim)
	}

	for _, line := range splitIntoLines(f.Phrase, max(cw-4, 8)) {
		rows = append(rows, "    "+ink.Render(line))
	}

	return rows
}

// factDoes is what a fact does, in the words the screen uses for it: a rule
// that can enforce itself, one that cannot yet, or a sentence that advises.
func (m Model) factDoes(f knowledge.Fact) string {
	p := m.opts.Words

	switch {
	case f.Off:
		return Paint(Dim).Render(p.T("knowledge.is_off", "off"))
	case f.Action() == knowledge.Stops:
		return Paint(Bad).Bold(true).Render(p.T("knowledge.stops", "stops"))
	case f.Stops:
		return Paint(Warn).Render(p.T("knowledge.no_check", "rule, no check yet"))
	default:
		return Paint(Live).Render(p.T("knowledge.warns", "aware"))
	}
}

// factFrom is where a fact came from, and how much use it has had.
//
// The source is the whole of why a fact can be trusted: one read off the code
// regenerates itself, one somebody said is somebody's opinion with a name on
// it, and one the record produced came out of a refusal that actually
// happened. A sentence nobody can trace is one the model may as well have
// made up.
func (m Model) factFrom(f knowledge.Fact) string {
	p := m.opts.Words

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

	return Paint(Dim).Render(said)
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
		rows[i] = fit("  "+row, w)
	}

	return rows
}

// knowledgeFoot is the line being corrected, or the ways out when nothing is.
//
// The two fields are drawn together rather than one at a time: what a rule
// says and what makes it stop are one thought, and somebody adding a check is
// reading the sentence it belongs to while they type it.
func (m Model) knowledgeFoot(cw int) []string {
	p := m.opts.Words

	if !m.knowledge.editing {
		return []string{Paint(Dim).Render(fit(p.T("knowledge.ways",
			"[↑↓] move · [e] edit · [space] turn off · [esc] back"), cw))}
	}

	return []string{
		m.factField(p.T("knowledge.field_phrase", "what it says"), factPhrase, cw),
		m.factField(p.T("knowledge.field_check", "the check that makes it stop"), factCheck, cw),
		"",
		Paint(Dim).Render(fit(p.T("knowledge.editing_ways",
			"[tab] the other field · [↵] save · [esc] leave it as it was"), cw)),
	}
}

// factField is one line being typed into, with the caret where the next
// character will land.
func (m Model) factField(label string, field, cw int) string {
	in := m.knowledge.in[field]

	text := in.val
	if field == m.knowledge.field {
		text = withCaret(in.val, in.at)
	}

	ink := Paint(Dim)
	if field == m.knowledge.field {
		ink = Paint(Accent)
	}

	return fit(ink.Render(label+": ")+Text(Primary).Render(text), cw)
}

// withCaret puts the block where the caret is, which is at the end of the line
// as often as not.
func withCaret(s string, at int) string {
	runes := []rune(s)
	at = min(max(at, 0), len(runes))

	if at == len(runes) {
		return s + Paint(Accent).Render("█")
	}

	return string(runes[:at]) + Paint(Accent).Render(string(runes[at])) + string(runes[at+1:])
}
