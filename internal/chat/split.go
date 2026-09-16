package chat

// One line of chat, cut into words the way a shell cuts one.
//
// A chat has no shell in front of it, so the quoting nobody thinks about at a
// terminal has to happen here: `/board new -id ACME-12 "the webhook retries
// on 5xx"` is five words and not eight, and a sentence that arrives as eight
// is a task whose title is its first word.

import (
	"fmt"
	"strings"
)

// split cuts a line on spaces, keeping what quotes hold together.
//
// Both kinds of quote, because a phone's keyboard decides which one it feels
// like giving you. What is deliberately not here is escaping: a backslash in
// a chat message is a backslash, and a grammar with one more rule is a
// grammar somebody gets wrong at a bus stop.
func split(line string) ([]string, error) {
	var (
		out   []string
		word  strings.Builder
		quote rune
		open  bool
	)

	flush := func() {
		if word.Len() > 0 {
			out = append(out, word.String())
			word.Reset()
		}
	}

	for _, r := range line {
		switch {
		case open && r == quote:
			// The empty string is a word somebody typed on purpose: it is
			// how a field is set to nothing.
			out = append(out, word.String())
			word.Reset()

			open = false
		case open:
			word.WriteRune(r)
		case r == '"' || r == '\'':
			flush()

			quote, open = r, true
		case r == ' ' || r == '\t' || r == '\n':
			flush()
		default:
			word.WriteRune(r)
		}
	}

	if open {
		// Said rather than guessed at. A quote nobody closed is a message
		// that got cut off or a keyboard that did something clever, and
		// running half of it is worse than asking again.
		return nil, fmt.Errorf("a quote was opened and never closed")
	}

	flush()

	return out, nil
}
