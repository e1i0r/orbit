package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsDefaultUnreadCapWhenTheFileIsAbsent(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if got.UnreadCap != 5 {
		t.Errorf("UnreadCap = %d, want 5 for a store that has never saved settings", got.UnreadCap)
	}
	if got.Flow != "task" {
		t.Errorf("Flow = %q, want task for a store that has never saved settings", got.Flow)
	}
}

func TestSettingsRoundTrips(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	want := Settings{
		Language:  "es",
		Autopilot: true,
		UnreadCap: 20,
		Engine:    "claude",
		Model:     "opus",
		Flow:      "careful",
	}
	if err := s.SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if got != want {
		t.Errorf("Settings() = %+v, want %+v", got, want)
	}
}

func TestSettingsZeroUnreadCapMeansNoCapAndIsHonoredOnceSaved(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.SaveSettings(Settings{UnreadCap: 0}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if got.UnreadCap != 0 {
		t.Errorf("UnreadCap = %d, want 0 — an explicitly saved zero must not be promoted to the new-store default", got.UnreadCap)
	}
}

func TestSettingsThatWillNotParseYieldsTheDefaults(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	path := filepath.Join(s.Root(), "settings.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings: %v — a settings file that will not parse must not fail the reader", err)
	}
	if got.UnreadCap != 5 {
		t.Errorf("UnreadCap = %d, want the default 5", got.UnreadCap)
	}
	if got.Flow != "task" {
		t.Errorf("Flow = %q, want the default task", got.Flow)
	}
}

func TestSettingsFileIsPrivate(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.SaveSettings(Settings{Language: "en"}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	info, err := os.Stat(filepath.Join(s.Root(), "settings.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("settings file is %o, want 600", perm)
	}
}

func TestSaveSettingsMovesAnUnreadableFileAsideRatherThanOverwritingIt(t *testing.T) {
	root := t.TempDir()
	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	path := filepath.Join(root, "settings.json")
	broken := `{"engine": "codex", "model": "gpt-5", "theme": "mid`
	if err := os.WriteFile(path, []byte(broken), 0o600); err != nil {
		t.Fatalf("write broken settings: %v", err)
	}

	// What a switch flipped on screen does: read (which yields the
	// defaults, because the file will not parse), change one field, write
	// the whole thing back.
	cfg, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	cfg.Autopilot = true
	if err := s.SaveSettings(cfg); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	kept, err := os.ReadFile(path + unreadableSuffix)
	if err != nil {
		t.Fatalf("the unreadable settings were overwritten instead of kept: %v", err)
	}
	if string(kept) != broken {
		t.Errorf("kept %q, want the file exactly as it was: %q", kept, broken)
	}
	got, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings after save: %v", err)
	}
	if !got.Autopilot {
		t.Error("the save did not go through; a file nobody can parse must not lock the settings")
	}
}

func TestSaveSettingsLeavesAReadableFileWhereItIs(t *testing.T) {
	root := t.TempDir()
	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.SaveSettings(Settings{Engine: "claude"}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if err := s.SaveSettings(Settings{Engine: "codex"}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	aside := filepath.Join(root, "settings.json"+unreadableSuffix)
	if _, err := os.Stat(aside); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a settings file that parses was moved aside: %v", err)
	}
}

func TestSaveSettingsLeavesNoTemporaryFileBehind(t *testing.T) {
	root := t.TempDir()
	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, engine := range []string{"claude", "codex", "opencode"} {
		if err := s.SaveSettings(Settings{Engine: engine}); err != nil {
			t.Fatalf("SaveSettings %q: %v", engine, err)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read root: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 1 || names[0] != "settings.json" {
		t.Errorf("state root holds %v, want only settings.json: the write goes through a temporary and has to take it with it", names)
	}
	info, err := os.Stat(filepath.Join(root, "settings.json"))
	if err != nil {
		t.Fatalf("stat settings: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600: a file arriving by rename must not be more open than one written in place", info.Mode().Perm())
	}
}
