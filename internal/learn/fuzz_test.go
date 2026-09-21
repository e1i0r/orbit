package learn

// What Orbit reads that somebody else wrote: a sentence a person typed and
// an answer a model gave.
//
// Both arrive as text with nothing behind them. A person types whatever they
// type, in either of two languages, and a model answers in a shape it was
// asked for and is under no obligation to keep. Every reading here has to
// hold against both — a tray that fills with garbage is a tray nobody opens,
// and a reading that falls over takes the run with it.

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzARuleIsSomethingSaidAboutTheFuture, read against anything a person can
// type into a terminal.
func FuzzARuleIsSomethingSaidAboutTheFuture(f *testing.F) {
	for _, seed := range []string{
		"", "always run the tests first", "should I always run the tests?",
		"never push on a Friday", "siempre corré los tests", "¿siempre?",
		"please always wrap errors", strings.Repeat("a", aRule+1),
		"\n\n", "   ", "ALWAYS ", "always", "no quiero que",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, said string) {
		if !aboutTheFuture(said) {
			return
		}

		// A sentence past the length of a rule is a paragraph — a briefing,
		// a question, an explanation — and reading one as a rule fills the
		// tray with everything anybody ever said.
		if len(strings.TrimSpace(said)) > aRule {
			t.Errorf("%q is %d characters and read as a rule", said, len(strings.TrimSpace(said)))
		}

		// A question is not a rule however it opens: answering it by
		// writing down what was asked is the wrong way round.
		if trimmed := strings.TrimSpace(said); strings.HasSuffix(trimmed, "?") {
			t.Errorf("%q is a question and read as a rule", said)
		}
	})
}

// FuzzRulesInAModelsAnswer.
//
// A model that explains itself first, bullets its answer or invents a topic
// has not broken anything: what it wrote that cannot be read is simply not
// offered. What must not happen is the other way round — something it wrote
// becoming a rule nobody can trace, or one habit filling the tray.
func FuzzRulesInAModelsAnswer(f *testing.F) {
	for _, seed := range []string{
		"", "testing | anything that parses input gets fuzz tests",
		"Sure! Here is what I found:\n\n- testing | a thing\n  vibes | another",
		"testing | " + strings.Repeat("a", aRule+1),
		"testing |", "| a phrase with no topic", "no separator at all",
		strings.Repeat("testing | one more thing\n", 10),
		"testing | a thing | and a third field",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, answer string) {
		got := rulesIn(answer)

		if len(got) > atMost {
			t.Errorf("one habit produced %d rules, and the cap is %d", len(got), atMost)
		}

		for i, one := range got {
			if !aTopic(one.topic) {
				t.Errorf("rule %d is filed under %q, which nobody wrote down", i, one.topic)
			}

			if one.phrase == "" {
				t.Errorf("rule %d says nothing", i)
			}

			if len(one.phrase) > aRule {
				t.Errorf("rule %d is %d characters, past the %d a rule may be", i, len(one.phrase), aRule)
			}

			// A phrase is as good as the answer it came out of: a model
			// whose output was cut through a multi-byte character hands
			// over bytes that are not text, and nothing here mends them.
			if utf8.ValidString(answer) && !utf8.ValidString(one.phrase) {
				t.Errorf("rule %d is not text: %q", i, one.phrase)
			}
		}
	})
}

// FuzzARuleFromAFilePointsAtALine.
//
// This is the whole reason a reading of somebody's CONTRIBUTING can be
// trusted. A model asked to summarise two years of it will produce plausible
// rules nobody ever wrote, and the only way to tell those from the real ones
// without reading the file yourself is to make it point at the line — so a
// rule that comes back must be one whose quote is really in the file, at the
// line it names.
func FuzzARuleFromAFilePointsAtALine(f *testing.F) {
	f.Add("testing | a change comes with a test | Every change ships with a test.",
		"# Contributing\n\nEvery change ships with a test.\nNever push on a Friday.\n", 5)
	f.Add("testing | invented | A sentence nobody wrote.", "# Contributing\n", 5)
	f.Add("", "", 0)
	f.Add("a|b|c", "c", 1)

	f.Fuzz(func(t *testing.T, answer, body string, room int) {
		if room < 0 || room > 64 {
			t.Skip()
		}

		got := quoted(answer, body, room)

		if len(got) > room {
			t.Errorf("a reading with room for %d came back with %d", room, len(got))
		}

		lines := strings.Split(body, "\n")

		for i, one := range got {
			at, err := strconv.Atoi(one.line)
			if err != nil {
				t.Errorf("rule %d points at %q, which is not a line number", i, one.line)
				continue
			}

			if at < 1 || at > len(lines) {
				t.Errorf("rule %d points at line %d of a file with %d lines", i, at, len(lines))
			}

			if !aTopic(one.topic) {
				t.Errorf("rule %d is filed under %q, which nobody wrote down", i, one.topic)
			}

			if one.phrase == "" || len(one.phrase) > aRule {
				t.Errorf("rule %d says %q, which is not a rule's length", i, one.phrase)
			}
		}
	})
}

// FuzzTwoWordsAreOneWord is the comparison every habit is grouped by.
//
// It walks two words side by side, which is one index away from reading past
// the end of the shorter — and what it answers is what a reader is shown
// beside the sentences, so it must be a part of both words and not something
// assembled from them.
func FuzzTwoWordsAreOneWord(f *testing.F) {
	f.Add("tests", "testing")
	f.Add("test", "testing")
	f.Add("", "")
	f.Add("a", "")
	f.Add("subí", "subilo")
	f.Add("🛰", "🛰️")

	f.Fuzz(func(t *testing.T, a, b string) {
		// Words reach this from meaningful(), which splits text. Bytes that
		// are not text become one replacement rune each on the way in, so
		// what comes back is a prefix of the runes and not of the bytes —
		// which is a true answer about a word nobody typed.
		if !utf8.ValidString(a) || !utf8.ValidString(b) {
			sameWord(a, b)

			return
		}

		stem, same := sameWord(a, b)

		if !same {
			if stem != "" {
				t.Errorf("sameWord(%q, %q) said no and answered %q", a, b, stem)
			}

			return
		}

		if !strings.HasPrefix(a, stem) || !strings.HasPrefix(b, stem) {
			t.Errorf("sameWord(%q, %q) = %q, which is not the front of both", a, b, stem)
		}

		if n := utf8.RuneCountInString(stem); n < shortWord {
			t.Errorf("sameWord(%q, %q) = %q, %d runes, and %d is the least worth comparing by",
				a, b, stem, n, shortWord)
		}
	})
}
