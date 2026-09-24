package ui

// The one claim this whole change exists to keep: every setting Orbit
// declares is a row somebody can see and turn.

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/verb"
)

// TestEverySettingDeclaredIsASettingDrawn.
//
// Thirteen settings were declared and nine rows were drawn, and two of those
// nine were not settings at all — so the window showed seven of thirteen.
// Among the six missing were whether Orbit may interrupt you and which
// account may command it over a chat: decisions somebody takes by looking at
// a screen, not by remembering a key to type at a terminal.
//
// It happened because the table was written out by hand in one package and
// the settings were declared in another, and nothing failed when the two
// stopped agreeing. This is what fails now.
func TestEverySettingDeclaredIsASettingDrawn(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = unfolded(m.openSettings())

	drawn := map[string]bool{}
	for _, r := range m.settingRowsList() {
		drawn[r.Key] = true
	}

	for _, one := range verb.Shipped(m.opts.Words) {
		if !drawn[one.Name] {
			t.Errorf("%s is a setting and the window draws no row for it", one.Name)
		}
	}
}

// TestEveryRowDrawnIsASettingOrADial. The other direction, which is what
// keeps this from being satisfied by drawing rows that write nowhere.
//
// Two rows are neither: effort and thinking are not in the settings file at
// all, and the window is what keeps them. They are named here so that a
// third one appearing has to be named too.
func TestEveryRowDrawnIsASettingOrADial(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = unfolded(m.openSettings())

	declared := map[string]bool{}
	for _, one := range verb.Shipped(m.opts.Words) {
		declared[one.Name] = true
	}

	theWindows := []string{"effort", "thinking"}

	for _, r := range m.settingRowsList() {
		if declared[r.Key] || slices.Contains(theWindows, r.Key) {
			continue
		}

		t.Errorf("the window draws a row called %q, and nothing declares it", r.Key)
	}
}

// TestEverySettingDrawnSaysWhatItIs. A row with no sentence under it is a
// name and a switch, and a name and a switch is what a settings file already
// was — the screen is only worth opening if it explains itself.
func TestEverySettingDrawnSaysWhatItIs(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = unfolded(m.openSettings())

	for _, r := range m.settingRowsList() {
		if r.About == "" {
			t.Errorf("the %s row says nothing about what it is", r.Key)
		}
	}
}

// TestEverySettingDrawnCanBeReached. A row the cursor cannot get to is a row
// that is not there, whatever the table says — and with thirteen settings
// the table is twice the height of the screen.
func TestEverySettingDrawnCanBeReached(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = unfolded(m.openSettings())

	rows := m.settingRowsList()
	seen := map[string]bool{}

	// Twice the rows, because every group's heading is a stop on the way.
	for range 2 * len(rows) {
		seen[rows[m.settings.Chosen()].Key] = true
		m = m.wheelSettings(1)

		if m.settings.OnHeading() != "" {
			continue
		}

		// Every row the cursor lands on is drawn, which is the half that
		// the count alone cannot check.
		whole := m.settingsRows(m.frame.Body.H, m.frame.Body.W)
		if !onScreen(whole, rows[m.settings.Chosen()].Key) {
			t.Errorf("the cursor is on %s and the screen does not draw it",
				rows[m.settings.Chosen()].Key)
		}
	}

	for _, r := range rows {
		if !seen[r.Key] {
			t.Errorf("the cursor never reached %s", r.Key)
		}
	}
}

// onScreen is whether a name is anywhere on the drawn screen, read past the
// colours the rows are painted in.
func onScreen(rows []string, name string) bool {
	for _, line := range rows {
		if strings.Contains(ansi.Strip(line), name) {
			return true
		}
	}

	return false
}

// TestARowWithNoDialIsWrittenInto. Four of the thirteen offer nothing to
// choose from — a chat id, two budgets and a percentage — and the click that
// turns every other row has nothing to turn on these. It opens the line
// instead, which is the only gesture that changes one at all.
func TestARowWithNoDialIsWrittenInto(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = unfolded(m.openSettings())

	rows := m.settingRowsList()

	at := -1

	for i, r := range rows {
		if len(r.Options) == 0 {
			at = i
			break
		}
	}

	if at < 0 {
		t.Fatal("no row on this screen is written into, so there is nothing to check")
	}

	next, _ := m.leftClick(point.Target{Kind: point.SettingsRow, Pane: at})

	got := asModel(t, next)
	if !got.settings.Editing() {
		t.Errorf("clicking %s did not open the line", rows[at].Key)
	}

	if got.settings.Typed() != rows[at].Val {
		t.Errorf("the line opened with %q, want what the row holds", got.settings.Typed())
	}
}

// TestARowWithNoDialShowsWhatItHolds. It draws no pills, so without this it
// drew a name and an empty space — which reads as a setting that is broken
// rather than one that is typed into.
func TestARowWithNoDialShowsWhatItHolds(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = unfolded(m.openSettings())

	held := m.opts.Settings.Choose
	if _, err := held(m.opts.Words, "chat-id", "7912204269"); err != nil {
		t.Fatalf("the fixture refused a chat id: %v", err)
	}

	// To the chat id's own row, and not to the end of the table: which
	// setting is last is a fact about the vocabulary, and this test is
	// about a row that has no pills.
	at := -1

	for i, r := range m.settingRowsList() {
		if r.Key == "chat-id" {
			at = i

			break
		}
	}

	if at < 0 {
		t.Fatal("there is no chat id row")
	}

	for m.settings.Chosen() < at {
		m = m.wheelSettings(1)
	}

	if !onScreen(m.settingsRows(m.frame.Body.H, m.frame.Body.W), "7912204269") {
		t.Error("the chat id is set and the row does not show it")
	}
}
