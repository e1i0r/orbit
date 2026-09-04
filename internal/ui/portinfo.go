package ui

// What the window is told about the things it draws.
//
// port.go is the contract — the interfaces the window asks through and the
// closures it is handed. These are the answers those ports come back with,
// and they are here because a struct of fields cannot be split across files
// while the descriptions beside them can: one file was over the ceiling.

import (
	"time"

	"github.com/e1i0r/orbit/internal/words"
)

// QuotaReading is what the window learns about one engine's quota.
//
// Money and Sourced are carried as answers rather than as the billing mode
// they were derived from, because the mode is not this package's to read:
// internal/quota decides what a number about an engine means, and the window
// is told the outcome. Sourced is separate from a window count for the
// difference it protects — an engine nobody can read a window for is not an
// engine with no window left.
type QuotaReading struct {
	Engine  string
	Money   bool
	Sourced bool
	Windows []QuotaWindow
}

// QuotaWindow is what the window learns about remaining quota.
type QuotaWindow struct {
	Key      string
	Label    string
	Pct      float64
	ResetsIn time.Duration
}

// EngineInfo is what the window knows about an engine's dials and setup.
//
// Setup is a function of a printer for the reason Command.About is: the
// steps are sentences a reader reads, so they go through internal/words like
// every other line on this screen, and they follow a language changed after
// this slice was handed over.
type EngineInfo struct {
	Name      string
	Available bool
	Setup     func(*words.Printer) []string
	Models    []ChoiceInfo
	Efforts   []ChoiceInfo
	CanThink  bool
}

// ChoiceInfo is one selectable value for an engine dial.
type ChoiceInfo struct {
	ID    string
	Label string
}

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
