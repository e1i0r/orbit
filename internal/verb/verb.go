// Package verb is every action Orbit can be asked for, declared once.
//
// Orbit has four ways in — the command line, the window, the browser and the
// MCP server — and until this existed each of them carried its own list of
// what could be asked for. They drifted, quietly and in every direction: the
// MCP server could start a stopped task and the browser could not, the
// browser could let a phase past its gate and the MCP server could not, and
// the command line could open a pull request where neither of the others
// could. Nobody decided any of that. It is what happens when the same idea
// is written down in four places.
//
// So the idea is written down here, once, and the four are ports over it.
// What a verb is called, what it needs beyond a task, whether it spends
// money and whether it leaves this machine are facts about the verb, not
// about the way in — and a way in that does not offer one of them has to say
// why, in verb_test.go, rather than simply not have it.
//
// This package declares and does not do. The doing is internal/task's and
// stays there: what is centralised is the vocabulary, because that is what
// was drifting.
package verb

import "github.com/e1i0r/orbit/internal/words"

// A Verb is one thing Orbit can be asked to do.
type Verb struct {
	// Name is what it is called everywhere. The surfaces spell it in their
	// own idiom — a command, a route, a tool — but they spell the same
	// word, so a reader who runs two of them learns one vocabulary.
	Name string
	// Under is the verb this one belongs to, and empty for the ones that
	// belong to nobody.
	//
	// Some things Orbit can be asked for come in families: a tray is listed,
	// and a row of it is kept or dropped. Flat names make those read as
	// three unrelated words that happen to share a prefix — and they sort
	// apart in the one place somebody goes looking for them, which is the
	// list of what can be asked for.
	//
	// A family is one level deep and stays that way. Two is a tree, a tree
	// needs a way to be walked, and nothing Orbit does has been hard to say
	// in two words.
	Under string
	// Was is what this verb answered to before it joined a family, and
	// empty for the ones that have always had the name they have.
	//
	// A rename that breaks every script somebody wrote is not a rename; it
	// is a removal with something new standing next to it. So the old name
	// keeps working, everywhere, and the first thing it says is what to
	// type instead. Declared here once so that the four ways in cannot
	// disagree about which old name meant what.
	Was string
	// About is the sentence a reader is shown, through internal/words so
	// that it is the same sentence in both languages.
	About func(*words.Printer) string
	// Takes is what it needs beyond the task it is about.
	Takes []Field
	// OnTask is whether it is about one task. The rest are about the board
	// or the machine: writing a task down, saying something to the
	// supervisor, telling Orbit something true about the code.
	OnTask bool
	// Spends says asking for this runs an engine, which costs money. Every
	// surface has to say so before it asks.
	Spends bool
	// Reads says it changes nothing. A reading is asked for the same way
	// an action is — the board, one task, a flow, what a change reaches —
	// and it drifted between the four surfaces for the same reason: three
	// lists of questions, each grown on its own.
	//
	// It is a field rather than a second list because the surfaces treat
	// them differently and have to be able to tell: a reading is a GET, an
	// action is a POST behind a guard, and a model may be given every
	// reading while being trusted with only some of the actions.
	Reads bool
	// Outward says it leaves this machine. Everything else Orbit does can
	// be undone by asking again; a pull request is on somebody's GitHub the
	// moment it opens, and a merge is in the branch other people work from.
	Outward bool
}

// Path is the whole of what this verb is called: its own word for the ones
// that belong to nobody, and both words for a child.
//
// It is the identity every surface keys on. Name alone is not: two families
// may each have a keep, and a map of those would hold one of them.
func (v Verb) Path() string {
	if v.Under == "" {
		return v.Name
	}

	return v.Under + " " + v.Name
}

// Children is the verbs that belong to this one, in the order they were
// declared.
func (v Verb) Children() []Verb {
	var out []Verb

	for _, other := range Every() {
		if other.Under == v.Name {
			out = append(out, other)
		}
	}

	return out
}

// Field is one thing a verb needs typed into it.
type Field struct {
	Name  string
	About func(*words.Printer) string
	Kind  Kind
	// Needed says the verb cannot be asked for without it. A field that is
	// not needed has a working zero: no reason given, no restart, no flow
	// named and so the one the settings chose.
	Needed bool
}

// Kind is what a field holds. Three, because three is what the verbs
// actually take: a sentence, a name out of a list Orbit already knows, and
// a yes or no.
type Kind int

const (
	// Words is prose a person writes: a note, a directive, a reason.
	Words Kind = iota
	// Named is one of something Orbit can list — a flow, an engine, a
	// repository — so a surface can offer the list rather than a blank box.
	Named
	// YesOrNo is a switch.
	YesOrNo
)
