package ui

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

// The second half of the window's vocabulary.
//
// theme.go names what a piece of text means — running, failed, needs you.
// This file names where it sits and how much of the reader's attention it is
// asking for: the paper under a card, the line around it, and the three
// weights of text that can be set on it.
//
// The two are separate questions. A failed phase is Bad whether its name is
// the loudest thing on the pane or a qualifier trailing behind it, and a card
// is raised whatever is written on it. Asking them as one left the panes with
// seven meanings and two weights, Dim and not-Dim, so the prose a reader came
// for was painted the same grey as the keyboard hints.
//
// This is the two-layer model of app.frauddi.com's tokens.css: the hexes live
// here, and a pane asks for the job rather than for the colour.

// Tone is how much a piece of text is meant to be noticed.
type Tone int

// The three weights, and the order is the ladder: each rung recedes from
// the one above it. Primary is the zero value because text nobody assigned a
// tone to is text somebody wrote to be read.
const (
	Primary   Tone = iota // what the reader came for: prose, titles, values
	Secondary             // what qualifies it: a cost, a count, a verdict
	Tertiary              // furniture: labels, keys, rules, separators
)

// Level is how far a surface stands from the window's own paper.
type Level int

// The three levels. Base is the zero value: a surface nobody chose is the
// window's own paper, which is the one that cannot be wrong.
const (
	Base   Level = iota // the window's paper, and the terminal's
	Raised              // a card: one step towards the reader
	Sunken              // a well: code, output, anything quoted verbatim
)

// Shell is one theme's paper and ink, as opposed to Palette, which is its
// meanings. Tertiary text is Palette.Dim: furniture already had a name.
type Shell struct {
	Text   string // Primary
	Muted  string // Secondary
	Base   string
	Raised string
	Sunken string
	Line   string // the border of a card at rest
}

// Every theme's shell. The neutrals are each theme's own published ones, so
// a card in dracula is dracula's current-line grey and not a grey invented
// here that happens to sit near it.
var themeShells = map[string]Shell{
	"frauddi": {
		Text: "#E2E8F0", Muted: "#94A3B8",
		Base: "#0B1016", Raised: "#131A23", Sunken: "#080C11", Line: "#1E2A38",
	},
	"monokai": {
		Text: "#F8F8F2", Muted: "#A59F85",
		Base: "#272822", Raised: "#31322C", Sunken: "#1E1F1C", Line: "#3E3D32",
	},
	"tokyo-night": {
		Text: "#C0CAF5", Muted: "#9AA5CE",
		Base: "#1A1B26", Raised: "#24283B", Sunken: "#16161E", Line: "#2F334D",
	},
	"dracula": {
		Text: "#F8F8F2", Muted: "#BFC7D5",
		Base: "#282A36", Raised: "#343746", Sunken: "#21222C", Line: "#44475A",
	},
	"nord": {
		Text: "#ECEFF4", Muted: "#D8DEE9",
		Base: "#2E3440", Raised: "#3B4252", Sunken: "#272C36", Line: "#434C5E",
	},
	"catppuccin": {
		Text: "#CDD6F4", Muted: "#A6ADC8",
		Base: "#1E1E2E", Raised: "#292C3C", Sunken: "#181825", Line: "#45475A",
	},
}

// WindowBackground is the paper the whole window is painted on: the current
// theme's Base, handed to the terminal once per frame rather than stamped
// onto every cell.
//
// The terminal is asked to change its own background for as long as the
// program is up, which is the one way to do this that costs nothing per
// line. Painting it into the frame instead would mean every renderer in this
// package remembering to fill the cells it does not write, and a renderer
// that forgets leaves a stripe of the reader's own console across the
// window.
//
// It is the theme's own colour and not one invented here, for the reason the
// shells above are the published neutrals: a window that sits half a shade
// off the palette it is drawing looks like a bug in the palette.
func WindowBackground() color.Color {
	return lipgloss.Color(currentShell().Base)
}

// WindowForeground is the ink that goes with that paper: the theme's Text.
//
// It is set with the background and for the same reason. A terminal told to
// change its background keeps drawing unstyled text in whatever foreground
// the reader configured, and the two were never chosen together — a console
// set to near-black text over a light scheme is unreadable the moment the
// paper under it goes dark. Handing both over as a pair is the only way this
// package can promise a legible frame on a terminal it did not configure.
func WindowForeground() color.Color {
	return lipgloss.Color(currentShell().Text)
}

