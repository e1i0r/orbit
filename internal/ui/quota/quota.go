// Package quota is the screen that says what is left of every engine's
// windows, and when each one comes back.
//
// The header carries a chip for the engine a run would go to, because a chip
// is what fits beside the engine's name and the share used is the part a
// reader glances at. It was drawn and answered nothing — the fact behind a
// percentage is several windows of several engines, and one line of the
// header has room for none of that. This screen is where the pointer lands
// now, and it is the only place the whole reading is written down: every
// engine Orbit can run, the windows each has, and the hour each returns.
//
// It is entered through this file. The screen holds nothing — there is
// nothing on it to choose — so there is no State: it is drawn from what the
// Env answers and left again.
package quota

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/roster"
	"github.com/e1i0r/orbit/internal/words"
)

// Env is what this screen needs of the world: the words it speaks, the keys
// it answers, and the two doors that say which engines there are and what is
// left of each one.
type Env struct {
	Words *words.Printer
	Keys  keymap.Keys
	// Engines is every engine the window offers, in the order the picker
	// lists them: a reader who chose one two screens ago and came looking
	// for what it has left would read its absence as an answer, and it
	// would be the wrong one.
	Engines func() []roster.Engine
	// Read is what is known about one engine's quota right now.
	Read func(engine string) roster.Reading
}

// Out is what the screen asks the window for.
type Out struct {
	// Leave is the reader closing the screen.
	Leave bool
}

// Key answers the keyboard while the screen is up.
//
// Nothing on it is chosen, so the only keys are the ones that leave: esc,
// back, and q. Every other key does nothing rather than reaching the board
// underneath, which is the rule the cheat sheet and the supervisor's thread
// follow for the same reason: a reader looking at one screen should not be
// able to move a cursor on another.
//
// q closes this screen rather than the window, as it does on every other
// screen that is not the board. It is here because the screen is opened with
// Q, and a reader whose caps lock is down opens it without meaning to — and
// then presses the key that closes everything else and is answered with
// nothing.
func Key(msg tea.KeyPressMsg, e Env) Out {
	if msg.Code == tea.KeyEscape || key.Matches(msg, e.Keys.Back) || key.Matches(msg, e.Keys.Quit) {
		return Out{Leave: true}
	}

	return Out{}
}

// Readings is every engine the window offers, each with what is known about
// its quota right now. An engine with no source is still a row — silence is
// what it says instead.
func Readings(e Env) []roster.Reading {
	if e.Read == nil || e.Engines == nil {
		return nil
	}

	engines := e.Engines()

	out := make([]roster.Reading, 0, len(engines))
	for _, eng := range engines {
		out = append(out, e.Read(eng.Name))
	}

	return out
}

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}
