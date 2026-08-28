package ui

// settings_screen_coverage_test.go is the settings screen's own behaviour:
// what a key does to it, and what applySetting does with every key name the
// screen can send it.

import (
	"io"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestApplySettingEveryKey(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. language returns a languageMsg command.
	next, cmd := m.applySetting("language", "es")

	m = asModel(t, next)
	if cmd == nil {
		t.Fatal("applySetting(language) returned no command")
	}

	if lm, ok := cmd().(languageMsg); !ok || lm.Lang != "es" {
		t.Errorf("applySetting(language) command = %#v, want languageMsg{Lang: es}", cmd())
	}

	// 2. autopilot on, then off.
	next, _ = m.applySetting("autopilot", "on")

	m = asModel(t, next)
	if !m.opts.Settings.Autopilot() {
		t.Error("applySetting(autopilot, on) left autopilot off")
	}

	next, _ = m.applySetting("autopilot", "off")

	m = asModel(t, next)
	if m.opts.Settings.Autopilot() {
		t.Error("applySetting(autopilot, off) left autopilot on")
	}

	// 3. unread-cap: a number is taken, a non-number is silently ignored —
	// both still narrate the same way, since applySetting's sentence names
	// what was typed rather than what stuck.
	next, _ = m.applySetting("unread-cap", "7")
	m = asModel(t, next)
	wantBand(t, m, "unread-cap is now 7")
	next, _ = m.applySetting("unread-cap", "not-a-number")
	m = asModel(t, next)
	wantBand(t, m, "unread-cap is now not-a-number")

	// 4. engine, with an effort that no longer validates for it, resets the
	// effort to the new engine's first option; an effort that still
	// validates is left alone. The fixture's settings port is a stub whose
	// SetModel/SetEngine do not persist, so the model half of the reset is
	// exercised without being independently observable — knobs.Effort is a
	// real field on Model and is.
	m.knobs.Effort = "xhigh" // not valid for codex
	next, _ = m.applySetting("engine", "codex")

	m = asModel(t, next)
	if m.knobs.Effort != "default" {
		t.Errorf("applySetting(engine, codex) left effort %q, want it reset to default", m.knobs.Effort)
	}

	m.knobs.Effort = "low" // valid for every engine
	next, _ = m.applySetting("engine", "claude")

	m = asModel(t, next)
	if m.knobs.Effort != "low" {
		t.Errorf("applySetting(engine, claude) changed a still-valid effort to %q", m.knobs.Effort)
	}

	// 5. model, effort and thinking.
	next, _ = m.applySetting("model", "opus")
	m = asModel(t, next)
	wantBand(t, m, "model is now opus")
	next, _ = m.applySetting("effort", "low")

	m = asModel(t, next)
	if m.knobs.Effort != "low" {
		t.Errorf("applySetting(effort) = %q, want low", m.knobs.Effort)
	}

	next, _ = m.applySetting("thinking", "off")

	m = asModel(t, next)
	if m.knobs.Thinking != "off" {
		t.Errorf("applySetting(thinking) = %q, want off", m.knobs.Thinking)
	}

	// 6. flow and theme, the theme also switching the live palette — a real
	// global the stub cannot intercept.
	next, _ = m.applySetting("flow", "careful")
	m = asModel(t, next)
	wantBand(t, m, "flow is now careful")

	old := CurrentTheme()

	t.Cleanup(func() { SetCurrentTheme(old) })

	next, _ = m.applySetting("theme", "nord")

	m = asModel(t, next)
	if CurrentTheme() != "nord" {
		t.Errorf("applySetting(theme) left the live theme %q, want nord", CurrentTheme())
	}

	// 7. a settings port that is nil, and a Do port that is not: the
	// switch is skipped, but the palette command still runs.
	m.opts.Settings = nil

	var ranArgs []string

	m.opts.Do = func(_ string, args []string, _ io.Writer) error {
		ranArgs = args
		return nil
	}
	next, _ = m.applySetting("unread-cap", "3")

	m = asModel(t, next)
	if len(ranArgs) != 2 || ranArgs[0] != "unread-cap" || ranArgs[1] != "3" {
		t.Errorf("applySetting with a nil settings port ran Do with args %v, want [unread-cap 3]", ranArgs)
	}
}

func TestSettingsKeyEditingAndNavigation(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenSettings

	// 1. Navigating with j/k and arrow keys wraps at both ends.
	rows := m.settingRowsList()
	m.settings.sel = 0
	next, _ := m.settingsKey(tea.KeyPressMsg{Code: 'k', Text: "k"})

	m = asModel(t, next)
	if m.settings.sel != len(rows)-1 {
		t.Errorf("up from row 0 wrapped to %d, want %d", m.settings.sel, len(rows)-1)
	}

	next, _ = m.settingsKey(tea.KeyPressMsg{Code: 'j', Text: "j"})

	m = asModel(t, next)
	if m.settings.sel != 0 {
		t.Errorf("down from the last row wrapped to %d, want 0", m.settings.sel)
	}

	// 2. 'e' enters editing mode with the row's current value typed in.
	next, _ = m.settingsKey(tea.KeyPressMsg{Code: 'e', Text: "e"})

	m = asModel(t, next)
	if !m.settings.editing || m.settings.typed != rows[0].val {
		t.Errorf("editing 'e' = %+v, want editing with typed=%q", m.settings, rows[0].val)
	}

	// 3. Typing appends, backspace trims, an unmatched control key does
	// nothing, esc cancels.
	next, _ = m.settingsKey(tea.KeyPressMsg{Code: 'x', Text: "x"})

	m = asModel(t, next)
	if m.settings.typed != rows[0].val+"x" {
		t.Errorf("typed = %q, want the row's value with x appended", m.settings.typed)
	}

	next, _ = m.settingsKey(tea.KeyPressMsg{Code: tea.KeyBackspace})

	m = asModel(t, next)
	if m.settings.typed != rows[0].val {
		t.Errorf("backspace left %q, want the row's original value", m.settings.typed)
	}

	next, _ = m.settingsKey(tea.KeyPressMsg{Code: tea.KeyUp})

	m = asModel(t, next)
	if !m.settings.editing {
		t.Error("an unmatched key while editing left editing mode")
	}

	next, _ = m.settingsKey(tea.KeyPressMsg{Code: tea.KeyEscape})

	m = asModel(t, next)
	if m.settings.editing || m.settings.typed != "" {
		t.Errorf("esc while editing left %+v, want editing cleared", m.settings)
	}

	// 4. Enter submits a typed value; left/right cycle an option.
	m.settings.sel = 0
	m.settings.editing, m.settings.typed = true, "es"
	next, _ = m.settingsKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	m = asModel(t, next)
	if m.settings.editing {
		t.Error("enter did not leave editing mode")
	}

	before := m.settingRowsList()[0].val
	next, _ = m.settingsKey(tea.KeyPressMsg{Code: tea.KeyRight})

	m = asModel(t, next)
	if m.settingRowsList()[0].val == before {
		t.Error("right did not cycle the first row's option")
	}

	next, _ = m.settingsKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = asModel(t, next)

	// 5. Back and quit both abandon the screen.
	next, _ = m.settingsKey(tea.KeyPressMsg{Code: tea.KeyEscape})

	m = asModel(t, next)
	if m.screen != screenList {
		t.Errorf("esc from settings left screen %v, want screenList", m.screen)
	}
}

func TestSettingsKeyWithoutASettingsPort(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen, m.opts.Settings = screenSettings, nil

	// With no settings file there are no rows, so only back/quit do
	// anything; every other key is a no-op rather than a panic.
	next, _ := m.settingsKey(tea.KeyPressMsg{Code: 'j', Text: "j"})
	m = asModel(t, next)
	next, _ = m.settingsKey(tea.KeyPressMsg{Code: tea.KeyEscape})

	m = asModel(t, next)
	if m.screen != screenList {
		t.Errorf("esc with no settings rows left screen %v, want screenList", m.screen)
	}
}

func TestCycleSettingOutOfRangeIsANoOp(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.settings.sel = -1
	next, cmd := m.cycleSetting(1)

	got := asModel(t, next)
	if cmd != nil || got.settings.sel != -1 {
		t.Errorf("cycleSetting with sel out of range mutated the model: %+v", got.settings)
	}
}

func TestSettingsSubmitOutOfRangeClearsEditing(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.settings.sel, m.settings.editing = 999, true
	next, _ := m.settingsSubmit()

	got := asModel(t, next)
	if got.settings.editing {
		t.Error("settingsSubmit with sel out of range left editing on")
	}
}

func TestPadRightBothBranches(t *testing.T) {
	if got := padRight("short", 10); got != "short     " {
		t.Errorf("padRight(short, 10) = %q, want it padded to 10 cells", got)
	}

	if got := padRight("already-long-enough", 4); got != "already-long-enough" {
		t.Errorf("padRight already past width = %q, want it left alone", got)
	}
}
