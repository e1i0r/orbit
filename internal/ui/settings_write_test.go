package ui

// settings_write_test.go is what happens when a setting cannot be written:
// the file refuses, or the value is not one this screen may put in it.

import (
	"errors"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// TestASettingThatCouldNotBeWrittenSaysSoInsteadOfSayingItIsSet. Discarded
// with `_ =`, these writes leave the band saying "theme is now nord"
// whatever comes back — the screen draws a setting that is not in the file,
// and the next time it is opened the switch has flipped itself back. The
// settings file has a lock, so refusing is not hypothetical: a second orbit
// holding it makes every one of these fail.
func TestASettingThatCouldNotBeWrittenSaysSoInsteadOfSayingItIsSet(t *testing.T) {
	for _, c := range []struct{ key, val string }{
		{"language", "es"},
		{"autopilot", "on"},
		{"unread-cap", "7"},
		{"engine", "codex"},
		{"model", "opus"},
		{"flow", "careful"},
		{"theme", "nord"},
	} {
		m, _ := testModel(t, 100, 30)
		m.opts.Settings = &settingsFile{fail: errors.New("settings file is locked by another orbit")}

		old := theme.CurrentTheme()

		t.Cleanup(func() { theme.SetCurrentTheme(old) })

		next, _ := m.applySetting(c.key, c.val)

		got := asModel(t, next)
		wantBand(t, got, "locked by another orbit")

		if strings.Contains(got.message, "is now") {
			t.Errorf("applySetting(%s) reported the write as done: %q", c.key, got.message)
		}
	}
}

// TestARefusedThemeDoesNotRepaintTheWindow. The palette is a global, so a
// theme written into it after a refused write is one the settings file does
// not have and the next run will not restore.
func TestARefusedThemeDoesNotRepaintTheWindow(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Settings = &settingsFile{fail: errors.New("disk full")}

	old := theme.CurrentTheme()

	t.Cleanup(func() { theme.SetCurrentTheme(old) })

	if _, _ = m.applySetting("theme", "nord"); theme.CurrentTheme() != old {
		t.Errorf("a refused theme write repainted the window to %q", theme.CurrentTheme())
	}
}

// TestAFlowNameThatIsAPathIsRefused. `orbit set` has always checked this and
// this screen never did, so what the command line would not take, the window
// typed into the same file.
func TestAFlowNameThatIsAPathIsRefused(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	next, _ := m.applySetting("flow", "../../etc/passwd")

	got := asModel(t, next)
	if strings.Contains(got.message, "is now") {
		t.Errorf("applySetting(flow) took a path as a flow name: %q", got.message)
	}
}
