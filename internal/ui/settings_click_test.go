package ui

// A click on the settings table turns the dial of the setting drawn under
// the pointer, and of no other.

import (
	"maps"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/settings"
)

// TestASettingIsTurnedWhereItIsDrawn, at every scroll and on every row of
// the body.
//
// The window counted the row for itself: the line, less the four rows of
// the title, plus the offset, over three lines a setting. What it did not
// know is that the table stops short of the floor, where the line saying
// how to leave is drawn — so a click there named the setting just under
// the window, and at the columns the pills are drawn in it turned that
// setting's dial. Changing a setting you cannot see is the worst of the
// three screens this arithmetic was copied to.
func TestASettingIsTurnedWhereItIsDrawn(t *testing.T) {
	m, _ := testModel(t, 100, 24)
	m = unfolded(m.openSettings())

	list := m.settingRowsList()
	if len(list) == 0 {
		t.Fatal("the fixture window has no settings")
	}

	for _, notch := range []int{0, 1, 4, 40} {
		at := m
		for range notch {
			at = at.wheelSettings(1)
		}

		h, w := at.frame.Body.H, at.frame.Body.W
		drawn := ansi.Strip(strings.Join(at.settingsRows(h, w), "\n"))

		// The ways out are drawn under the table, so the last row of the
		// body belongs to no setting.
		if got := at.hitSettings(5, at.frame.Body.Y+h-1); got.Kind != point.None {
			t.Errorf("wheel %d: a click on the ways out answers row %d", notch, got.Pane)
		}

		for line := range h {
			// A heading folds its group and turns nothing.
			got := at.hitSettings(settings.PillsAt+1, at.frame.Body.Y+line)
			if got.Kind != point.SettingsRow || got.Key == settingsFold {
				continue
			}

			if got.Pane < 0 || got.Pane >= len(list) {
				t.Fatalf("wheel %d, row %d answers setting %d of %d", notch, line, got.Pane, len(list))
			}

			// The name and not the sentence under it: a sentence is cut
			// to the window and would read as missing when it is only
			// short.
			if key := list[got.Pane].Key; !strings.Contains(drawn, key) {
				t.Errorf("wheel %d, row %d turns %q, which is not on the screen:\n%s",
					notch, line, key, drawn)
			}
		}
	}
}

// TestAClickUnderASettingsNameChangesNothing. A row is three lines and only
// the first carries the dial; the description and the blank under it
// answered the pills' columns too, so a click on "whether a run walks its
// whole flow" turned autopilot off. There the click only moves the cursor.
func TestAClickUnderASettingsNameChangesNothing(t *testing.T) {
	m, _ := testModel(t, 100, 40)
	m = unfolded(m.openSettings())

	values := func(m Model) map[string]string {
		held := map[string]string{}
		for _, r := range m.settingRowsList() {
			held[r.Key] = r.Val
		}

		return held
	}

	before := values(m)

	for i, r := range m.settingRowsList() {
		name, shown := m.settings.LineOf(i, m.settingsEnv())
		if !shown {
			continue
		}

		for _, under := range []int{1, 2} {
			for x := 2; x < m.frame.Body.W; x += 3 {
				hit := m.hitSettings(x, m.frame.Body.Y+name+under)
				if hit.Kind != point.SettingsRow {
					continue
				}

				after := clicked(t, m, hit)
				if got := values(after); !maps.Equal(got, before) {
					t.Fatalf("a click %d lines under %q at column %d changed the settings: %v, want %v",
						under, r.Key, x, got, before)
				}

				if after.settings.Chosen() != i {
					t.Errorf("a click under %q put the cursor on row %d, want %d",
						r.Key, after.settings.Chosen(), i)
				}
			}
		}
	}
}

// TestAClickOnAGroupsHeadingFoldsIt. The screen opens with every group
// folded; a click on a heading opens its group, and a second click folds
// it again.
func TestAClickOnAGroupsHeadingFoldsIt(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openSettings()

	first := m.settingRowsList()[0]
	if _, shown := m.settings.LineOf(0, m.settingsEnv()); shown {
		t.Fatalf("%s is drawn on a screen that opens folded", first.Key)
	}

	var hit point.Target

	for line := range m.frame.Body.H {
		got := m.hitSettings(4, m.frame.Body.Y+line)
		if got.Key == settingsFold && got.Field == first.Group {
			hit = got

			break
		}
	}

	if hit.Key != settingsFold {
		t.Fatalf("no line of the screen is the %s heading", first.Group)
	}

	opened := clicked(t, m, hit)
	if _, shown := opened.settings.LineOf(0, opened.settingsEnv()); !shown {
		t.Errorf("after a click on %s, %s is still hidden", first.Group, first.Key)
	}

	folded := clicked(t, opened, hit)
	if _, shown := folded.settings.LineOf(0, folded.settingsEnv()); shown {
		t.Errorf("a second click on %s left %s drawn", first.Group, first.Key)
	}
}
