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

// Strong and endStrong mark the one thing an answer emphasises: the name a
// row is filed under.
//
// Control characters and not asterisks, because what is being marked up is
// text the verbs wrote for a terminal and it is full of every punctuation
// mark there is. A marker that could occur in the content is a marker that
// will, and the answer that carries it arrives with half a sentence in bold.
//
// They are this package's own and mean nothing to any service: an adapter
// turns them into whatever it has, and one with nothing takes them out.
const (
	Strong    = "\x02"
	endStrong = "\x03"
	// Raw marks text that is already formatted and must not be escaped.
	Raw    = "\x04"
	endRaw = "\x05"
)

// atMost is how long one answer may be.
//
// Telegram's own ceiling is 4096 characters and every other service is near
// it. The margin is for the line that says what was cut: a message refused
// by the service for being one character over is an answer nobody sees at
// all, which is worse than an answer that ends early and says so.
const atMost = 3800

// Prose is an answer that was already written for a person to read, and is
// marked so that no adapter escapes it.
//
// The supervisor answers in markdown because it was asked to. Everything
// else here is a verb's output — columns padded for a terminal, full of
// hyphens and dots that nobody meant as formatting — and the two want
// opposite treatment. Escaping both deletes the first's formatting; escaping
// neither loses whole messages, which is what shipped for an hour: MarkdownV2
// refuses over an unescaped hyphen, and `unread-cap` has one.
func Prose(said string) string { return Raw + said + endRaw }

// Reply is one verb's answer, dressed for a chat.
func Reply(out verb.Out, p *words.Printer) string {
	said := strings.TrimRight(out.Said, "\n")
	if said == "" {
		return p.T("chat.nothing_said", "done")
	}

	return cut(narrowed(said), p)
}

// phone is how wide a line may be and still be read without dragging it
// sideways.
//
// A guess, and a conservative one: a phone in portrait shows about forty
// monospaced characters at a comfortable size. It is not a measurement of
// anybody's screen — there is no way to have one — it is the width past
// which laying something out in columns stops buying anything, because
// nobody can see two columns at once anyway.
const phone = 40

// narrowed is an answer as a phone can read it.
//
// The verbs write for a terminal, where a column is padded out to line up
// under the one above it. That padding is invisible on a wide screen and is
// the whole problem on a narrow one: a monospaced block does not wrap, so a
// hundred-character line becomes a horizontal scroll bar over three words.
//
// So the padding goes — runs of spaces become two, which still separates the
// columns — and what is still too wide afterwards gives up on being a table
// at all. Plain text wraps, and four wrapped lines a reader can see are
// worth more than one aligned line they have to drag.
func narrowed(said string) string {
	lines := strings.Split(said, "\n")

	widest := 0

	for i, line := range lines {
		lines[i] = squeezed(line)
		if n := utf8.RuneCountInString(lines[i]); n > widest {
			widest = n
		}
	}

	// A table is a table only while it has rows. One line is a sentence
	// whatever it was padded into, and a monospaced sentence is a person
	// being shouted at in a typewriter font.
	if !laidOut(strings.Join(lines, "\n")) {
		return lines[0]
	}

	if widest > phone {
		return spaced(lines)
	}

	return "```\n" + strings.Join(lines, "\n") + "\n```"
}

// spaced is a listing that has given up being a table: a blank line between
// its rows, so that a reader can tell where one ends.
//
// It is the whole of what is left to work with. The rows wrap now — that is
// the point of dropping the block — and twelve wrapped rows with nothing
// between them are one paragraph of twelve settings, which is harder to read
// than the scroll bar was.
//
// A line that begins with a space is not a row. It is the one above it,
// continued, which is how the verbs write a value too long for its column;
// separating it from what it belongs to would be inventing a row that is not
// there.
func spaced(lines []string) string {
	var b strings.Builder

	for i, line := range lines {
		switch {
		case i == 0:
		case strings.HasPrefix(line, " "):
			b.WriteString("\n")
		default:
			b.WriteString("\n\n")
		}

		b.WriteString(named(line))
	}

	return b.String()
}

// named is one row with the name it is filed under set apart.
//
// The first column and nothing else. Wrapped rows run into one another
// otherwise — twelve settings become one paragraph — and what a reader is
// scanning for is the name, which is the only part of a row they knew before
// they opened the message.
//
// The separator is the one squeezed just wrote, so this is reading its own
// output rather than guessing where a column ended.
func named(line string) string {
	at := strings.Index(line, "  ")
	if at <= 0 || strings.HasPrefix(line, " ") {
		return line
	}

	return Strong + line[:at] + endStrong + line[at:]
}

// squeezed is one line with the column padding taken out.
//
// Two spaces and not one, because the columns are still columns: "ACME-3 to
// do" reads as three things and "ACME-3  to do" as two, which is what the
// padding was there to say.
func squeezed(line string) string {
	var (
		b     strings.Builder
		blank int
	)

	for _, r := range strings.TrimRight(line, " \t") {
		if r == ' ' || r == '\t' {
			blank++

			continue
		}

		if blank > 0 {
			b.WriteString(padding(blank, b.Len()))

			blank = 0
		}

		b.WriteRune(r)
	}

	return b.String()
}

// padding is what a run of spaces becomes: itself when it was a single space
// inside a sentence, and two when it was a column being lined up.
//
// A leading run is kept as one space rather than dropped, because an indent
// is how a listing says a line belongs under the one above it.
func padding(blank, written int) string {
	switch {
	case written == 0:
		return " "
	case blank == 1:
		return " "
	}

	return "  "
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
