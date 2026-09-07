package markdown

// The inline marks inside one line: bold, code spans, and a link that is
// read as its own words rather than as the address behind them.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// Inline sets bold (**text**) and code (`text`) spans, and the
// words between them.
//
// The words between them are painted here rather than by wrapping the line
// in one style, because a span's own reset ends whatever style it was
// nested in and everything after it would fall back to the terminal's
// foreground.
func Inline(s string) string {
	var b strings.Builder

	rest := unlink(s)

	for {
		delim := firstDelim(rest)
		if delim == "" {
			break
		}

		before, tail, _ := strings.Cut(rest, delim)

		span, after, closed := strings.Cut(tail, delim)
		if !closed {
			break // an unpaired mark is a character somebody typed
		}

		if before != "" {
			b.WriteString(theme.Text(theme.Primary).Render(before))
		}

		if delim == "**" {
			b.WriteString(theme.Text(theme.Primary).Bold(true).Render(span))
		} else {
			b.WriteString(theme.Paint(theme.Accent).Render(span))
		}

		rest = after
	}

	if rest != "" {
		b.WriteString(theme.Text(theme.Primary).Render(rest))
	}

	return b.String()
}

// unlink turns [words](address) into the words.
//
// Nothing in a terminal can be clicked, so the address is not an affordance:
// it is a hundred characters of noise in the middle of a sentence, and a URL
// has nowhere to break, so wrapping puts half a path on one line and the
// rest on the next. What a reader needs is what the writer wrote around it.
//
// An address with no words shows the address, because a sentence pointing at
// nothing is worse than a long one.
//
// A bracket that opens nothing — the list `[a, b, c]`, the footnote `[1]` —
// is left exactly as it was typed: the loop only consumes a bracket it can
// see the whole of, and gives back everything it was holding otherwise.
func unlink(s string) string {
	var b strings.Builder

	rest := s

	for {
		before, tail, opened := strings.Cut(rest, "[")
		if !opened {
			break
		}

		words, address, closed := strings.Cut(tail, "](")
		if !closed {
			break
		}

		addr, after, ended := strings.Cut(address, ")")
		if !ended {
			break
		}

		b.WriteString(before)

		if strings.TrimSpace(words) == "" {
			words = strings.TrimSpace(addr)
		}

		b.WriteString(words)

		rest = after
	}

	b.WriteString(rest)

	return b.String()
}

// Plain is a line with its inline marks taken out.
//
// A heading strip has one line for the title and no room to paint spans in
// it, so the marks that would have said "this is code" are read as the
// characters they are. Taking them out is what a label wants; painting them
// is what the panes do.
func Plain(s string) string {
	return strings.NewReplacer("**", "", "`", "").Replace(unlink(s))
}

// firstDelim is whichever inline mark opens first, so that a backtick inside
// a bold run is set as part of it rather than closing something else.
func firstDelim(s string) string {
	bold, code := strings.Index(s, "**"), strings.Index(s, "`")

	switch {
	case bold < 0 && code < 0:
		return ""
	case bold < 0:
		return "`"
	case code < 0 || bold < code:
		return "**"
	default:
		return "`"
	}
}
