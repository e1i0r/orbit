package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

// settings reads back what `orbit set` wrote, through the store rather than
// by parsing the JSON here: the round trip is the thing worth asserting, and
// a second parser in a test is a second opinion about the format.
func settings(t *testing.T, orbitHome string) store.Settings {
	t.Helper()
	t.Setenv("ORBIT_HOME", orbitHome)

	s, err := store.Open()
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}

	cfg, err := s.Settings()
	if err != nil {
		t.Fatalf("read the settings: %v", err)
	}

	return cfg
}

func TestSetTurnsAutopilotOnAndSaysWhatItNowIs(t *testing.T) {
	_, orbitHome := workspace(t)

	code, out, errOut := run(t, "settings", "set", "autopilot", "on")
	if code != 0 {
		t.Fatalf("set autopilot on exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "autopilot is now on") {
		t.Errorf("set said %q, which does not say what the setting now is", out)
	}

	if cfg := settings(t, orbitHome); !cfg.Autopilot {
		t.Errorf("autopilot is %v on disk, want true", cfg.Autopilot)
	}
}

// A confirmation that repeated the argument would say "autopilot is now 1",
// which is not what the file holds.
func TestSetSpeaksTheSwitchInTheWordsTheWindowUses(t *testing.T) {
	_, orbitHome := workspace(t)
	if code, out, errOut := run(t, "settings", "set", "autopilot", "1"); code != 0 {
		t.Fatalf("set autopilot 1 exited %d: %s", code, errOut)
	} else if !strings.Contains(out, "autopilot is now on") {
		t.Errorf("set said %q, want it to say on", out)
	}

	if cfg := settings(t, orbitHome); !cfg.Autopilot {
		t.Errorf("autopilot is %v on disk, want true", cfg.Autopilot)
	}

	if code, out, errOut := run(t, "settings", "set", "autopilot", "off"); code != 0 {
		t.Fatalf("set autopilot off exited %d: %s", code, errOut)
	} else if !strings.Contains(out, "autopilot is now off") {
		t.Errorf("set said %q, want it to say off", out)
	}

	if cfg := settings(t, orbitHome); cfg.Autopilot {
		t.Errorf("autopilot is %v on disk, want false", cfg.Autopilot)
	}
}

// Setting one line must not blank the others. `orbit set` reads the file,
// changes one field and writes it back for exactly this reason.
func TestSetLeavesEverySettingItWasNotAskedAboutAlone(t *testing.T) {
	_, orbitHome := workspace(t)
	for _, args := range [][]string{
		{"settings", "set", "unread-cap", "9"},
		{"settings", "set", "language", "es"},
		{"settings", "set", "autopilot", "on"},
		{"settings", "set", "model", "sonnet"},
	} {
		if code, _, errOut := run(t, args...); code != 0 {
			t.Fatalf("%v exited %d: %s", args, code, errOut)
		}
	}

	cfg := settings(t, orbitHome)
	if cfg.UnreadCap != 9 || cfg.Language != "es" || !cfg.Autopilot || cfg.Model != "sonnet" {
		t.Errorf("the settings are %+v, and one of them was written over", cfg)
	}
}

// Zero is a setting somebody chose — no cap — and it has to survive the
// round trip through an omitempty field, which is what defaultUnreadCap
// exists to keep separate from never having chosen at all.
func TestSetCanTurnTheUnreadCapOff(t *testing.T) {
	_, orbitHome := workspace(t)
	if code, _, errOut := run(t, "settings", "set", "unread-cap", "0"); code != 0 {
		t.Fatalf("set unread-cap 0 exited %d: %s", code, errOut)
	}

	if cfg := settings(t, orbitHome); cfg.UnreadCap != 0 {
		t.Errorf("the unread cap is %d, want 0", cfg.UnreadCap)
	}
}

// The default flow is a setting like any other, and `orbit new` with no
// -flow is what reads it.
func TestSetChoosesTheFlowANewTaskIsWrittenAgainst(t *testing.T) {
	_, orbitHome := workspace(t)
	if code, out, errOut := run(t, "settings", "set", "flow", "careful"); code != 0 {
		t.Fatalf("set flow careful exited %d: %s", code, errOut)
	} else if !strings.Contains(out, "flow is now careful") {
		t.Errorf("set said %q, which does not say what the setting now is", out)
	}

	if cfg := settings(t, orbitHome); cfg.Flow != "careful" {
		t.Errorf("the default flow is %q on disk, want careful", cfg.Flow)
	}
}

func TestSetRefusesWhatItCannotDoAndSaysWhy(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		says string
	}{
		{"a key nothing knows", []string{"settings", "set", "colour", "blue"}, "colour"},
		{"a cap that is not a number", []string{"settings", "set", "unread-cap", "lots"}, "whole number"},
		{"a cap below zero", []string{"settings", "set", "unread-cap", "-1"}, "negative"},
		{"a switch that is neither", []string{"settings", "set", "autopilot", "maybe"}, "on or off"},
		{"a flow name that is a path", []string{"settings", "set", "flow", "../task"}, "flow"},
		{"no value at all", []string{"settings", "set", "autopilot"}, "settings set needs value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, orbitHome := workspace(t)

			code, _, errOut := run(t, tc.args...)
			if code == 0 {
				t.Fatalf("%v exited 0", tc.args)
			}

			if !strings.Contains(errOut, tc.says) {
				t.Errorf("the refusal is %q, and does not say %q", errOut, tc.says)
			}

			if _, err := os.Stat(filepath.Join(orbitHome, "settings.json")); err == nil {
				t.Errorf("a refused setting wrote the settings file anyway")
			}
		})
	}
}

