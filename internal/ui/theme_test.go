package ui

// The window on its theme's paper: that the background it draws on is the
// one the theme names, and that the two bars share one ink.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

func TestTheWindowIsPaintedOnItsThemesPaper(t *testing.T) {
	t.Cleanup(func() { theme.SetCurrentTheme("frauddi") })

	for _, tc := range []struct{ theme, want string }{
		{"frauddi", "#0B1016"},
		{"dracula", "#282A36"},
	} {
		theme.SetCurrentTheme(tc.theme)

		if got := theme.WindowBackground(); got != lipgloss.Color(tc.want) {
			t.Errorf("%s background = %v, want %s", tc.theme, got, tc.want)
		}
	}

	// The pair, not one of them: paper handed over without ink leaves every
	// unstyled cell in the foreground the reader's console was set to, which
	// was chosen against a different background.
	m, _ := testModel(t, 80, 24)

	v := m.View()
	if v.BackgroundColor == nil || v.ForegroundColor == nil {
		t.Errorf("the window draws paper %v and ink %v; it needs both", v.BackgroundColor, v.ForegroundColor)
	}
}

// The window's furniture is one ink and it is not a faint one.
//
// The header chips, the key bar's labels and the chips beside them are one
// surface as far as a reader is concerned, and they were three: chips and
// hints in faint grey, the cli chip in Live. Faint grey on a theme's own
// paper is text that is drawn and cannot be read, which is what the bars are
// for at the moment nobody knows what to press.
func TestTheBarsShareOneInkAndItIsNotFaint(t *testing.T) {
	m, _ := testModel(t, 140, 40)

	ink, _, ok := strings.Cut(theme.Chrome().Render("probe"), "probe")
	if !ok || ink == "" {
		t.Fatalf("Chrome() rendered %q, want a colour before the text", theme.Chrome().Render("probe"))
	}

	faint, _, _ := strings.Cut(theme.Paint(theme.Dim).Render("probe"), "probe")

	for _, line := range []struct{ name, drawn string }{
		{"header", m.headerLine(140)},
		{"bar", m.barLine(140)},
	} {
		if !strings.Contains(line.drawn, ink) {
			t.Errorf("the %s is drawn in no ink of Chrome's: %q", line.name, line.drawn)
		}

		if strings.Contains(line.drawn, faint) {
			t.Errorf("the %s still carries faint text: %q", line.name, line.drawn)
		}
	}
}
