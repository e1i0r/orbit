package cli

// Theme() and write() (settings.go) and set() (set.go) branches nothing else
// in this package reaches: a theme nobody has chosen defaulting to
// "monokai" rather than "", write()'s read-modify-write failing on either
// half of the round trip, and set()'s own store.Open/Settings/SaveSettings
// failures.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

// Both setters over this file — the command and the window — go through the
// store's own update, so that a change made in one while the other is holding
// the file is refused rather than written over. The lock is left in place by
// hand here: a real second writer holds it for the length of one read and one
// write, which is too short to arrange from a test.

// TestSetWaitsForAChangeAlreadyUnderWayAndThenNamesTheLock. `orbit set` used
// to read the file, change one field and write the whole of it back, with a
// window free to do the same in between; what it wrote was a copy made before
// the window's change existed, and the setting the window saved was gone with
// both sides reporting success.
func TestSetWaitsForAChangeAlreadyUnderWayAndThenNamesTheLock(t *testing.T) {
	_, orbitHome := workspace(t)

	if err := os.MkdirAll(orbitHome, 0o700); err != nil {
		t.Fatalf("mkdir the state root: %v", err)
	}

	lock := filepath.Join(orbitHome, "settings.json.lock")
	if err := os.WriteFile(lock, nil, 0o600); err != nil {
		t.Fatalf("write the held lock: %v", err)
	}

	code, _, errOut := run(t, "set", "engine", "codex")
	if code == 0 {
		t.Fatal("set wrote the settings while another change held them")
	}

	if !strings.Contains(errOut, lock) {
		t.Errorf("the refusal does not say which file to delete:\n%s", errOut)
	}

	if cfg := settings(t, orbitHome); cfg.Engine != "" {
		t.Errorf("engine is %q on disk; a change that was refused must not have been written", cfg.Engine)
	}
}

// TestASwitchFlippedOnScreenWaitsForTheSameLock, and says so rather than
// showing the reader a setting that was never saved.
func TestASwitchFlippedOnScreenWaitsForTheSameLock(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := newSettings(s)
	if err != nil {
		t.Fatal(err)
	}

	lock := filepath.Join(root, "settings.json.lock")
	if err := os.WriteFile(lock, nil, 0o600); err != nil {
		t.Fatalf("write the held lock: %v", err)
	}

	err = cfg.SetEngine("codex")
	if err == nil {
		t.Fatal("the window wrote the settings while another change held them")
	}

	if !strings.Contains(err.Error(), lock) {
		t.Errorf("the refusal does not say which file to delete: %v", err)
	}

	if got := cfg.Engine(); got != "" {
		t.Errorf("the window reads back %q; a change it could not save must not be shown as saved", got)
	}
}

func TestThemeDefaultsToMonokaiWhenNobodyHasChosenOne(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := newSettings(s)
	if err != nil {
		t.Fatal(err)
	}

	if got := cfg.Theme(); got != "monokai" {
		t.Errorf("Theme() on a fresh store = %q, want monokai", got)
	}
}

func TestSettingsAdapterWriteFailsWhenSettingsCannotBeReread(t *testing.T) {
	home := t.TempDir()

	s, err := store.New(home)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := newSettings(s)
	if err != nil {
		t.Fatal(err)
	}

	// A directory where settings.json goes fails write()'s own re-read.
	if err := os.MkdirAll(filepath.Join(home, "settings.json"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := cfg.SetAutopilot(true); err == nil {
		t.Error("SetAutopilot succeeded while settings.json was unreadable")
	}
}

func TestSettingsAdapterWriteFailsWhenSettingsCannotBeSaved(t *testing.T) {
	home := t.TempDir()

	s, err := store.New(home)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := newSettings(s)
	if err != nil {
		t.Fatal(err)
	}
	// A real write first, so settings.json exists as an ordinary file.
	if err := cfg.SetAutopilot(true); err != nil {
		t.Fatalf("SetAutopilot: %v", err)
	}

	// The directory, not the file: settings are replaced by renaming a
	// temporary over the top, and a rename asks the directory for
	// permission rather than the file it replaces.
	if err := os.Chmod(home, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(home, 0o700) }() //nolint:errcheck

	if err := cfg.SetLanguage("es"); err == nil {
		t.Error("SetLanguage succeeded while settings.json could not be written")
	}
}

func TestSetEarlyExitsAndStoreFailures(t *testing.T) {
	// 1. A flag parse failure.
	t.Setenv("ORBIT_HOME", t.TempDir())

	if code, _, errOut := run(t, "set", "-nosuchflag"); code == 0 {
		t.Error("set with an unknown flag exited 0")
	} else if errOut == "" {
		t.Error("set failed silently on a bad flag")
	}

	// 2. store.Open fails: a regular file where $ORBIT_HOME would go.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ORBIT_HOME", filepath.Join(blocker, "orbit"))

	if code, _, errOut := run(t, "set"); code == 0 {
		t.Error("set with an unmakeable state root exited 0")
	} else if errOut == "" {
		t.Error("set failed silently with an unmakeable state root")
	}

	// 3. s.Settings() fails: settings.json is a directory.
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	if err := os.MkdirAll(filepath.Join(home, "settings.json"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if code, _, errOut := run(t, "set"); code == 0 {
		t.Error("set over unreadable settings exited 0")
	} else if errOut == "" {
		t.Error("set failed silently over unreadable settings")
	}
}

func TestSetFailsWhenSettingsCannotBeSaved(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)
	// A first, real set to make settings.json exist as an ordinary file.
	if code, _, errOut := run(t, "set", "autopilot", "on"); code != 0 {
		t.Fatalf("set autopilot on exited %d: %s", code, errOut)
	}

	// See the note above: the write goes through a rename, so it is the
	// directory that has to refuse.
	if err := os.Chmod(home, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(home, 0o700) }() //nolint:errcheck

	code, _, errOut := run(t, "set", "autopilot", "off")
	if code == 0 {
		t.Error("set into a directory that refuses writes exited 0")
	}

	if errOut == "" {
		t.Error("set failed silently into a directory that refuses writes")
	}
}
