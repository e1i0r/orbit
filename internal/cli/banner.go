package cli

import (
	"io"
	"os"

	"charm.land/lipgloss/v2"
)

// banner is the program's mark with its rings on, beside its name.
//
// The drawing is thirteen cells a row and plain ASCII: an oblate body —
// planets are wider than they are tall — with the o above the middle and
// the rings as an ellipse crossing in front of it, past either side.
// ASCII is the one width every terminal agrees on — a mark like the header
// pill's ◉ reads two cells on fonts that render it so, and the line beside
// it comes out shifted.
//
// Colour is ANSI by number rather than hex: hexes live in internal/ui/theme
// and nowhere else, and a number follows the terminal's own theme instead
// of fighting it — teal for the body, after the header badge, magenta for
// the rings, after colibri's bird. Pipes, logs and tests get the plain
// drawing; colour there is escape codes in a file.
func banner(version, tagline string, colored bool) string {
	body := func(s string) string { return s }
	ring := func(s string) string { return s }
	name := func(s string) string { return s }

	if colored {
		bodyOf := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
		ringOf := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
		nameOf := lipgloss.NewStyle().Bold(true)
		body = func(s string) string { return bodyOf.Render(s) }
		ring = func(s string) string { return ringOf.Render(s) }
		name = func(s string) string { return nameOf.Render(s) }
	}

	return "    " + body("_____") + "      " + name("orbit "+version) + "\n" +
		"   " + body("/     \\") + "     " + tagline + "\n" +
		ring("--(   ") + body("o") + ring("   )--") + "\n" +
		"   " + body("\\_____/") + "\n"
}

// useColor is whether the banner may spend ANSI: on the reader's own
// terminal, and not when they asked for none. Anything piped, logged or
// tested answers false, which is the same terminal test top's frame choice
// makes — a writer that is not the process's own stdout collected the
// output, and a pipe is owed text, not escapes.
func useColor(out io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	return interactive(out)
}
