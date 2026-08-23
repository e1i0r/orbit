package cli

// The window's settings port, over the settings file.
//
// It is here and not in internal/store because the shape ui.Settings asks
// for — five methods, getters that cannot fail — is the window's shape and
// not the file's. internal/store answers one struct and an error, which is
// the honest shape for a file; turning one into the other is a decision, and
// this is the layer where store and ui are allowed to meet.

import (
	"sync"

	"github.com/e1i0r/orbit/internal/store"
)

// settingsAdapter answers ui.Settings out of the settings file.
//
// Every getter reads through (*store.Store).Settings and never the file
// itself, and that is the whole reason this type exists rather than a struct
// of five fields read once at startup. UnreadCap 0 means "no cap"
// deliberately, and the number that stops a file nobody has written from
// meaning that is defaultUnreadCap — which store applies and the JSON does
// not carry. An adapter that parsed settings.json itself, or that answered
// the zero value when a read failed, would turn "never configured" into "let
// everything through", and the brake would be gone with nothing on screen
// having changed. The window guards with limit > 0 on both sides, so the
// zero is load-bearing and cannot be defended against downstream.
//
// Reading through also keeps the number on screen and the number task.Start
// enforces the same number. Start re-reads the file when a run is actually
// spawned; a window holding a copy from when it opened would show a cap
// somebody changed an hour ago in another terminal, and refuse — or fail to
// refuse — for a reason the reader cannot see.
type settingsAdapter struct {
	store *store.Store

	// mu guards last. The getters are called from the render and the
	// setters from a Cmd on another goroutine, so this is not theoretical.
	mu sync.Mutex
	// last is the newest configuration that was read without error, and it
	// is what a failed read answers with. It is never the zero Settings:
	// newSettings refuses to build an adapter it could not read once, so
	// there is always a real answer behind this field.
	last store.Settings
}

// newSettings reads the settings once and refuses to build an adapter over a
// file it cannot read.
//
// The refusal is the point. ui.Settings's getters return no error, so a read
// failure has nowhere to go once the window is open; here it has somewhere,
// and what a reader gets is one sentence naming the file rather than a
// window quietly running without the brake. A file that has never been
// written and one that will not parse are not failures — store answers both
// with the defaults, for the same reason a broken catalogue answers with
// English.
func newSettings(s *store.Store) (*settingsAdapter, error) {
	cfg, err := s.Settings()
	if err != nil {
		return nil, err
	}
	return &settingsAdapter{store: s, last: cfg}, nil
}

// read is the one path every getter takes to the file.
//
// A read that fails answers with the last configuration that did not, which
// is the only answer available: the interface has no error to return. It is
// silent, and that is a known cost — a settings file that becomes unreadable
// while the window is open leaves the window drawing the last good answer
// with nothing saying so. It is the better half of the trade against
// answering zero, which would disable the cap.
func (a *settingsAdapter) read() store.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	cfg, err := a.store.Settings()
	if err != nil {
		return a.last
	}
	a.last = cfg
	return cfg
}

// write changes one field and puts the whole file back, which is what keeps
// a setting this window does not know about — an engine, a model — from
// being erased by a switch being flipped on screen.
func (a *settingsAdapter) write(change func(*store.Settings)) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	cfg, err := a.store.Settings()
	if err != nil {
		return err
	}
	change(&cfg)
	if err := a.store.SaveSettings(cfg); err != nil {
		return err
	}
	a.last = cfg
	return nil
}

// Autopilot is whether a run walks its gates without asking.
func (a *settingsAdapter) Autopilot() bool { return a.read().Autopilot }

// SetAutopilot writes that switch down.
func (a *settingsAdapter) SetAutopilot(on bool) error {
	return a.write(func(cfg *store.Settings) { cfg.Autopilot = on })
}

// Language is the code the saved settings name, which is one of the four
// sources words.Resolve weighs and not the answer on its own.
func (a *settingsAdapter) Language() string { return a.read().Language }

// SetLanguage writes the reader's choice down, so the next window opens in
// it without a flag.
func (a *settingsAdapter) SetLanguage(lang string) error {
	return a.write(func(cfg *store.Settings) { cfg.Language = lang })
}

// UnreadCap is how many finished tasks may sit unread before a new run is
// refused. Zero is no cap at all, and it means that only because somebody
// chose it — see the type's own comment.
func (a *settingsAdapter) UnreadCap() int { return a.read().UnreadCap }
