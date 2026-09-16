package known

// The screen that lists what Orbit knows.
//
// It is obligatory rather than nice: a system that puts sentences into the
// model's context and refuses work over them, with no way to see which
// sentences, is a system nobody leaves switched on. If it cannot be seen it
// is not trusted, and what is not trusted gets turned off wholesale.
//
// A screen of its own and not a section of the repositories list. What Orbit
// knows is wider than the checkouts — the general facts and the ones about a
// language belong to no repository at all — and hanging the whole of it off
// one of them leaves those with nowhere to go.
//
// It is entered through this file: State is what the screen is, Env is what
// the window lends it, and Out is the sentence or the exit it asks for.

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/typing"
	"github.com/e1i0r/orbit/internal/words"
)

// Env is what this screen needs of the world: the words it speaks, the keys
// it answers, and the three doors of the store. It is handed no board — the
// one thing it needs from one, which repository is being worked in, arrives
// as a path.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	// Frame is the room the window lends it. The list scrolls, so how many
	// rows there are to scroll into is part of every gesture and not only
	// of drawing: a cursor moved is a cursor that has to be brought back
	// on screen, and that cannot be worked out at draw time from a value
	// the key press already threw away.
	Frame layout.Frame
	// All is everything Orbit has learned, across every repository on the
	// board. Nil in a window built without a store, and then the screen
	// says there is nothing rather than pretending it read.
	All func() []knowledge.Fact
	// Replace writes one fact over another. The old one travels with it
	// because a fact's file is named after its sentence: writing alone
	// would leave the old copy behind, still told and still refusing work.
	Replace func(was, now knowledge.Fact) error
	// Turn switches one fact off, or on again.
	Turn func(f knowledge.Fact) error
	// Waiting is what you said to the supervisor that read as a rule and
	// nobody has answered yet. Nil in a window built without a store, and
	// then the tray is simply not there.
	Waiting func() []Said
	// Keep writes one of them down as a fact of yours. The words and the
	// place are handed over rather than read back out of the tray, because
	// correcting and placing are how most of these are accepted.
	Keep func(at time.Time, phrase, check, where string) error
	// Drop says it was not a rule. The sentence stays in the thread where it
	// was said, which is where it belonged all along.
	Drop func(at time.Time) error
	// Story is what one rule has put you through, in sentences: when you
	// kept it, where it stopped the work and how often you got past it,
	// and what you paused it for.
	//
	// Sentences and not numbers, because a rule that works perfectly never
	// stops anything — so a count of nothing means two opposite things and
	// no number tells them apart. Nil in a window built without a store,
	// and then the review says there is nothing to show rather than
	// pretending it read.
	Story func(f knowledge.Fact) []string
	// Places are the folders of one checkout, offered when a rule is being
	// filed so that choosing where it applies is picking rather than
	// remembering which folders the project has and spelling one right.
	Places func(repo string) []string
	// Commands are what a checkout already runs on itself — its Makefile's
	// targets — offered as the check of a rule that refuses work. A rule
	// that stops the work is worth nothing without one, and a command
	// somebody half-remembers is worse than none.
	Commands func(repo string) []string
	// Repo is the one repository on the board, and empty when there is more
	// than one. A fact written here is about it; choosing one of several
	// for somebody is how a rule ends up on the wrong project.
	Repo string
}

// A Said is one sentence somebody said to the supervisor that read as a
// rule, waiting to be told whether it was one.
//
// The screen's own shape, and not the shape of the package that holds the
// tray: what reaches the window is data, through a port, here as everywhere
// else.
type Said struct {
	// At is when it was said, and it is the sentence's name: the thread is
	// append-only and no two turns share an instant.
	At   time.Time
	Text string
	// From is where it was said: the task it was typed at, or the way in it
	// came through when it was about no task. It is what a reader needs
	// before they can agree with anything — the same words said while
	// correcting one run and said to the supervisor are the same rule, and
	// which it was is how somebody decides whether it was meant that widely.
	From string
	// Where is the folder the work was in when it was said, relative to the
	// checkout it came out of, and empty when it came out of no one folder.
	// It is what the editor's place line opens with: the commonest correction
	// is a path, and the commonest path is this one.
	Where string
}

// Out is what the screen asks the window for.
type Out struct {
	// Said is a sentence for the band.
	Said string
	// Leave is the reader closing the screen.
	Leave bool
}

// said is one sentence and nothing else.
func said(text string) Out { return Out{Said: text} }

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}

