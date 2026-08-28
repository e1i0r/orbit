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
	Theme     string `json:"theme,omitempty"`
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
//
// A file that is there but will not parse is moved aside first rather than
// overwritten. Settings answers an unparseable file with the defaults —
// deliberately, so a reader always has something usable — and every setter
// above this one is a read-modify-write. Without this, one switch flipped on
// screen would replace a whole configuration with the defaults plus that
// switch, and the engine, model and theme somebody chose would be gone with
// nothing anywhere to say they had ever been set.
//
// Moving it aside is not repair and does not pretend to be. Whatever was in
// there is left where a person can read it, and the write goes ahead, so a
// file nobody can parse cannot lock the settings screen either.
func (s *Store) SaveSettings(cfg Settings) error {
	path := s.settingsPath()
	if err := keepUnreadable(path); err != nil {
		return err
	}
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return fmt.Errorf("create %q: %w", filepath.Dir(path), err)
	}
	return writeAtomically(path, body)
}

// writeAtomically puts body at path in one step: a temporary file beside it,
// flushed, then renamed over the top.
//
// os.WriteFile truncates first and writes second, and everything between
// those two is a settings.json that exists, is shorter than it should be and
// will not parse. A process killed there — or a machine that loses power —
// leaves exactly the file keepUnreadable above was written to cope with, so
// this is the other half of the same fix: one stops the loss, this stops the
// damage that causes it.
//
// The temporary is made in the same directory rather than in the system's
// temp dir, because a rename is atomic only within one filesystem; across
// two it is a copy, which is the thing being avoided.
func writeAtomically(path string, body []byte) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create a temporary file beside %q: %w", path, err)
	}
	tmp := f.Name()
	// Every path that does not end in the rename takes the temporary with
	// it, so a save that failed leaves the directory as it found it rather
	// than littered with half-written settings.
	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()
	if err = f.Chmod(fileMode); err != nil {
		return fmt.Errorf("set the mode of %q: %w", tmp, err)
	}
	if _, err = f.Write(body); err != nil {
		return fmt.Errorf("write %q: %w", tmp, err)
	}
	if err = f.Sync(); err != nil {
		return fmt.Errorf("flush %q: %w", tmp, err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("close %q: %w", tmp, err)
	}
	if err = os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace %q with %q: %w", path, tmp, err)
	}
	return nil
}

// unreadableSuffix is what an unparseable settings file is renamed with. It
// is a suffix rather than a directory so the two sit side by side: whoever
// opens the state root to find out why their engine reset sees both files in
// the same listing.
const unreadableSuffix = ".unreadable"

// keepUnreadable moves a settings file that will not parse out of the way,
// and leaves every other file exactly where it is.
//
// The parse it does is the same one Settings does — unmarshalling into
// Settings, not merely checking the JSON is well formed — because the two
// have to agree on what "unreadable" means. A file that is valid JSON but
// has a string where the unread cap goes is a file Settings answers with the
// defaults, so it is a file this has to keep.
//
// A file that cannot be read at all is not this function's problem: the
// write that follows will hit the same fault and report it in its own terms.
func keepUnreadable(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil //nolint:nilerr // a file that is absent or unreadable is the write's problem, not this one's
	}
	var probe Settings
	if json.Unmarshal(body, &probe) == nil {
		return nil
	}
	aside := path + unreadableSuffix
	if err := os.Rename(path, aside); err != nil {
		return fmt.Errorf("move the unreadable %q aside to %q: %w", path, aside, err)
	}
	return nil
}
