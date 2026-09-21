package store

// The settings file, which is the one file here a person edits by hand.
//
// It is JSON on disk and a person with an editor is free to write anything
// in it — a trailing comma, a string where a number goes, half a file after
// a machine went down. None of that may stop a command: a reader always has
// something usable, because the alternative is a cockpit that will not open
// until somebody finds the typo.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"
)

// FuzzSettingsAlwaysAnswerSomethingUsable.
func FuzzSettingsAlwaysAnswerSomethingUsable(f *testing.F) {
	for _, seed := range []string{
		"", "{}", `{"engine":"claude"}`, `{"budget_task":1.5}`,
		`{"budget_task":"a lot"}`, "{", "null", "[]", `{"unread_cap":-1}`,
		`{"autopilot":true,"theme":"frauddi"}`, "\x00", `{"engine":`,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, body string) {
		root := t.TempDir()

		s, err := New(root)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte(body), 0o600); err != nil {
			t.Fatalf("write the settings: %v", err)
		}

		cfg, err := s.Settings()
		if err != nil {
			t.Fatalf("a settings file of %q stopped a command: %v", body, err)
		}

		// Whatever came back is a configuration this can write again, which
		// is what every setter depends on: they are all read-modify-write,
		// and one that could not write what it read would lose the rest.
		if _, err := json.Marshal(cfg); err != nil {
			t.Errorf("a settings file of %q answered something unwritable: %v", body, err)
		}

		// The words in it reach a terminal and a browser.
		for what, value := range map[string]string{
			"engine": cfg.Engine, "model": cfg.Model, "theme": cfg.Theme,
			"flow": cfg.Flow, "language": cfg.Language,
		} {
			if !utf8.ValidString(value) {
				t.Errorf("a settings file of %q answered a %s that is not text: %q", body, what, value)
			}
		}
	})
}

// FuzzSettingsSurviveBeingSaved.
//
// Every setter is a read-modify-write, and a file that will not parse is
// moved aside rather than overwritten — so a reader who flips one switch
// keeps the engine, model and theme they chose. What this asks is the half
// after that: whatever is written comes back as what was written.
func FuzzSettingsSurviveBeingSaved(f *testing.F) {
	f.Add("claude", "opus", 1.25, 3, true)
	f.Add("", "", 0.0, 0, false)
	f.Add("codex", "gpt", -1.0, -1, true)
	f.Add("a name with spaces", "ñ", 0.1, 99, false)

	f.Fuzz(func(t *testing.T, engine, model string, budget float64, cap int, autopilot bool) {
		if !utf8.ValidString(engine) || !utf8.ValidString(model) {
			t.Skip()
		}

		// A budget that is not a number at all cannot be written as JSON,
		// and nothing upstream can make one: the settings table refuses it
		// where it is typed.
		if budget != budget || budget > 1e308 || budget < -1e308 {
			t.Skip()
		}

		root := t.TempDir()

		s, err := New(root)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		want := Settings{
			Engine: engine, Model: model,
			BudgetTask: budget, UnreadCap: cap, Autopilot: autopilot,
		}

		if err := s.SaveSettings(want); err != nil {
			t.Fatalf("SaveSettings(%+v): %v", want, err)
		}

		got, err := s.Settings()
		if err != nil {
			t.Fatalf("read them back: %v", err)
		}

		if got.Engine != want.Engine || got.Model != want.Model {
			t.Errorf("saved %q/%q and read back %q/%q", want.Engine, want.Model, got.Engine, got.Model)
		}

		if got.BudgetTask != want.BudgetTask || got.UnreadCap != want.UnreadCap {
			t.Errorf("saved %v/%d and read back %v/%d",
				want.BudgetTask, want.UnreadCap, got.BudgetTask, got.UnreadCap)
		}

		if got.Autopilot != want.Autopilot {
			t.Errorf("saved autopilot=%v and read back %v", want.Autopilot, got.Autopilot)
		}
	})
}