func currentShell() Shell {
	if s, ok := themeShells[currentThemeName]; ok {
		return s
	}

	return themeShells[defaultTheme]
}

// Text is the style for one weight of text. It sets a foreground and nothing
// else, so it composes with a surface rather than fighting one.
func Text(t Tone) lipgloss.Style {
	sh := currentShell()
	style := lipgloss.NewStyle()

	switch t {
	case Secondary:
		return style.Foreground(lipgloss.Color(sh.Muted))
	case Tertiary:
		return style.Foreground(lipgloss.Color(currentPalette().Dim))
	case Primary:
		return style.Foreground(lipgloss.Color(sh.Text))
	}

	return style.Foreground(lipgloss.Color(sh.Text))
}

// Surface is the paper a block is set on. Like Text it sets one attribute,
// because a card is a background and a border and a padding chosen by
// whoever draws it, not a look decided here.
func Surface(l Level) lipgloss.Style {
	sh := currentShell()
	style := lipgloss.NewStyle()

	switch l {
	case Raised:
		return style.Background(lipgloss.Color(sh.Raised))
	case Sunken:
		return style.Background(lipgloss.Color(sh.Sunken))
	case Base:
		return style.Background(lipgloss.Color(sh.Base))
	}

	return style.Background(lipgloss.Color(sh.Base))
}

// Rule is the colour of a border at rest.
func Rule() color.Color {
	return lipgloss.Color(currentShell().Line)
}

// tintWeight is how far a badge's fill is carried down onto the paper it
// sits on. At 0.86 the hue is still legible as itself and the label on top
// of it still reads, which is the trade a soft badge is making.
const tintWeight = 0.86

// Tint is a badge's fill: the role's own hue laid most of the way down onto
// the paper, with the hue itself as the text on top of it.
//
// It is derived rather than written down because a fill per role per theme is
// thirty more hexes to keep in step with the six they are derived from, and
// the first one that drifts is a badge nobody can read.
func Tint(r Role) lipgloss.Style {
	hue := roleColour(r)

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(hue)).
		Background(lipgloss.Color(mix(hue, currentShell().Base, tintWeight))).
		Bold(true).
		Padding(0, 1)
}

// roleColour is the one hex a role stands for, without the weight and the
// boldness Paint decides on top of it.
func roleColour(r Role) string {
	pal := currentPalette()

	switch r {
	case OK:
		return pal.OK
	case Bad:
		return pal.Bad
	case Warn:
		return pal.Warn
	case Live:
		return pal.Live
	case Dim:
		return pal.Dim
	case Sel:
		return pal.SelBlock
	case Accent:
		return pal.Accent
	}

	return pal.Accent
}

// mix walks from one colour to another: weight 0 is the first unchanged, 1
// is the second. A hex neither side can parse is left as it was found, and
// TestEveryTokenIsAWellFormedHex is what keeps that from happening quietly.
func mix(from, to string, weight float64) string {
	a, aOK := hexRGB(from)
	b, bOK := hexRGB(to)

	if !aOK || !bOK {
		return from
	}

	var out [3]float64
	for i := range out {
		out[i] = a[i] + (b[i]-a[i])*weight
	}

	return fmt.Sprintf("#%02X%02X%02X", int(out[0]+0.5), int(out[1]+0.5), int(out[2]+0.5))
}

// hexRGB reads #RRGGBB into three channels. Nothing shorter is accepted: a
// three-digit hex and a named colour are both things this file does not
// write, and guessing at one would hide the token that went missing.
//
// It walks the string rather than indexing into it because internal/ui does
// not measure strings in bytes, and a rule with an exception for the one
// caller that is certain it holds ASCII is a rule with an exception.
func hexRGB(s string) ([3]float64, bool) {
	var (
		out    [3]float64
		digits [6]int
		found  int
	)

	for i, c := range s {
		if i == 0 {
			if c != '#' {
				return out, false
			}

			continue
		}

		v, ok := hexDigit(c)
		if !ok || found == len(digits) {
			return out, false
		}

		digits[found] = v
		found++
	}

	if found != len(digits) {
		return out, false
	}

	for i := range out {
		out[i] = float64(digits[2*i]*16 + digits[2*i+1])
	}

	return out, true
}

func hexDigit(c rune) (int, bool) {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0'), true
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10, true
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10, true
	}

	return 0, false
}
