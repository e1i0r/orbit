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

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/keymap"
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
	// Keep writes one of them down as a fact of yours. The words are handed
	// over rather than read back out of the tray, because correcting is how
	// most of these are accepted.
	Keep func(at time.Time, phrase, check string) error
	// Drop says it was not a rule. The sentence stays in the thread where it
	// was said, which is where it belonged all along.
	Drop func(at time.Time) error
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
	// holds the two things about it that are text: what it says, and the
	// command that decides whether it can stop the work.
	//
	// Two fields and not the whole record. A fact's scope and its source are
	// what make it traceable, and neither is something to retype — the
	// source is where it came from, which nobody may edit, and the scope is
	// moved with its own gesture rather than by typing a path.
	editing bool
	field   int
	in      [factFields]typing.Field
}

// The two fields of a fact that are typed into.
const (
	factPhrase = iota
	factCheck
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

	return s
}

// last is the bottom row the cursor can be on: the tray, and then the facts
// under it, in the order they are drawn.
func (s State) last() int { return max(len(s.waiting)+len(s.facts)-1, 0) }

// View is the screen drawn: a title, the facts that belong to no repository,
// then each repository's own.
func (s State) View(h, w int, e Env) []string {
	return s.rows(h, w, e)
}

// Key is every key on this screen.
func (s State) Key(msg tea.KeyPressMsg, e Env) (State, Out) {
	if s.editing {
		return s.editingKey(msg, e)
	}

	switch {
	case msg.Code == tea.KeyEscape || key.Matches(msg, e.Keys.Back):
		return State{}, Out{Leave: true}
	case msg.Code == tea.KeyUp:
		s.sel = max(s.sel-1, 0)
		return s, Out{}
	case msg.Code == tea.KeyDown:
		s.sel = min(s.sel+1, s.last())
		return s, Out{}
	case msg.Code == tea.KeySpace:
		return s.turnFact(e)
	case msg.Code == 'e' || msg.Code == 'E':
		return s.editFact(e), Out{}
	case msg.Code == 'k' || msg.Code == 'K':
		return s.keepAsSaid(e)
	case msg.Code == 'd' || msg.Code == 'D':
		return s.dropSaid(e)
	case msg.Code == 'n' || msg.Code == 'N':
		return s.newFact(e), Out{}
	case msg.Code == tea.KeyLeft:
		return s.moveFact(wider, e)
	case msg.Code == tea.KeyRight:
		return s.moveFact(narrower, e)
	}

	return s, Out{}
}

// The two directions a fact can be moved in.
const (
	wider    = -1
	narrower = 1
)

// moveFact widens a fact to everything, or narrows it back to the repository.
//
// It is the common correction: a rule written in the supervisor is about the
// repository being worked in by default, and then turns out to be true
// everywhere. The language level is not on this ladder — a fact is about a
// language because somebody said so, not because it drifted there.
func (s State) moveFact(dir int, e Env) (State, Out) {
	was, ok := s.onFact()
	if e.Replace == nil || !ok {
		return s, Out{}
	}

	now := was
	switch {
	case dir == wider && was.Scope.Kind == knowledge.Repo:
		now.Scope = knowledge.Scope{Kind: knowledge.General}
	case dir == narrower && was.Scope.Kind == knowledge.General:
		here := hereScope(e)
		if here.Kind != knowledge.Repo {
			return s, said(e.Words.T("knowledge.no_repo_here",
				"there is more than one repository here; say which with the supervisor"))
		}

		now.Scope = here
	default:
		return s, Out{}
	}

	if err := e.Replace(was, now); err != nil {
		return s, said(err.Error())
	}

	return s.Sync(e), Out{}
}

// turnFact turns the fact under the cursor off, or on again.
//
// Off and not deleted: disagreeing with a fact and losing the record that it
// was ever there are different things, and the second is not something a
// keystroke should do. What it stops is the fact being told and the gate
// refusing work over it.
func (s State) turnFact(e Env) (State, Out) {
	f, ok := s.onFact()
	if e.Turn == nil || !ok {
		return s, Out{}
	}

	f.Off = !f.Off

	if err := e.Turn(f); err != nil {
		return s, said(err.Error())
	}

	return s.Sync(e), Out{}
}

// ordered is the facts in the order the screen draws them: the ones that
// belong to no repository first, then each repository's own.
//
// General first because it is what applies everywhere and what somebody
// looking for "why did it do that" checks before anything narrower.
func (s State) ordered() (rootless, owned []knowledge.Fact) {
	for _, f := range s.facts {
		if f.Scope.Kind == knowledge.General || f.Scope.Kind == knowledge.Language {
			rootless = append(rootless, f)
			continue
		}

		owned = append(owned, f)
	}

	return rootless, owned
}
