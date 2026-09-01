package ui

// Seven roles, and every one of them has to be buildable with no terminal in
// the room: this suite never opens one, and a style that had to ask the
// terminal what colour its background is would be a style no test could
// build. The table is also where "weight before colour" is checked, because
// bold and faint are the whole of the hierarchy on a NO_COLOR terminal, a
// mono ssh session, and --once piped into a file.

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestEveryRoleIsAStyleAndSaysSoWithoutColour(t *testing.T) {
	cases := []struct {
		name       string
		role       Role
		bold       bool
		faint      bool
		background bool
	}{
		{"accent", Accent, true, false, false},
		{"ok", OK, false, false, false},
		{"bad", Bad, true, false, false},
		{"warn", Warn, false, false, false},
		{"live", Live, true, false, false},
		{"dim", Dim, false, true, false},
		{"sel", Sel, false, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			style := Paint(c.role)
			if style.GetBold() != c.bold {
				t.Errorf("bold is %v, want %v", style.GetBold(), c.bold)
			}

			if style.GetFaint() != c.faint {
				t.Errorf("faint is %v, want %v", style.GetFaint(), c.faint)
			}

			if unset(style.GetForeground()) {
				t.Error("no foreground — every role is a colour as well as a weight")
			}

			if unset(style.GetBackground()) == c.background {
				t.Errorf("background set is %v, want %v — only the cursor's role is a block", !unset(style.GetBackground()), c.background)
			}
		})
	}
}

// unset reports whether a colour is the absence of one.
func unset(c color.Color) bool {
	_, ok := c.(lipgloss.NoColor)
	return ok
}

// TestPaintIsAPureFunctionOfItsRole is the property that keeps the theme out
// of the event loop: the same role paints the same style every time, with no
// setting, no cached answer from the terminal and no package variable in
// between.
func TestPaintIsAPureFunctionOfItsRole(t *testing.T) {
	for _, role := range Roles() {
		first, second := Paint(role).Render("orbit"), Paint(role).Render("orbit")
		if first != second {
			t.Errorf("role %d painted %q and then %q", role, first, second)
		}
	}

	seen := map[string]Role{}

	for _, role := range Roles() {
		painted := Paint(role).Render("orbit")
		if other, ok := seen[painted]; ok {
			t.Errorf("roles %d and %d paint identically — a role that cannot be told from another is not a level", other, role)
		}

		seen[painted] = role
	}
}

// TestAnUnknownRolePaintsNothing keeps Paint total. A Role no constant names
// is a slip in arithmetic somewhere above, and the honest answer is plain
// text — not a panic in the middle of a frame, and not a colour that means
// something else.
func TestAnUnknownRolePaintsNothing(t *testing.T) {
	painted := Paint(Role(42)).Render("orbit")
	if painted != "orbit" {
		t.Errorf("an unnamed role painted %q, want the text unstyled", painted)
	}
}

func TestAvailableThemes(t *testing.T) {
	themes := AvailableThemes()
	if len(themes) < 5 {
		t.Errorf("got %d themes, want at least 5", len(themes))
	}

	for _, name := range themes {
		SetCurrentTheme(name)

		if CurrentTheme() != name {
			t.Errorf("CurrentTheme() = %q, want %q", CurrentTheme(), name)
		}

		for _, r := range Roles() {
			s := Paint(r).Render("orbit")
			if s == "" {
				t.Errorf("theme %s role %d painted empty string", name, r)
			}
		}
	}

	SetCurrentTheme("monokai")
}

// The window is painted on its theme's own paper, and changing the theme
// changes it: a background written down here rather than read from the shell
// would be the one colour on screen that does not move with the setting.
func TestTheWindowIsPaintedOnItsThemesPaper(t *testing.T) {
	t.Cleanup(func() { SetCurrentTheme("frauddi") })

	for _, tc := range []struct{ theme, want string }{
		{"frauddi", "#0B1016"},
		{"dracula", "#282A36"},
	} {
		SetCurrentTheme(tc.theme)

		if got := WindowBackground(); got != lipgloss.Color(tc.want) {
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

	ink, _, ok := strings.Cut(Chrome().Render("probe"), "probe")
	if !ok || ink == "" {
		t.Fatalf("Chrome() rendered %q, want a colour before the text", Chrome().Render("probe"))
	}

	faint, _, _ := strings.Cut(Paint(Dim).Render("probe"), "probe")

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
