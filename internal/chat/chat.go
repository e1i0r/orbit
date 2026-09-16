package chat

// Orbit asked for from a chat.
//
// The fifth way in. internal/verb declares every action Orbit can be asked
// for — what it is called, what it takes, what it says about itself — and
// the four that came before this one are all built from that one table: the
// command line, the window, the browser, the tool a model calls. This is the
// same table read again, for somebody holding a phone.
//
// It is a way in and not a product. `/task start ACME-3 -engine codex` is
// parsed into the same In the terminal builds and handed to the same body,
// so a verb behaves the same however it was asked — including the parts
// nobody thinks about until they differ: who the record says did it, what it
// refuses, and what it answers.
//
// Nothing here talks to a network. What arrives is a line of text and what
// leaves is a line of text; which chat carried them is the adapter's, and
// there is one per service.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// Asked is one message read as a request: which verb, and what it was asked
// with.
type Asked struct {
	Verb string
	In   verb.In
}

// door is what the record says about where a verb was asked for. The other
// four say "the command line", "the cockpit", "the browser", "a tool call";
// this is the fifth, and a reader six months later can tell which of them
// started a run.
const door = "a chat"

// Read turns one message into the verb it asks for.
//
// A message that does not begin with a slash is not a command and is not an
// error either: a chat is a place people also talk. The caller decides what
// to do with prose — today, nothing.
func Read(text string, p *words.Printer) (Asked, bool, error) {
	said := strings.TrimSpace(text)
	if !strings.HasPrefix(said, "/") {
		return Asked{}, false, nil
	}

	line, err := split(strings.TrimPrefix(said, "/"))
	if err != nil {
		return Asked{}, true, err
	}

	if len(line) == 0 {
		return Asked{}, true, fmt.Errorf("%s", p.T("chat.no_verb",
			"a slash with nothing after it asks for nothing"))
	}

	v, rest, found := verbOf(line)
	if !found {
		return Asked{}, true, fmt.Errorf("%s", p.T("chat.no_such_verb",
			"{name} is not something Orbit can be asked for; /help lists what is",
			words.Arg{Name: "name", Value: line[0]}))
	}

	in := verb.In{Args: map[string]string{}, By: "operator", Door: door}

	rest = flags(v, in.Args, rest)

	// The task comes first among what is left, the way it does on a command
	// line: `/task start ACME-3` and `orbit task start ACME-3` are the same
	// words in the same order, which is the whole point of one vocabulary.
	if v.OnTask {
		if len(rest) == 0 {
			return Asked{}, true, fmt.Errorf("%s", p.T("chat.needs_task",
				"{verb} is about one task, and none was named",
				words.Arg{Name: "verb", Value: v.Path()}))
		}

		in.Task, rest = rest[0], rest[1:]
	}

	if over := verb.Fill(v, in.Args, rest); len(over) > 0 {
		return Asked{}, true, fmt.Errorf("%s", p.T("verb.takes_no_more",
			"{verb} takes nothing after {extra}",
			words.Arg{Name: "verb", Value: v.Path()},
			words.Arg{Name: "extra", Value: strings.Join(over, " ")}))
	}

	return Asked{Verb: v.Path(), In: in}, true, nil
}

// verbOf is the verb a line names: both of its words when it belongs to a
// family, one when it does not.
//
// Two first, because a family's child is both words — `task start`, never
// `start` — and a parent that swallowed the line would run the listing when
// somebody asked for the child.
func verbOf(line []string) (verb.Verb, []string, bool) {
	if len(line) > 1 {
		if v, ok := verb.One(line[0] + " " + line[1]); ok {
			return v, line[2:], true
		}
	}

	if v, ok := verb.One(line[0]); ok {
		return v, line[1:], true
	}

	return verb.Verb{}, nil, false
}

// flags reads the `-name value` pairs off the front and answers what is
// left.
//
// Only what was actually typed reaches the verb, for the reason the command
// line learned the hard way: a declared flag is not an answer, and a verb
// that cannot tell an empty value from an absent one throws away the thing
// it was asked to change.
func flags(v verb.Verb, args map[string]string, line []string) []string {
	var rest []string

	for i := 0; i < len(line); i++ {
		name := strings.TrimPrefix(strings.TrimPrefix(line[i], "-"), "-")

		field, declared := fieldOf(v, name)
		if !strings.HasPrefix(line[i], "-") || !declared {
			rest = append(rest, line[i])

			continue
		}

		next := ""
		if i+1 < len(line) {
			next = line[i+1]
		}

		// A switch is on unless the word after it is an answer to one.
		// `-restart use the retry helper` is a restart and a directive, and
		// reading the "use" as its value takes the first word of what the
		// person actually said.
		if field.Kind == verb.YesOrNo && !isYesOrNo(next) {
			args[name] = "true"

			continue
		}

		if next == "" || strings.HasPrefix(next, "-") {
			args[name] = "true"

			continue
		}

		args[name], i = next, i+1
	}

	return rest
}

// fieldOf is the field a verb declares under that name.
func fieldOf(v verb.Verb, name string) (verb.Field, bool) {
	for _, f := range v.Takes {
		if f.Name == name {
			return f, true
		}
	}

	return verb.Field{}, false
}

// isYesOrNo says whether a word is an answer to a switch.
//
// The same words In.Yes reads, plus their opposites — a caller who typed
// `-run off` meant the switch and not a positional, whatever Yes makes of
// the word afterwards.
func isYesOrNo(word string) bool {
	switch strings.ToLower(word) {
	case "true", "yes", "1", "on", "false", "no", "0", "off":
		return true
	}

	return false
}
