package store

// Settings is the user's persisted configuration, one JSON file at the root
// of the state tree. Every field's Go zero value is a working setting in
// its own right — UnreadCap: 0 means no cap — so a settings file that will
// not parse yields those defaults rather than failing, for the same reason
// a broken translation catalogue yields English: a reader that cannot read
// still has to hand back something the rest of the program can use.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// defaultUnreadCap is what a store that has never saved settings starts
// with. It is deliberately not the zero value: UnreadCap: 0 means "no cap",
// and that has to stay something a user chooses, never a fact about never
// having chosen at all.
const defaultUnreadCap = 5

// defaultFlow is which pipeline a task is written against when the user has
// never chosen one. The word is also written down in internal/flow, as
// flow.Default, because this package imports nothing of Orbit's — that
// absence is what keeps the on-disk layout from depending on anything above
// it. Two copies of a word drift, so a test in internal/task, which imports
// both, is what holds them together.
const defaultFlow = "task"

// Settings is the user's persisted configuration.
type Settings struct {
	Language  string `json:"language,omitempty"`
	Autopilot bool   `json:"autopilot,omitempty"`
	UnreadCap int    `json:"unreadCap,omitempty"`
	Engine    string `json:"engine,omitempty"`
	Model     string `json:"model,omitempty"`
	Flow      string `json:"flow,omitempty"`
}

// settingsPath is the one file settings live in, at the root of the state
// tree rather than under any one repository: settings are not scoped to a
// repository.
func (s *Store) settingsPath() string {
	return filepath.Join(s.root, "settings.json")
}

// Settings reads the persisted configuration.
//
// A file that has never been saved and a file that will not parse are
// answered the same way: the defaults. Only a real I/O error — not "there
// is nothing here yet" and not "what is here is not JSON" — is returned.
func (s *Store) Settings() (Settings, error) {
	path := s.settingsPath()
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Settings{UnreadCap: defaultUnreadCap, Flow: defaultFlow}, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("read %q: %w", path, err)
	}
	var cfg Settings
	if err := json.Unmarshal(body, &cfg); err != nil {
		// A settings file that will not parse yields the defaults, not a
		// failure — the same reasoning a broken catalogue answers with
		// English rather than an error.
		return Settings{UnreadCap: defaultUnreadCap, Flow: defaultFlow}, nil //nolint:nilerr // deliberate: unparseable settings yield the defaults
	}
	return cfg, nil
}

// SaveSettings writes the configuration, replacing whatever was there.
func (s *Store) SaveSettings(cfg Settings) error {
	path := s.settingsPath()
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return fmt.Errorf("create %q: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, body, fileMode); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}
