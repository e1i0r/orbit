package verb

// The rest of the settings table, through the verb a person actually types.
//
// Each of these is one line of what a reader sees when they run `orbit
// settings`, and the table is the only place several of them are ever shown:
// a value written down and read back as something else is a setting somebody
// believes they changed.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// TestNoCapIsHowTheUnreadCapIsTurnedOff, which is why zero has to be
// accepted: it is the number that means there is no cap, and a reader who
// cannot type it has no way to turn the thing off.
func TestNoCapIsHowTheUnreadCapIsTurnedOff(t *testing.T) {
	w := worldOf(t)

	if out := set(t, w, "unread-cap", "0"); !strings.Contains(out.Said, "0") {
		t.Errorf("an unread cap of none answered %q", out.Said)
	}

	if out := set(t, w, "unread-cap", "3"); !strings.Contains(out.Said, "3") {
		t.Errorf("an unread cap of three answered %q", out.Said)
	}

	for _, value := range []string{"-1", "many"} {
		mustRefuse(t, w, "settings set", In{
			Args: map[string]string{"key": "unread-cap", "value": value}, By: "operator",
		})
	}
}

// TestTheDefaultFlowHasToBeAFlowNameSomebodyCouldSave, because the value is
// a filename before it is anything else: a name with a slash in it would be
// a default flow that looks for a file outside the directory flows live in.
func TestTheDefaultFlowHasToBeAFlowNameSomebodyCouldSave(t *testing.T) {
	w := worldOf(t)

	if out := set(t, w, "flow", "review"); !strings.Contains(out.Said, "review") {
		t.Errorf("setting the default flow answered %q", out.Said)
	}

	table := mustAsk(t, w, "settings", In{By: "operator"})
	if !strings.Contains(table.Said, "review") {
		t.Errorf("the table does not carry the flow that was set:\n%s", table.Said)
	}

	for _, value := range []string{"../escape", "a/b", ""} {
		mustRefuse(t, w, "settings set", In{
			Args: map[string]string{"key": "flow", "value": value}, By: "operator",
		})
	}
}

// TestTheSwitchesAreWrittenDownAndReadBack.
//
// Each of these is a yes or a no that changes what every command does, and
// the table is where a reader checks which way it is set. A verb that took
// the value and answered without writing it would leave them reading their
// own instruction back.
func TestTheSwitchesAreWrittenDownAndReadBack(t *testing.T) {
	for _, key := range []string{"check-record", "notify", "autopilot"} {
		w := worldOf(t)

		on := set(t, w, key, "on")
		if !strings.Contains(on.Said, "on") {
			t.Errorf("turning %s on answered %q", key, on.Said)
		}

		if table := mustAsk(t, w, "settings", In{By: "operator"}); !rowSays(table.Said, key, "on") {
			t.Errorf("the table's %s row does not read on:\n%s", key, table.Said)
		}

		off := set(t, w, key, "off")
		if !strings.Contains(off.Said, "off") {
			t.Errorf("turning %s off answered %q", key, off.Said)
		}

		if table := mustAsk(t, w, "settings", In{By: "operator"}); !rowSays(table.Said, key, "off") {
			t.Errorf("the table's %s row does not read off:\n%s", key, table.Said)
		}
	}
}

// TestTheThemeRowNamesTheOneTheWindowWillDraw.
//
// Nothing chosen is not nothing drawn: the window has a theme it falls back
// to, and a table printing an empty cell there tells a reader their cockpit
// has no theme when it plainly has one.
func TestTheThemeRowNamesTheOneTheWindowWillDraw(t *testing.T) {
	w := worldOf(t)

	fresh := mustAsk(t, w, "settings", In{By: "operator"})
	if !rowSays(fresh.Said, "theme", theme.DefaultTheme) {
		t.Errorf("a table nobody has chosen a theme in reads:\n%s", fresh.Said)
	}

	set(t, w, "theme", "monokai")

	chosen := mustAsk(t, w, "settings", In{By: "operator"})
	if !rowSays(chosen.Said, "theme", "monokai") {
		t.Errorf("the theme that was chosen is not in the table:\n%s", chosen.Said)
	}

	if rowSays(chosen.Said, "theme", theme.DefaultTheme) {
		t.Errorf("the table names the default beside the theme that replaced it:\n%s", chosen.Said)
	}
}
