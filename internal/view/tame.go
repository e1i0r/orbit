package view

// Text that came from somewhere else, made fit to draw.
//
// An entry's text is whatever the engine printed, and a title is the first
// line of a file somebody — or something — else wrote. Both reach the
// window as they were, and the window hands them to a terminal, which reads
// an escape sequence as an instruction rather than as characters. A phase
// whose output holds ESC[2J clears the reader's screen on every frame that
// draws it; one holding a colour sequence paints the rest of the row, and
// the rows after it, in a colour no theme chose. A bell rings.
//
// Stripping happens here, where a record becomes something a reader is
// shown, rather than in the drawing: the drawing adds escape sequences of
// its own, and a pass over the finished row could not tell which were the
// window's and which came out of an engine.

import "strings"

// tame is one piece of somebody else's text with the control characters
// taken out: escape sequences whole, and every other control character on
// its own. Line breaks stay — the panes split on them — and a tab becomes a
// space, because a terminal jumps to the next stop and nothing measuring
// the row in cells knows where that is.
func tame(s string) string {
	if !strings.ContainsFunc(s, isControl) {
		return s
	}

	var b strings.Builder

	b.Grow(len(s))

	for i := 0; i < len(s); {
		switch c := s[i]; {
		case c == 0x1b:
			i += escapeLen(s[i:])
		case c == '\n':
			b.WriteByte(c)

			i++
		case c == '\t':
			b.WriteByte(' ')

			i++
		case c < 0x20 || c == 0x7f:
			i++
		default:
			b.WriteByte(c)

			i++
		}
	}

	return b.String()
}

// isControl is the cheap look that decides whether a string needs taming at
// all, which almost none of them do.
func isControl(r rune) bool { return r < 0x20 && r != '\n' || r == 0x7f }

// escapeLen is how many bytes the escape sequence starting at s takes.
//
// Three shapes reach a terminal from a program's output: CSI, which is
// ESC [ then the parameters and a letter; OSC, which is ESC ] then a string
// ended by a bell or by ESC \; and the two-byte escapes, which are ESC and
// one more character. Anything that runs off the end of the string is the
// rest of it: a sequence cut in half is still not characters.
func escapeLen(s string) int {
	if len(s) < 2 {
		return len(s)
	}

	switch s[1] {
	case '[':
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}

		return len(s)
	case ']':
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}

			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}

		return len(s)
	}

	return 2
}
