package learn

// What the model answered about a file, and the check that it did not make
// it up.
//
// Every rule comes back with the sentence in the file it was drawn from, and
// that sentence has to actually be in the file. It is the whole reason this
// can be trusted: a model asked to summarise two years of CONTRIBUTING will
// produce plausible rules nobody ever wrote, and the only way to tell those
// from the real ones without reading it yourself is to make it point at the
// line.
//
// A quote it cannot find is a rule that is dropped, silently and without
// failing the rest. A reader is not owed an argument about what the model
// meant; they are owed the five rules it could prove.

import (
	"strconv"
	"strings"
)

// FromAPaper is what By says when the rule was read out of something the
// project already had written down.
//
// Neither a person nor a model working alone. Somebody wrote the sentence,
// two years ago, about this project — and a model only noticed it was still a
// rule. What a reader needs before they agree is exactly that: not "you said
// this" but "the project says this, here".
const FromAPaper = "the project"

// FromTheHistory is what By says when the rule was read off what the
// repository has actually done rather than off what it says about itself.
//
// Its own value because the two disagree often and the difference is the
// point: a document says what somebody wanted, and the history says what the
// team kept doing.
const FromTheHistory = "the history"

// aFound is one rule the model got out of a file, with where it is.
type aFound struct {
	topic  string
	phrase string
	line   string
}

// aboutThisProject is what the model is shown: the file, and what to do with
// it.
//
// Its own words and nothing around them — not the code, not the record. What
// is being asked is which sentences in this file are still rules, which is a
// reading of the file and not an analysis of the project.
func aboutThisProject(paper, body string, room int, does []string) string {
	var b strings.Builder

	b.WriteString("Here is " + paper + " from a software project:\n\n---\n" + body + "\n---\n\n")

	if len(does) > 0 {
		b.WriteString("And here is what the project's own commit history says it actually does:\n\n")

		for _, one := range does {
			b.WriteString("- " + one + "\n")
		}

		b.WriteString("\nLeave out anything the file asks for that the history above contradicts. " +
			"A sentence nobody has held to for a year is not a rule.\n\n")
	}

	b.WriteString("Find the standing rules in it — the things it expects anybody working " +
		"here to do or not do. Ignore what merely describes the project, explains how to " +
		"install it, or lists its features.\n\n" +
		"Answer with one line per rule, and nothing else:\n\n" +
		"  <topic> | <the rule, in one sentence> | <the exact sentence from the file above " +
		"that it came from, copied word for word>\n\n" +
		"The topic must be one of: " + strings.Join(topics, ", ") + ".\n\n" +
		"The third field must be text copied exactly from the file. A rule you cannot point " +
		"at a line for is one to leave out.\n\n" +
		"Write at most " + strconv.Itoa(room) + " lines, and fewer is better: bring the " +
		"rules somebody here would still say out loud today, not everything the file " +
		"mentions. Writing nothing at all is a fine answer.")

	return b.String()
}

// quoted reads the model's answer, keeping the rules whose quote is really in
// the file and dropping the rest.
func quoted(answer, body string, room int) []aFound {
	var out []aFound

	for _, line := range strings.Split(answer, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "-*• \t")

		topic, rest, found := strings.Cut(line, "|")
		if !found {
			continue
		}

		phrase, quote, found := strings.Cut(rest, "|")
		if !found {
			continue
		}

		one, ok := aRuleFrom(topic, phrase, quote, body)
		if !ok {
			continue
		}

		out = append(out, one)

		if len(out) == room {
			break
		}
	}

	return out
}

// aRuleFrom is one line of the answer read back, and false for one that does
// not hold up.
func aRuleFrom(topic, phrase, quote, body string) (aFound, bool) {
	topic = strings.ToLower(strings.TrimSpace(topic))
	phrase = strings.TrimSpace(phrase)
	quote = strings.Trim(strings.TrimSpace(quote), `"`)

	if !aTopic(topic) || phrase == "" || len(phrase) > aRule || quote == "" {
		return aFound{}, false
	}

	at, there := lineOf(body, quote)
	if !there {
		return aFound{}, false
	}

	return aFound{topic: topic, phrase: phrase, line: strconv.Itoa(at)}, true
}

// lineOf is which line of the file a quote is on, and false when it is on
// none of them.
//
// Compared with the spacing taken out, because a model copying a sentence out
// of a wrapped paragraph rejoins it with one space where the file had a
// newline — and refusing that would throw away real quotes for a difference
// nobody can see.
func lineOf(body, quote string) (int, bool) {
	want := flattened(quote)
	if want == "" {
		return 0, false
	}

	lines := strings.Split(body, "\n")

	// One line at a time first, which is where most quotes are.
	for i, line := range lines {
		if strings.Contains(flattened(line), want) {
			return i + 1, true
		}
	}

	// Then across the wrap: a sentence the file broke over two or three
	// lines is still that sentence.
	//
	// A span never starts on a blank line. Joined, a blank contributes
	// nothing and the match is the same either way — so starting there
	// would answer with the empty line above the sentence, which is not
	// where a reader would say it is.
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		for j := i + 2; j < min(i+4, len(lines)+1); j++ {
			if strings.Contains(flattened(strings.Join(lines[i:j], " ")), want) {
				return i + 1, true
			}
		}
	}

	return 0, false
}

// flattened is text with every run of spacing turned into one space, which is
// the only difference between a quote and the line it was copied from that
// does not matter.
func flattened(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}
