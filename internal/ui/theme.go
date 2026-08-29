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

// Palette holds the color tokens for one visual theme.
type Palette struct {
	Accent   string
	OK       string
	Bad      string
	Warn     string
	Live     string
	Dim      string
	SelText  string
	SelBlock string
}

var themePalettes = map[string]Palette{
	"frauddi": {
		Accent:   "#38BDF8",
		OK:       "#16B798",
		Bad:      "#EF4444",
		Warn:     "#F59E0B",
		Live:     "#2DD4BF",
		Dim:      "#64748B",
		SelText:  "#FFFFFF",
		SelBlock: "#0F766E",
	},
	"monokai": {
		Accent:   "#66D9EF",
		OK:       "#A6E22E",
		Bad:      "#F92672",
		Warn:     "#FD971F",
		Live:     "#00E5FF",
		Dim:      "#75715E",
		SelText:  "#FFFFFF",
		SelBlock: "#005F87",
	},
	"tokyo-night": {
		Accent:   "#7AA2F7",
		OK:       "#9ECE6A",
		Bad:      "#F7768E",
		Warn:     "#FF9E64",
		Live:     "#7DCFFF",
		Dim:      "#565F89",
		SelText:  "#FFFFFF",
		SelBlock: "#283457",
	},
	"dracula": {
		Accent:   "#BD93F9",
		OK:       "#50FA7B",
		Bad:      "#FF5555",
		Warn:     "#FFB86C",
		Live:     "#8BE9FD",
		Dim:      "#6272A4",
		SelText:  "#FFFFFF",
		SelBlock: "#44475A",
	},
	"nord": {
		Accent:   "#88C0D0",
		OK:       "#A3BE8C",
		Bad:      "#BF616A",
		Warn:     "#EBCB8B",
		Live:     "#81A1C1",
		Dim:      "#4C566A",
		SelText:  "#ECEFF4",
		SelBlock: "#3B4252",
	},
	"catppuccin": {
		Accent:   "#89B4FA",
		OK:       "#A6E3A1",
		Bad:      "#F38BA8",
		Warn:     "#FAB387",
		Live:     "#94E2D5",
		Dim:      "#6C7086",
		SelText:  "#CDD6F4",
		SelBlock: "#313244",
	},
}

// defaultTheme is the theme a build starts on, and the one a settings file
// naming none is read as. The palette map below is keyed by name and a map
// has no first entry, so this says which one out loud rather than leaving
// two places to write the same word.
const defaultTheme = "frauddi"

var currentThemeName = defaultTheme

// AvailableThemes lists the default selectable themes.
func AvailableThemes() []string {
	return []string{defaultTheme, "monokai", "tokyo-night", "dracula", "nord", "catppuccin"}
}

// SetCurrentTheme sets the active theme for painting.
func SetCurrentTheme(name string) {
	if _, ok := themePalettes[name]; ok {
		currentThemeName = name
	}
}

// CurrentTheme returns the active theme name.
func CurrentTheme() string {
	return currentThemeName
}

func currentPalette() Palette {
	if p, ok := themePalettes[currentThemeName]; ok {
		return p
	}

	return themePalettes["monokai"]
}

// ink is a badge's two colours together: what its text is painted, and what
// that text sits on. Pill takes them apart because lipgloss does, but a
// badge is only legible as a pair, and half a pair is how you get white on
// white.
type ink struct{ fg, bg string }

// The queue badges' inks.
//
// These were eight hex literals spread across four argument lists in the
// header, which is the one thing the top of this file says does not happen
// anywhere in the window. They are the same in every theme on purpose: the
// four queues are the program's own vocabulary, not decoration, and a reader
// who learns that amber means "needs you" should not have to learn it again
// after changing the theme.
var (
	inkToDo     = ink{"#38BDF8", "#0C4A6E"}
	inkRunning  = ink{"#2DD4BF", "#134E4A"}
	inkNeedsYou = ink{"#FBBF24", "#78350F"}
	inkDone     = ink{"#4ADE80", "#14532D"}

	// inkUpgrade is the notice that a newer orbit is out: pastel mint on
	// deep emerald.
	inkUpgrade = ink{"#86EFAC", "#064E3B"}
)

// Pill renders text as a styled badge with background and padding.
func Pill(text string, fg, bg string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg)).
		Bold(true).
		Padding(0, 1).
		Render(text)
}

// PillSelected renders a badge as chosen, by swapping its two colours and
// nothing else.
//
// It is PillActive without the mark, and the difference is width. PillActive
// prefixes "▶ ", which makes the badge two cells wider the moment it is
// selected and shifts everything drawn after it. The queue badges can afford
// that. The name badge cannot: it is the first thing on the line, so two
// extra cells move all four queue badges, and hitHeader places every one of
// them by a column written down in target.go rather than by measuring what
// was drawn.
func PillSelected(text string, fg, bg string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(bg)).
		Background(lipgloss.Color(fg)).
		Bold(true).
		Padding(0, 1).
		Render(text)
}

// PillActive renders an active/focused badge with inverted contrast.
func PillActive(text string, fg, bg string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(bg)).
		Background(lipgloss.Color(fg)).
		Bold(true).
		Padding(0, 1).
		Render("▶ " + text)
}

// Roles returns every role, in the order they are declared.
func Roles() []Role {
	return []Role{Accent, OK, Bad, Warn, Live, Dim, Sel}
}

// Paint is the style for one role in the current theme.
func Paint(r Role) lipgloss.Style {
	style := lipgloss.NewStyle()
	pal := currentPalette()

	switch r {
	case Accent:
		return style.Bold(true).Foreground(lipgloss.Color(pal.Accent))
	case OK:
		return style.Foreground(lipgloss.Color(pal.OK))
	case Bad:
		return style.Bold(true).Foreground(lipgloss.Color(pal.Bad))
	case Warn:
		return style.Foreground(lipgloss.Color(pal.Warn))
	case Live:
		return style.Bold(true).Foreground(lipgloss.Color(pal.Live))
	case Dim:
		return style.Faint(true).Foreground(lipgloss.Color(pal.Dim))
	case Sel:
		return style.Foreground(lipgloss.Color(pal.SelText)).Background(lipgloss.Color(pal.SelBlock))
	}

	return style
}
