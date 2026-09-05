package ui

// What the operator typed into the supervisor, taken apart.
//
// One line does four things, and which one is decided by how it starts. It is
// four rather than four screens because the whole point is that somebody
// types what they mean while they are thinking it — "/rule coverage stays
// above 90%" in the middle of a conversation, not a form with a scope picker
// and a check field.
//
// The parsing lives here, in the window, and the writing does not: the window
// says what was meant and a port does it. That is the same line every other
// gesture on this screen is drawn on.

import "strings"

// spokenKind is which of the four things a line turned out to be.
type spokenKind int

const (
	// saidMessage is the ordinary case: something said to the supervisor.
	saidMessage spokenKind = iota
	// saidRule is a fact that stops the work, and saidAware one that only
	// warns. They are two words because they are two powers, and the
	// operator chooses which by which one they type.
	saidRule
	saidAware
	// saidNote is a line about one task, which lands in its notes.
	saidNote
	// saidNothing is a gesture nobody finished typing. It is not an empty
	// rule and not an empty message: it is a line to do nothing with.
	saidNothing
)

// The words that start a line, and the flags that widen one.
const (
	ruleWord  = "/rule"
	awareWord = "/aware"
	atWord    = "@"

	generalFlag = "--general"
	langFlag    = "--lang"
)

// spoken is one line, read.
type spoken struct {
	Kind spokenKind
	// Task is the id a note is about.
	Task string
	// Scope is empty for the repository being worked in, "general" for
	// everything, and a language's name for a language. Empty is the
	// default because it is what somebody typing quickly means; the two
	// that reach every repository have to be asked for out loud.
	Scope string
	// Phrase is what was actually said, with the gesture taken off.
	Phrase string
}

// parseSaid reads one line.
//
// A line that starts with something looking like a gesture but is not one —
// "/ruleset", "and/or" — is a message, because it is one. Only the exact
// words, followed by a space or the end of the line, are gestures.
func parseSaid(text string) spoken {
	said := strings.TrimSpace(text)

	switch {
	case word(said, ruleWord):
		return fact(saidRule, rest(said, ruleWord))
	case word(said, awareWord):
		return fact(saidAware, rest(said, awareWord))
	case mentions(said):
		return note(said)
	default:
		return spoken{Kind: saidMessage, Phrase: text}
	}
}

// word is whether a line is this gesture: the word itself, and then either
// nothing or a space. "/ruleset the thing" is not "/rule".
func word(said, gesture string) bool {
	after, is := strings.CutPrefix(said, gesture)

	return is && (after == "" || strings.HasPrefix(after, " "))
}

// mentions is whether a line points at a task: the sign, and then something.
func mentions(said string) bool {
	after, is := strings.CutPrefix(said, atWord)

	return is && after != ""
}

// rest is what follows a gesture.
func rest(said, gesture string) string {
	return strings.TrimSpace(strings.TrimPrefix(said, gesture))
}

// fact reads the scope flags off the front of what was said, and then the
// sentence.
func fact(kind spokenKind, said string) spoken {
	scope := ""

	switch {
	case word(said, generalFlag):
		scope, said = "general", rest(said, generalFlag)
	case word(said, langFlag):
		lang, phrase, _ := strings.Cut(rest(said, langFlag), " ")
		scope, said = lang, strings.TrimSpace(phrase)
	}

	if said == "" || scope == langFlag {
		return spoken{Kind: saidNothing}
	}

	return spoken{Kind: kind, Scope: scope, Phrase: said}
}

// note reads the task a line is about, and what it says about it.
func note(said string) spoken {
	id, phrase, _ := strings.Cut(strings.TrimPrefix(said, atWord), " ")

	phrase = strings.TrimSpace(phrase)
	if id == "" || phrase == "" {
		return spoken{Kind: saidNothing}
	}

	return spoken{Kind: saidNote, Task: id, Phrase: phrase}
}