// TestTheConfirmationIsInTheReadersLanguage. `orbit set` is the command a
// reader uses to choose their language, and the line confirming that choice
// was the one line of it written in English no matter what they chose.
func TestTheConfirmationIsInTheReadersLanguage(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	if code, _, errOut := run(t, "settings", "set", "language", "es"); code != 0 {
		t.Fatalf("set language es exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "settings", "set", "autopilot", "on")
	if code != 0 {
		t.Fatalf("set autopilot on exited %d: %s", code, errOut)
	}

	if strings.Contains(out, "is now") {
		t.Errorf("the confirmation is in English for a reader who chose Spanish:\n%s", out)
	}

	if !strings.Contains(out, "autopilot") || !strings.Contains(out, "on") {
		t.Errorf("the confirmation does not say what the setting is now:\n%s", out)
	}
}

// TestTheRefusalsAreInTheReadersLanguage is the other half of the sweep
// above. Choosing a language is the first thing a reader does with this
// command, and every way of getting the next one wrong answered in English.
//
// The flow name that is a path is not here: that refusal is the flow
// package's, written where the name is validated rather than where it is
// typed, and it is still English for everyone.
func TestTheRefusalsAreInTheReadersLanguage(t *testing.T) {
	refusal := func(t *testing.T, language string, args ...string) string {
		t.Helper()
		t.Setenv("ORBIT_HOME", t.TempDir())

		if code, _, errOut := run(t, "settings", "set", "language", language); code != 0 {
			t.Fatalf("set language %s exited %d: %s", language, code, errOut)
		}

		code, _, errOut := run(t, args...)
		if code == 0 {
			t.Fatalf("%v exited 0", args)
		}

		return errOut
	}

	for _, args := range [][]string{
		{"settings", "set", "colour", "blue"},
		{"settings", "set", "unread-cap", "lots"},
		{"settings", "set", "unread-cap", "-1"},
		{"settings", "set", "autopilot", "maybe"},
		{"settings", "set", "autopilot"},
	} {
		t.Run(strings.Join(args[1:], " "), func(t *testing.T) {
			english := refusal(t, "en", args...)
			if spanish := refusal(t, "es", args...); spanish == english {
				t.Errorf("both readers are refused with %q", english)
			}
		})
	}
}
