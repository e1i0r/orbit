package ui

// Seven roles, and every one of them has to be buildable with no terminal in
// the room: this suite never opens one, and a style that had to ask the
// terminal what colour its background is would be a style no test could
// build. The table is also where "weight before colour" is checked, because
// bold and faint are the whole of the hierarchy on a NO_COLOR terminal, a
// mono ssh session, and --once piped into a file.

import (
	"image/color"
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
