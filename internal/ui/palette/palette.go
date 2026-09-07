// Package palette is the ':' line that reaches every command nobody gave a key
// to. One line at the bottom, the commands that match what has been typed
// above it, and a selection among them.
//
// It is a package of its own because it answers one question — "which
// command did the reader mean?" — and because the answer needs four small
// pieces that belong together: the text being typed, the list it filters,
// the selection inside it, and the geometry both the renderer and the
// pointer read. What a chosen command then does is not decided here: the
// screen answers with the one that was chosen and the window runs it.
//
// The input is not bubbles/textinput, for the reason the board's filter is
// not: that component imports a clipboard module this build does not have.
// Like the filter, it is a plain string with a rune appended and a rune
// removed — the day it needs selection is the day the dependency is worth
// arguing for.
package palette

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/words"
)

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}

// Env is what this screen needs of the world: the words it speaks, the keys
// it answers, the room it has, and the commands there are.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	Frame layout.Frame
	// Commands is what can be run, in the order the table lists them: the
	// order a reader learned, which a prefix must not reshuffle.
	Commands []Command
}

// A Command is one row of the list: what the window shows of a command, and
// nothing of what the command does.
//
// The sentences arrive already in the reader's language. Whether a command
// is refused, and why, is the window's reading of it — this screen only says
// so.
type Command struct {
	Name string
	// Args is the usage fragment after the name; empty when there is none.
	Args string
	// About is what the command does, and Because is why it is refused. A
	// refusal replaces the description rather than joining it: the reason is
	// the part a reader acts on, and a line carrying both is a line
	// truncated to lose whichever mattered.
	About   string
	Because string
	Refused bool
	// NeedsArgs says the command refuses when it is given none, and Args
	// says what it wants. The line stays up when one is chosen without
	// them, because what is missing goes on the end of what is already
	// typed — closing it to print the same sentence into a pane with
	// nowhere to type is where this used to end.
	NeedsArgs bool
	// AboutATask keeps a verb off this line. The line is opened on the
	// board, where there is no task for such a verb to be about: choosing
	// one here left the reader holding "pr takes -repo <dir> <id>", a usage
	// string to satisfy by hand for a task on the screen behind it.
	AboutATask bool
}

// Out is what the screen asks the window for.
type Out struct {
	// Said is a sentence for the band.
	Said string
	// Leave is the line coming down.
	Leave bool
	// Run is the command the reader chose, and Line is what they had typed:
	// the arguments after the name are theirs, not this screen's to read.
	Run  string
	Line string
}

// State is the palette while it is up, and nothing while it is down: open
// false means every other field is dead weight the next Open resets anyway.
//
// offset is the first candidate the body is showing, moved only by
// ensureVisible — the list scrolls to keep the selection on screen rather
// than the reader scrolling to find it, because the list is short enough
// that finding it was never the problem.
type State struct {
	open bool

	// typed is what has been typed into the line, as a plain string.
	typed string

	sel    int // an index into candidates()
	offset int // the first candidate the body is showing
}

// Open brings the line up empty. Whatever was typed last time is last
// time's question; a palette that reopened mid-word would answer a command
// the reader is no longer asking.
func Open() State { return State{open: true} }

// OpenWith brings it up with something already on the line, for the
// one caller that knows what the reader is about to type: the menu, whose
// task commands need an id it has no way of supplying.
func OpenWith(typed string) State { return State{open: true, typed: typed} }

// Up is whether the line is showing, which is what decides who owns the
// keyboard and what the body draws.
func (s State) Up() bool { return s.open }

// Typed is what is on the line, which the window runs when a command is
// chosen: the arguments after the name are the reader's.
func (s State) Typed() string { return s.typed }

// firstWord is the name part of what has been typed, through the same split
// the runner uses on the same line: what it calls the command here is what
// gets run there, and an empty line is a prefix everything matches.
func firstWord(typed string) string {
	fields := strings.Fields(typed)
	if len(fields) == 0 {
		return ""
	}

	return fields[0]
}

func matchesSettingsAlias(prefix string) bool {
	for _, alias := range []string{"configuraciones", "config", "set", "ajustes"} {
		if strings.HasPrefix(alias, prefix) {
			return true
		}
	}

	return false
}

// candidates is every command whose name starts with what has been typed,
// in the order the table handed them over — the table's order is the one a
// reader learned, and reshuffling it under a prefix would move rows between
// two keystrokes.
//
// Only the first word is the prefix. What follows it is the command's own
// arguments, and matching against the whole line meant that the moment a
// space was typed nothing matched: `cancel PAY-11` had no selection, so ⏎
// on it did nothing at all and the line sat there looking broken.
//
// The match ignores case, as the board's filter does: a reader who types a
// capital because a sentence started with one still means the command.
func (s State) candidates(cmds []Command) []Command {
	prefix := strings.ToLower(firstWord(s.typed))

	var out []Command

	for _, c := range cmds {
		// A verb about one task is not on this line. The line is opened on
		// the board, where there is no task for such a verb to be about:
		// choosing one here left the reader holding "pr takes -repo <dir>
		// <id>", a usage string to satisfy by hand for a task that is on
		// the screen behind it. The task's own menu is where they belong,
		// and it is the rule the board's menu already applies.
		if c.AboutATask {
			continue
		}

		name := strings.ToLower(c.Name)
		if strings.HasPrefix(name, prefix) {
			out = append(out, c)
		} else if c.Name == "settings" && matchesSettingsAlias(prefix) {
			out = append(out, c)
		}
	}

	return out
}

// selected is the candidate the selection is on, or nil when there is
// nothing to be on — an empty list, or an index that lost its row to a
// shorter list after another rune landed.
func (s State) selected(cmds []Command) (Command, bool) {
	all := s.candidates(cmds)
	if s.sel < 0 || s.sel >= len(all) {
		return Command{}, false
	}

	return all[s.sel], true
}
