package chat

// What a chat offers, and the few things it does not.
//
// Everything, by construction: the list is verb.Every() and nothing here
// writes a second one. A verb added to the declaration is a verb somebody
// can ask for from a phone the same day, which is the whole reason that
// declaration exists.
//
// The exceptions are written down rather than left to be noticed. Each is a
// verb a chat cannot carry — not one nobody got round to — and internal/arch
// reads this list so that a verb quietly dropped here fails there.

import (
	"sort"
	"strings"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// cannot is a verb a chat does not offer, and why.
var cannot = map[string]string{
	"task take": "this hands a terminal to an engine, and a chat has no terminal to hand over",
	"task diff": "a diff is read in columns against a wide window; a phone would get " +
		"the first file and a scroll bar",
	"task tree":   "the same: a tree of a repository is a shape, not a paragraph",
	"task impact": "the same, and it is the slowest reading there is",
	"export":      "it writes the record into a directory the reader names, and a chat has no filesystem",
	"task compare": "it runs the flow's checks on both sides of a change, which takes minutes " +
		"and answers in columns",
}

// Offers is every verb a chat can be asked for, in the order /help lists
// them.
func Offers() []verb.Verb {
	var out []verb.Verb

	for _, v := range verb.Every() {
		if cannot[v.Path()] == "" {
			out = append(out, v)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Path() < out[j].Path() })

	return out
}

// Menu is what can be asked for, in the shape a service's own menu wants.
//
// A service will not take a space in a command, so a family's two words are
// joined with an underscore — and Read accepts that spelling back, so the
// menu a reader taps and the line a reader types are the same thing.
//
// The description is the verb's own sentence about itself, which is the same
// one `orbit <verb> -h` prints. One declaration, five doors.
func Menu(p *words.Printer) []Command {
	var out []Command

	for _, v := range Offers() {
		out = append(out, Command{
			Name:  strings.ReplaceAll(v.Path(), " ", "_"),
			About: v.About(p),
		})
	}

	return out
}

// Help is the list of what can be asked for, as one message.
//
// Built from the declaration, so a verb added to Orbit appears here without
// anybody remembering to come and add it — and it says what each one takes,
// because a command list nobody can act on is a command list.
func Help(p *words.Printer) string {
	var b strings.Builder

	b.WriteString(p.T("chat.help_title", "What you can ask for:") + "\n\n```\n")

	for _, v := range Offers() {
		line := "/" + v.Path()
		if shape := takesOf(v, p); shape != "" {
			line += " " + shape
		}

		b.WriteString(line + "\n")
	}

	return b.String() + "```"
}

// takesOf is the shape of one verb's line: the task it is about, then the
// fields it must have, then the sentence that takes the rest.
func takesOf(v verb.Verb, p *words.Printer) string {
	var parts []string

	if v.OnTask {
		parts = append(parts, "<id>")
	}

	for _, f := range v.Takes {
		switch {
		case f.Kind == verb.Words:
			continue
		case f.Needed:
			parts = append(parts, "<"+f.Name+">")
		default:
			parts = append(parts, "[-"+f.Name+"]")
		}
	}

	// The field of words is last because it takes everything left, which is
	// what makes `/task note ACME-3 it needs a test` read as a sentence
	// rather than as five arguments.
	for _, f := range v.Takes {
		if f.Kind == verb.Words {
			parts = append(parts, "<"+f.Name+">")

			break
		}
	}

	return strings.Join(parts, " ")
}
