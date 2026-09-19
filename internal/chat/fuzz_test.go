package chat

// A message from a channel, read as the verb it asks for.
//
// This is the least trusted door Orbit has. A command line is typed by
// whoever is at the machine and a tool call comes from a model the reader
// pointed at their own record — a chat message arrives from a room, and the
// room may hold somebody nobody vouched for. So what this reading promises
// has to hold for any bytes at all: it answers a verb Orbit knows, or it
// refuses, and it never falls over in between.

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// FuzzAMessageIsAVerbOrItIsNot.
func FuzzAMessageIsAVerbOrItIsNot(f *testing.F) {
	for _, seed := range []string{
		"", "hello", "/", "/task start ACME-1", "/board", "/task",
		`/task note ACME-1 "two words"`, "/task note ACME-1 'two words'",
		"/nosuchverb", "/task start --flow quick", "/  ", "//",
		`/task note ACME-1 "unclosed`, "/task start\nACME-1",
		"/task start ACME-1 extra words nobody asked for",
	} {
		f.Add(seed)
	}

	p := words.For("")

	f.Fuzz(func(t *testing.T, text string) {
		asked, isCommand, err := Read(text, p)

		// Prose is not a command and not an error either: a chat is a place
		// people also talk, and a reading that refused every sentence would
		// answer somebody's "good morning" with a complaint.
		if !strings.HasPrefix(strings.TrimSpace(text), "/") {
			if isCommand || err != nil {
				t.Errorf("%q is not a command and read as one: %v, %v", text, isCommand, err)
			}

			return
		}

		if err != nil {
			// A refusal says something, because it is printed back into the
			// room the message came from.
			if err.Error() == "" {
				t.Errorf("%q was refused with nothing said", text)
			}

			return
		}

		if !isCommand {
			t.Errorf("%q opens with a slash and did not read as a command", text)
		}

		// Whatever came back is a verb this build actually has. A name that
		// reached Run without one would be answered "declared and not
		// done", which is a sentence about Orbit and not about the message.
		if _, found := verb.One(asked.Verb); !found {
			t.Errorf("%q asked for %q, which no verb answers to", text, asked.Verb)
		}

		// And what it carries is text: it goes into the record, which is a
		// column, and into a reply printed back into the room.
		for name, value := range asked.In.Args {
			if !utf8.ValidString(name) || !utf8.ValidString(value) {
				t.Errorf("%q carried an argument that is not text: %q = %q", text, name, value)
			}
		}
	})
}

// FuzzSplitKeepsWhatQuotesHoldTogether.
//
// A phone's keyboard decides which kind of quote it feels like giving you,
// and there is deliberately no escaping: a backslash in a chat message is a
// backslash. What that leaves is a small grammar that has to hold for
// anything typed at a bus stop.
func FuzzSplitKeepsWhatQuotesHoldTogether(f *testing.F) {
	for _, seed := range []string{
		"", "one two", `"one two"`, "'one two'", `one "two three" four`,
		`"`, "'", `""`, "''", `"unclosed`, `mixed "quotes'`, "  spaced  out  ",
		"\ttabbed\tout", "acentos y ñ", "🛰 emoji",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		out, err := split(line)
		if err != nil {
			if out != nil {
				t.Errorf("split(%q) refused and still answered %v", line, out)
			}

			return
		}

		for i, word := range out {
			// A word may be empty on purpose — that is how a field is set
			// to nothing — but it may never hold the space it was cut on.
			if strings.ContainsAny(word, " \t") && !strings.ContainsAny(line, `"'`) {
				t.Errorf("split(%q) answered word %d as %q, with the space it was cut on", line, i, word)
			}
		}

		// Nothing is invented: every character of every word came out of
		// the line it was cut from.
		if utf8.ValidString(line) {
			for i, word := range out {
				if word != "" && !strings.Contains(line, word) {
					t.Errorf("split(%q) answered word %d as %q, which is not in it", line, i, word)
				}
			}
		}
	})
}
