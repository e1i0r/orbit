package known

// The foot of the screen: the line being typed into, the history of the rule
// being decided about, and the keys that act on what the cursor is on.
//
// Pinned to the bottom rather than run on after the last rule. What the keys
// do has to be on screen while somebody is going down a list looking for the
// rule that is in their way, and a hint that scrolled off the end is a hint
// nobody has.

import (
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// told is what the open rule has put you through, above the keys that decide
// about it.
//
// The evidence first and the decisions under it, in that order, because the
// whole point of this screen is that nothing is decided before it is read.
func (s State) told(cw int, e Env) []string {
	if !s.reviewing {
		return nil
	}

	lines := s.story(e)
	if len(lines) == 0 {
		return []string{theme.Paint(theme.Dim).Render(cells.Fit(e.Words.T("knowledge.nothing_happened",
			"nothing has happened to this one yet"), cw)), ""}
	}

	out := make([]string, 0, len(lines)+1)
	for _, one := range lines {
		out = append(out, theme.Text(theme.Primary).Render(cells.Fit("  "+one, cw)))
	}

	return append(out, "")
}

// foot is the line being corrected, or the ways out when nothing is.
//
// It is given the height of the screen because what it draws is what is left
// for the list above it, and a foot that grew past the terminal would take
// the rules with it.
func (s State) foot(cw, h int, e Env) []string {
	p := e.Words

	out := append([]string{""}, s.told(cw, e)...)

	if !s.editing {
		return append(out, dimmed(cells.Lines(s.ways(p), cw))...)
	}

	return append(out,
		s.line(p.T("knowledge.field_phrase", "what it says"), factPhrase, cw),
		s.line(p.T("knowledge.field_check", "the check that makes it stop"), factCheck, cw),
		s.line(p.T("knowledge.field_where", "the folder or file, if it is about one"), factWhere, cw),
		"",
		theme.Paint(theme.Dim).Render(cells.Fit(p.T("knowledge.editing_ways",
			"[tab] the other field · [↵] save · [esc] leave it as it was"), cw)),
	)
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
