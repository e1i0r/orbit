package ui

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
)

// withTheme runs fn on one theme and puts the window back on the one it was
// on, because the theme is package state and a test that leaves it moved is
// a failure in whatever runs next.
func withTheme(t *testing.T, name string, fn func()) {
	t.Helper()

	was := CurrentTheme()

	defer SetCurrentTheme(was)

	SetCurrentTheme(name)

	if CurrentTheme() != name {
		t.Fatalf("SetCurrentTheme(%q) left the window on %q", name, CurrentTheme())
	}

	fn()
}

// TestEveryThemeHasAShell. The meanings and the paper are two maps keyed by
// the same names, which is two places to add a theme and one place to forget
// it. A theme with no shell falls back to frauddi's paper, so the omission
// shows up as one theme quietly wearing another's greys.
func TestEveryThemeHasAShell(t *testing.T) {
	for _, name := range AvailableThemes() {
		if _, ok := themePalettes[name]; !ok {
			t.Errorf("theme %q is offered and has no palette", name)
		}

		if _, ok := themeShells[name]; !ok {
			t.Errorf("theme %q is offered and has no shell", name)
		}
	}

	if len(themeShells) != len(themePalettes) {
		t.Errorf("%d shells for %d palettes", len(themeShells), len(themePalettes))
	}
}

// TestEveryTokenIsAWellFormedHex. mix returns its first argument unchanged
// when it cannot read either colour, so a mistyped hex does not crash and
// does not look wrong either — it produces a badge whose fill is the hue at
// full strength, which reads as a design decision.
func TestEveryTokenIsAWellFormedHex(t *testing.T) {
	for name, sh := range themeShells {
		for label, hex := range map[string]string{
			"Text": sh.Text, "Muted": sh.Muted, "Base": sh.Base,
			"Raised": sh.Raised, "Sunken": sh.Sunken, "Line": sh.Line,
		} {
			if _, ok := hexRGB(hex); !ok {
				t.Errorf("%s.%s = %q, which is not #RRGGBB", name, label, hex)
			}
		}
	}

	for name, pal := range themePalettes {
		for label, hex := range map[string]string{
			"Accent": pal.Accent, "OK": pal.OK, "Bad": pal.Bad, "Warn": pal.Warn,
			"Live": pal.Live, "Dim": pal.Dim, "SelText": pal.SelText, "SelBlock": pal.SelBlock,
		} {
			if _, ok := hexRGB(hex); !ok {
				t.Errorf("%s.%s = %q, which is not #RRGGBB", name, label, hex)
			}
		}
	}
}

// TestMixLandsOnBothEnds pins the direction of the walk. Reversed, every
// badge in the window is filled with the paper's own colour and disappears,
// and every fill is still a valid hex while it happens.
func TestMixLandsOnBothEnds(t *testing.T) {
	const from, to = "#102040", "#204080"

	if got := mix(from, to, 0); got != "#102040" {
		t.Errorf("mix(%s, %s, 0) = %s, want the first colour", from, to, got)
	}

	if got := mix(from, to, 1); got != "#204080" {
		t.Errorf("mix(%s, %s, 1) = %s, want the second colour", from, to, got)
	}

	if got := mix(from, to, 0.5); got != "#183060" {
		t.Errorf("mix(%s, %s, 0.5) = %s, want the midpoint #183060", from, to, got)
	}

	if got := mix("not a colour", to, 0.5); got != "not a colour" {
		t.Errorf("mix on an unreadable colour = %q, want it back unchanged", got)
	}
}

// rgb8 reads a rendered colour back out as three channels, so a test can ask
// where a style's colour actually landed rather than what it was built from.
func rgb8(c color.Color) [3]float64 {
	r, g, b, _ := c.RGBA()

	return [3]float64{float64(r >> 8), float64(g >> 8), float64(b >> 8)}
}

func apart(a, b [3]float64) float64 {
	var d float64

	for i := range a {
		d += (a[i] - b[i]) * (a[i] - b[i])
	}

	return d
}

// TestATintSitsOnThePaper. A badge's fill is the point of the tint: the hue
// carried far enough down onto the paper that a label can be read on top of
// it. Carried too little it is the saturated block the panes used to draw,
// which is legible and shouts, and shouting is what a soft badge exists to
// stop.
func TestATintSitsOnThePaper(t *testing.T) {
	for _, name := range AvailableThemes() {
		withTheme(t, name, func() {
			paper := rgb8(lipgloss.Color(currentShell().Base))

			for _, r := range []Role{OK, Bad, Warn, Live, Accent} {
				hue := rgb8(lipgloss.Color(roleColour(r)))
				fill := rgb8(Tint(r).GetBackground())

				if apart(fill, paper) >= apart(fill, hue) {
					t.Errorf("%s: role %d fills nearer its hue than its paper", name, r)
				}
			}
		})
	}
}

// TestTheThreeTonesAreThreeColours. The ladder exists to give prose, its
// qualifiers and the furniture three different weights. Any two of them
// equal and the pane is back to the flat wall this replaced, with the code
// still asking for the right thing.
func TestTheThreeTonesAreThreeColours(t *testing.T) {
	for _, name := range AvailableThemes() {
		withTheme(t, name, func() {
			one := rgb8(Text(Primary).GetForeground())
			two := rgb8(Text(Secondary).GetForeground())
			three := rgb8(Text(Tertiary).GetForeground())

			if apart(one, two) == 0 || apart(two, three) == 0 || apart(one, three) == 0 {
				t.Errorf("%s: the three tones are not three colours: %v %v %v", name, one, two, three)
			}

			// Each rung recedes: what the reader came for is the brightest
			// thing on the pane, and the furniture is the dimmest.
			if lum(one) <= lum(two) || lum(two) <= lum(three) {
				t.Errorf("%s: the tones do not descend: %v %v %v", name, one, two, three)
			}
		})
	}
}

func lum(c [3]float64) float64 {
	return 0.2126*c[0] + 0.7152*c[1] + 0.0722*c[2]
}
