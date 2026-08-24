// Package ui is the window's vocabulary: what a piece of text means and how
// that meaning is painted, which key stands for which gesture, and what can
// be done to the task under the cursor. It holds no screen — the screens are
// built on top of it — and it holds no authority: it cannot append an event,
// build a path under the state root or start a model, because arch.layers
// does not let it import the packages that can.
package ui

import "charm.land/lipgloss/v2"

// Role is what a piece of text means. Colour is defined here and nowhere
// else in the window: a screen asks for a role, never for a colour, so the
// question "what does cyan mean in this program" has exactly one answer.
//
// The program this replaces painted cyan for both "running" and "heading",
// so a pane of headings read as a pane of running things, and there was no
// single place to go and fix it.
type Role int

// The seven roles, and there are deliberately only seven.
//
// Accent is first so that it is the zero value: a Role nobody set paints as
// the program's own colour, which is the harmless answer — the failing
// answers would be Bad, which cries wolf, or Live, which says something is
// happening when nothing is.
const (
	Accent Role = iota // the one accent: orbit's own colour, used sparingly
	OK                 // finished, passed, green
	Bad                // failed, and the only other thing that is ever bold red
	Warn               // needs you, but nothing is broken
	Live               // happening now, and reserved for that alone
	Dim                // present and not the point: counts, hints, furniture
	Sel                // the cursor's row
)

// The palette, as ANSI 256 colour numbers.
//
// They are constants rather than package variables because a package
// variable is state, and state a caller can reach is state a caller can
// change halfway through a frame.
//
// Every one of them is chosen to read on a light terminal and on a dark one,
// which is what lets Paint be a pure function of its Role. That is a real
// constraint and it costs something: a palette free to know the background
// could use a deeper red on white and a brighter one on black. It was taken
// anyway, because the alternative is asking the terminal what colour it is
// from inside the paint function — and the only synchronous way to do that
// blocks on terminal I/O outside the event loop, which is a hang, not a
// colour. When the window does want a light/dark pair, the shape is a Theme
// value built once from a tea.BackgroundColorMsg the event loop already
// receives, fed to lipgloss.LightDark and passed down as a parameter, and
// never a package variable Paint consults behind its caller's back.
const (
	accentColor = "39"  // a blue that is neither the terminal's nor a state's
	okColor     = "71"  // green, muted enough not to compete with bad
	badColor    = "167" // red, and the only red
	warnColor   = "179" // amber: attention, not alarm
	liveColor   = "37"  // teal, and it means one thing only
	dimColor    = "245" // grey, and a real step away from body text
	selColor    = "252" // the cursor block's text
	selBlock    = "240" // the cursor block itself
)

// Roles returns every role, in the order they are declared.
//
// It builds a new slice on every call rather than handing out a package
// variable, for the reason internal/view gives about Bands: a slice a caller
// can reorder is package state, and this package holds none.
func Roles() []Role {
	return []Role{Accent, OK, Bad, Warn, Live, Dim, Sel}
}

// Paint is the style for one role, and it is a pure function of that role:
// no setting is read, no terminal is asked anything, no answer is cached.
// That is what lets a test build all seven with no terminal in the room, and
// it is what keeps a colour decision out of the event loop.
//
// Weight is applied before colour, and it carries the hierarchy on its own.
// A NO_COLOR terminal, a monochrome ssh session and `--once` piped into a
// file all lose every colour here and keep bold, faint and the cursor block
// — so the screen still has three levels rather than one. Bold is spent on
// the three things worth interrupting a scan for: the program's own accent,
// a failure, and the thing happening right now. Green and amber are colour
// alone, because a row of bold green ticks shouts as loudly as a failure and
// then nothing on the screen is loud.
//
// It is total. A Role no constant names — arithmetic that went wrong, a
// value read from somewhere it should not have been — paints as plain text
// rather than panicking in the middle of a frame or borrowing a colour that
// already means something else.
func Paint(r Role) lipgloss.Style {
	style := lipgloss.NewStyle()
	switch r {
	case Accent:
		return style.Bold(true).Foreground(lipgloss.Color(accentColor))
	case OK:
		return style.Foreground(lipgloss.Color(okColor))
	case Bad:
		return style.Bold(true).Foreground(lipgloss.Color(badColor))
	case Warn:
		return style.Foreground(lipgloss.Color(warnColor))
	case Live:
		return style.Bold(true).Foreground(lipgloss.Color(liveColor))
	case Dim:
		return style.Faint(true).Foreground(lipgloss.Color(dimColor))
	case Sel:
		// A block, and not inverse video. The cursor is on screen at all
		// times, and reverse swaps whatever the terminal's own two colours
		// are — which on most terminals is the loudest thing the screen can
		// do. A stated background is quieter and, unlike reverse, it is the
		// same shade on everybody's terminal.
		return style.Foreground(lipgloss.Color(selColor)).Background(lipgloss.Color(selBlock))
	}
	return style
}