// State is the screen: which fact the cursor is on, and the facts as
// they were last read.
//
// The facts are held rather than asked for while drawing, for the reason the
// supervisor's side holds its own: the port reads two directories off disk,
// and a frame is drawn ten times a second.
type State struct {
	sel int
	// deep is the first line on show of a screen opened over the list: the
	// form, and a rule's own. It is apart from offset because coming back
	// from one of them should find the list where it was left.
	deep int
	// offset is the first line of the list on show. It is lines and not
	// rules: a rule is as tall as its sentence wraps, and a list scrolled
	// by rules jumps by however tall the next one happens to be.
	offset int
	// waiting is the tray, and facts is what Orbit already knows. The cursor
	// walks the two of them in the order they are drawn, which is why almost
	// nothing here indexes either one directly.
	waiting []Said
	facts   []knowledge.Fact
	// read is whether the port has been asked at all. It is not len(facts):
	// a workspace where nothing has been written down answers an empty
	// list, and without this the header's chip would ask again on every
	// board refresh — two directories off disk, twice a second, to be told
	// zero each time.
	read bool

	// editing is the fact under the cursor being corrected in place, and in
	// holds the three things about it that are text: what it says, the
	// command that decides whether it can stop the work, and the folder or
	// file it is about.
	//
	// Three fields and not the whole record. What is absent is the source —
	// where a fact came from is what makes it traceable, and it is not
	// something anybody retypes. How wide it is stays on ←/→, because
	// everywhere and one checkout are not paths and cannot be typed as one.
	editing bool
	// pausing says the line being typed is a reason rather than a
	// correction. One line, two gestures: what is typed is a sentence
	// either way, and this is which sentence it is.
	pausing bool
	// fresh says the form is writing a rule nobody had written before,
	// which is the one case its heading cannot work out from the fields: a
	// new rule and a rule whose sentence was emptied look the same.
	fresh bool
	// reading is the rule under the cursor opened on its own screen, with
	// everything about it and everything it has put you through. It is the
	// one place a rule's fate is decided: nothing here is a decision taken
	// in a hurry, because it is opened on purpose.
	reading bool
	field   int
	in      [factFields]typing.Field
}

// The three fields of a fact that are typed into.
const (
	factPhrase = iota
	factCheck
	factWhere
	factFields
)

// Open is the screen coming up, with the store read.
func Open(e Env) State {
	return State{}.Sync(e)
}

// Count is how many facts there are, for the chip in the header. It reads
// what was last loaded and never the store: the header is drawn on every
// frame, and reading walks every repository on the board.
func (s State) Count() int { return len(s.facts) }

// Unanswered is how many sentences are in the tray, for the same chip. A
// tray nobody is told about is a tray nobody opens, and this screen is not
// one somebody passes by accident.
func (s State) Unanswered() int { return len(s.waiting) }

// SyncOnce reads the store if it never has been, and does nothing after
// that. The header's chip is what wants it, on the first board that arrives:
// reading walks every repository, and the count only changes when a fact is
// written — which is where the store is read again.
func (s State) SyncOnce(e Env) State {
	if s.read {
		return s
	}

	return s.Sync(e)
}

// Sync reads the facts again, keeping the cursor on something real.
func (s State) Sync(e Env) State {
	if e.All == nil {
		return s
	}

	s.facts = e.All()
	if e.Waiting != nil {
		s.waiting = e.Waiting()
	}

	s.read = true
	s.sel = min(max(s.sel, 0), s.last())

	return s.keepSeen(e)
}

// pageRules is how far a page key moves: far enough to be worth pressing,
// and short enough that the reader can still tell where they landed.
const pageRules = 10

// last is the bottom row the cursor can be on: the tray, and then the facts
// under it, in the order they are drawn.
func (s State) last() int { return max(len(s.waiting)+len(s.facts)-1, 0) }

// View is the screen drawn: a title, the facts that belong to no repository,
// then each repository's own.
func (s State) View(h, w int, e Env) []string {
	return s.rows(h, w, e)
}

// Key is every key on this screen.
//
// Everything it hands back goes through keepSeen on the way out. A gesture
// changes how tall the foot is — the form is six rows where the line of keys
// was one — and a list left where it was when the window under it shrank is
// a list showing rows nobody is looking at.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	next, out := s.key(msg, e)
	if out.Leave {
		return next, out
	}

	return next.keepSeen(e), out
}

// Move walks the cursor, and brings the list with it.
func (s State) Move(d int, e Env) State { return s.move(d, e) }

// Scroll is the wheel, a notch of which is one rule rather than three lines.
//
// Three lines is what a plain list moves and about what one rule is: a rule
// is its sentence wrapped, and the one under the cursor says where it came
// from as well. Counted in lines the wheel crawled through a single rule;
// counted in rules it covers the distance three plain rows do everywhere
// else.
func (s State) Scroll(d int, e Env) State { return s.move(d, e) }

// Hit is what the screen has at that cell.
func (s State) Hit(x, y int, e Env) point.Target { return s.hit(x, y, e) }

// PointAt puts the cursor on the row that was clicked, and says whether it
// was already there — which is what makes the second click the one that
// opens.
func (s State) PointAt(i int, e Env) (State, bool) { return s.pointAt(i, e) }

// Chosen is the row under the cursor being opened: the second of the
// pointer's two clicks, and nothing a key does not also do.
func (s State) Chosen(e Env) (State, Out) { return s.chosen(e) }
