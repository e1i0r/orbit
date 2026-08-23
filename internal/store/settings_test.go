package store

import (
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
