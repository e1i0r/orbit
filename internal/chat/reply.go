package chat

// What a verb's answer looks like in a chat.
//
// Every verb writes one sentence for a reader — Out.Said — and it is written
// for a terminal: columns that line up, a listing under a heading. A chat is
// not a terminal. The text is not monospaced, there is a ceiling on how long
// one message can be, and nobody scrolls a wall on a phone.
//
// So this is the one place that decides how an answer is dressed, and it
// decides on one rule: **what was laid out stays laid out, and what was a
// sentence stays a sentence.** A listing put into a chat as prose is columns
// collapsed into a paragraph, and a sentence put into a code block is a
// person being shouted at in monospace.

import (
	"strings"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// atMost is how long one answer may be.
//
// Telegram's own ceiling is 4096 characters and every other service is near
// it. The margin is for the line that says what was cut: a message refused
// by the service for being one character over is an answer nobody sees at
// all, which is worse than an answer that ends early and says so.
const atMost = 3800

// Reply is one verb's answer, dressed for a chat.
func Reply(out verb.Out, p *words.Printer) string {
	said := strings.TrimRight(out.Said, "\n")
	if said == "" {
		return p.T("chat.nothing_said", "done")
	}

	if laidOut(said) {
		said = "```\n" + said + "\n```"
	}

	return cut(said, p)
}

// laidOut says whether the answer was written to be read in columns.
//
// Two lines or more is a listing: every verb that answers with one sentence
// answers with one line, and every one that answers with a table answers
// with several. That is a rule about how the verbs are written rather than a
// guess about their text, which is why it holds — and a one-line answer that
// happens to be a table loses nothing by being plain.
func laidOut(said string) bool {
	return strings.Contains(said, "\n")
}

// cut shortens an answer that will not fit and says that it did.
//
// Announced rather than silent, for the reason the record's own truncation
// is announced: a reader who cannot tell a short answer from a cut one
// believes the short one.
func cut(said string, p *words.Printer) string {
	if len(said) <= atMost {
		return said
	}

	end := "\n\n" + p.T("chat.cut", "…too long for one message; ask for it at a terminal")
	kept := whole(said, atMost)

	// Inside the fence rather than after it: a code block that opened and
	// never closed swallows whatever the service draws next.
	if strings.HasPrefix(said, "```") {
		return kept + "\n```" + end
	}

	return kept + end
}

// whole is the first n bytes of a string without a rune cut in half.
//
// A severed rune reaches the service as a replacement character, or as a
// message it refuses outright for not being valid UTF-8 — and an answer
// nobody sees is worse than one that ends a character early.
func whole(said string, n int) string {
	for n > 0 && !utf8.RuneStart(said[n]) {
		n--
	}

	return said[:n]
}
