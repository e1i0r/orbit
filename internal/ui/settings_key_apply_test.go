package ui

// settings_screen_coverage_test.go is the settings screen's own behaviour:
// what a key does to it, and what applySetting does with every key name the
// screen can send it.

import (
	"io"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestTheSettingsDialsAreTheEnginesOwn. The three dials were a switch on the
// engine's name written in this package, and it offered opencode a model
// called gemini-2.5-pro, which opencode has never had. A made-up engine is
// what proves the table is the port's: nothing in this package could have
// guessed it.
func TestTheSettingsDialsAreTheEnginesOwn(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Engines = func() []EngineInfo {
		return []EngineInfo{{
			Name:      "zeta",
			Available: true,
			Models:    []ChoiceInfo{{ID: "", Label: "default"}, {ID: "zeta/one", Label: "one"}},
			Efforts:   []ChoiceInfo{{ID: "", Label: "default"}, {ID: "brisk", Label: "brisk"}},
		}}
	}

	if err := m.opts.Settings.SetEngine("zeta"); err != nil {
		t.Fatalf("choose the engine: %v", err)
	}

	rows := map[string]settingRow{}
	for _, r := range m.settingRowsList() {
		rows[r.key] = r
	}

	if got := rows["engine"].options; !slices.Equal(got, []string{"zeta"}) {
		t.Errorf("the engine dial offers %v, want only what the port answered", got)
	}

	// The id is what the setting holds and the label is what the reader
	// picks from. They are two strings for opencode — the id is
	// provider-qualified — and drawing the id would put the provider in
	// front of every position on the dial.
	model := rows["model"]
	if !slices.Equal(model.options, []string{"zeta/one"}) {
		t.Errorf("the model dial holds %v, want zeta's own ids", model.options)
	}

	if got := model.label(0); got != "one" {
		t.Errorf("the model dial draws %q, want the label the port gave it", got)
	}

	if got := rows["effort"].options; !slices.Equal(got, []string{"brisk"}) {
		t.Errorf("the effort dial offers %v, want zeta's own", got)
	}
}

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

	// 3. unread-cap is the one setting a number is typed into. A number is
	// taken; what is not a number is refused in words. It used to be
	// dropped on the floor under a band saying "unread-cap is now
	// not-a-number", which named a cap that was nowhere.
	next, _ = m.applySetting("unread-cap", "7")
	m = asModel(t, next)
	wantBand(t, m, "unread-cap is now 7")
	next, _ = m.applySetting("unread-cap", "not-a-number")
	m = asModel(t, next)
	wantBand(t, m, "not a whole number")

	// A negative cap is no cap at all to every reader of it, so setting one
	// turned the brake off while looking like it set one.
	next, _ = m.applySetting("unread-cap", "-1")
	m = asModel(t, next)
	wantBand(t, m, "cannot be negative")

	// 4. engine, with an effort that no longer validates for it, resets the
	// effort to the new engine's first option; an effort that still
	// validates is left alone. The fixture's settings port is a stub whose
	// SetModel/SetEngine do not persist, so the model half of the reset is
	// exercised without being independently observable — knobs.Effort is a
	// real field on Model and is.
	// "default" is not among them, and deliberately: the engines answer
	// that choice with an empty id, and an effort of the literal word
	// "default" is one internal/task refuses before a run starts.
	m.knobs.Effort = "xhigh" // not valid for codex
	next, _ = m.applySetting("engine", "codex")

	m = asModel(t, next)
	if m.knobs.Effort != "low" {
		t.Errorf("applySetting(engine, codex) left effort %q, want it reset to codex's first", m.knobs.Effort)
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

	// 7. a settings port that is nil: there is nowhere to write, and the
	// screen says what it would have said rather than falling over.
	//
	// Nothing is shelled out to either. Every setting used to be written
	// twice — once through this port and once by running `orbit set` — so
	// the file was opened, locked and rewritten twice per keystroke, and
	// whatever the second one had to say was thrown away.
	m.opts.Settings = nil

	ran := false
	m.opts.Do = func(string, []string, io.Writer) error {
		ran = true
		return nil
	}
	next, _ = m.applySetting("unread-cap", "3")

	m = asModel(t, next)
	wantBand(t, m, "unread-cap is now 3")

	if ran {
		t.Error("applySetting shelled out to `orbit set` as well as writing the setting itself")
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
