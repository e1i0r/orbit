package theme

// The paper the window is drawn on, and the styles that compose with it.
//
// These are one attribute each on purpose: a card is a background and a
// border and a padding chosen by whoever draws it, and a style that set all
// three would decide the look of every block in the window from here.

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestThePaperAndTheInkAreChosenTogether. A terminal told to change its
// background keeps drawing unstyled text in whatever foreground the reader
// configured, and the two were never chosen together.
func TestThePaperAndTheInkAreChosenTogether(t *testing.T) {
	for _, name := range AvailableThemes() {
		SetCurrentTheme(name)

		back, front := WindowBackground(), WindowForeground()
		if back == nil || front == nil {
			t.Fatalf("%s hands over background=%v foreground=%v", name, back, front)
		}

		if fmtColor(back) == fmtColor(front) {
			t.Errorf("%s draws its text in the same colour as its paper", name)
		}
	}
}

// TestATonelessNameFallsBackToTheDefaultTheme, so a settings file naming a
// theme this build dropped still draws a window.
func TestATonelessNameFallsBackToTheDefaultTheme(t *testing.T) {
	SetCurrentTheme("no such theme")

	if got := currentShell(); got.Base == "" || got.Text == "" {
		t.Errorf("a theme nobody has answers with %+v", got)
	}

	SetCurrentTheme(DefaultTheme)
}

// TestEachWeightOfTextIsItsOwnColourAndNothingElse.
func TestEachWeightOfTextIsItsOwnColourAndNothingElse(t *testing.T) {
	SetCurrentTheme(DefaultTheme)

	seen := map[string]bool{}

	for _, tone := range []Tone{Primary, Secondary, Tertiary} {
		got := Text(tone).Render("x")
		if got == "x" {
			t.Errorf("tone %v painted nothing", tone)
		}

		seen[got] = true

		// One attribute: a foreground and no background behind it.
		if strings.Contains(got, "48;2;") {
			t.Errorf("tone %v set a background as well: %q", tone, got)
		}
	}

	if len(seen) != 3 {
		t.Errorf("the three weights of text are drawn %d different ways", len(seen))
	}
}

// TestEachSurfaceIsItsOwnPaperAndNothingElse.
func TestEachSurfaceIsItsOwnPaperAndNothingElse(t *testing.T) {
	SetCurrentTheme(DefaultTheme)

	seen := map[string]bool{}

	for _, level := range []Level{Base, Raised, Sunken} {
		got := Surface(level).Render("x")
		if got == "x" {
			t.Errorf("level %v painted nothing", level)
		}

		seen[got] = true

		if strings.Contains(got, "38;2;") {
			t.Errorf("level %v set a foreground as well: %q", level, got)
		}
	}

	if len(seen) != 3 {
		t.Errorf("the three surfaces are drawn %d different ways", len(seen))
	}
}

// TestTheWindowsOwnFurnitureIsOneInk. The three bars are one surface as far
// as the eye is concerned, and a chip in the header a shade off the hints in
// the footer reads as two kinds of thing.
func TestTheWindowsOwnFurnitureIsOneInk(t *testing.T) {
	SetCurrentTheme(DefaultTheme)

	if Chrome().Render("x") != Text(Secondary).Render("x") {
		t.Error("the header's chips and the footer's hints are drawn differently")
	}

	if Rule() == nil {
		t.Error("a border at rest has no colour")
	}
}

// TestAPillIsDrawnThreeWaysAndTheThreeAreTold apart: one on offer, one the
// cursor is on, and the one in force.
//
// The badge that is only selected carries no mark, and that is width and not
// taste: the mark makes it two cells wider the moment it is chosen, and the
// name badge is the first thing on the header's line — two extra cells there
// move all four queue badges, which the pointer places by column.
func TestAPillIsDrawnThreeWaysAndTheThreeAreTold(t *testing.T) {
	SetCurrentTheme(DefaultTheme)

	const fg, bg = "#000000", "#ffffff"

	plain, chosen, inForce := Pill("x", fg, bg), PillSelected("x", fg, bg), PillActive("x", fg, bg)
	if plain == chosen || chosen == inForce || plain == inForce {
		t.Errorf("the three states of a pill are drawn %q %q %q", plain, chosen, inForce)
	}

	if lipgloss.Width(chosen) != lipgloss.Width(plain) {
		t.Errorf("choosing a pill changed its width from %d to %d",
			lipgloss.Width(plain), lipgloss.Width(chosen))
	}

	if lipgloss.Width(inForce) <= lipgloss.Width(chosen) {
		t.Error("the pill in force carries no mark to tell it from the chosen one")
	}
}

// TestATabSaysWhetherItIsTheOneBeingRead.
func TestATabSaysWhetherItIsTheOneBeingRead(t *testing.T) {
	SetCurrentTheme(DefaultTheme)

	if ActiveTabBadge("1") == InactiveTabBadge("1") {
		t.Error("the tab being read is drawn like the ones that are not")
	}
}

// fmtColor is a colour as its four channels, for comparing two of them.
func fmtColor(c color.Color) string {
	r, g, b, a := c.RGBA()

	return fmt.Sprintf("%d/%d/%d/%d", r, g, b, a)
}
