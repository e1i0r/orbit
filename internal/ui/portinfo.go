package ui

// What the window is told about the things it draws.
//
// port.go is the contract — the interfaces the window asks through and the
// closures it is handed. These are the answers those ports come back with,
// and they are here because a struct of fields cannot be split across files
// while the descriptions beside them can: one file was over the ceiling.
//
// What the window is told about the engines and their quota is in
// internal/ui/roster, because the screens that draw it are packages of their
// own and a type they cannot name is a type they cannot be handed.

import (
	"github.com/e1i0r/orbit/internal/words"
)

// Command is one row of the palette: what the window shows of a command,
// and nothing of what the command does.
//
// About and Because are functions of a printer rather than strings because
// both are sentences a reader reads, and sentences go through
// internal/words like every other line this window draws — which also lets
// them follow a language changed after this slice was handed over.
type Command struct {
	Name  string // as the reader types it
	Args  string // the usage fragment after the name; empty when none
	About func(*words.Printer) string

	Refused bool                        // the window does not run it here
	Because func(*words.Printer) string // why, when Refused is set

	// NeedsArgs says the command refuses when it is given none, and Args
	// says what it wants. The command line can give it those and the
	// board's menu cannot, so it is the menu that reads this.
	NeedsArgs bool

	// AboutATask keeps the command off the board's menu. That menu is
	// opened on no row, and a verb about one task has no task there; the
	// menu of the row it is about is where it belongs.
	AboutATask bool
}
