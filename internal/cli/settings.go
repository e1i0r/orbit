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

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/ui"
)

// settingsAdapter answers ui.Settings out of the settings file.
//
// The getters answer from memory and read nothing. They used to go to
// (*store.Store).Settings on every call, which is an unconditional
// os.ReadFile plus a json.Unmarshal — there is no cache in the store — and
// the window asks several of them per frame: unreadCap once for the header,
// once more per band header beside atUnreadCap, and autopilotOn again. At
// the board's poll that is five to ten file reads and parses a second, every
// one of them blocking, in the middle of a render. This program rejects
// lipgloss.HasDarkBackground by name for being exactly that (internal/ui/ui.go),
// and it should not do it to itself.
//
// So the file is read on a clock instead of on a question: see poll, which
// rides the same Cmd the board is refreshed from — off the event loop, off
// the render, once per board.RefreshEvery. Read-through survives, which is
// the property the field exists for: `orbit set` in another terminal, or an
// editor saving settings.json, still reaches an open window, one poll later
// rather than on the next frame. Nothing on this screen needs a fresher
// answer than that, and half a second is under the time it takes to look up.
//
// What is served is never store.Settings's zero value. UnreadCap 0 means
// "no cap" deliberately, and the number that stops a file nobody has written
// from meaning that is defaultUnreadCap — which store applies and the JSON
// does not carry. An adapter that answered the zero value when a read failed
// would turn "never configured" into "let everything through", and the brake
// would be gone with nothing on screen having changed. The window guards
// with limit > 0 on both sides, so the zero is load-bearing and cannot be
// defended against downstream.
//
// Reading through the store rather than settings.json also keeps the number
// on screen and the number task.Start enforces the same number. Start
// re-reads the file when a run is actually spawned, so both sides see the
// same defaults applied by the same code.
type settingsAdapter struct {
	store *store.Store

	// writing serialises the setters' read-modify-write, so that two
	// switches flipped at once cannot lose one another's field. It is held
	// across file I/O and mu deliberately is not: a getter called from a
	// render must never wait on a write to disk.
	writing sync.Mutex

	// mu guards last and gen, and is never held across a read of the file.
	// The getters are called from the render and the setters and the poll
	// from Cmds on other goroutines, so this is not theoretical.
	mu sync.Mutex

	// last is the newest configuration that was read without error, and it
	// is what every getter answers with. It is never the zero Settings:
	// newSettings refuses to build an adapter it could not read once, so
	// there is always a real answer behind this field.
	last store.Settings

	// gen counts writes this adapter made. A poll reads the file without
	// holding mu, so a write can land while its read is out; gen is how
	// that poll knows its answer is the older one and drops it rather than
	// putting the switch the reader just flipped back for half a second.
	gen uint64
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
//
// It is also what makes the cache honest from the first frame: there is a
// real answer in last before anything is drawn, so no poll has to have run
// for the header to be right.
func newSettings(s *store.Store) (*settingsAdapter, error) {
	cfg, err := s.Settings()
	if err != nil {
		return nil, err
	}
	return &settingsAdapter{store: s, last: cfg}, nil
}

// read is the one path every getter takes, and it touches no file.
func (a *settingsAdapter) read() store.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.last
}

// reread is the other half of read's bargain: the file, on the poll's clock.
//
// A read that fails leaves the last configuration that did not standing,
// which is the only answer available — the interface has no error to return.
// It is silent, and that is a known cost: a settings file that becomes
// unreadable while the window is open leaves the window drawing the last
// good answer with nothing saying so. It is the better half of the trade
// against answering zero, which would disable the cap.
//
// The file is read without mu held, so a poll cannot stall a frame, and the
// answer is dropped if a write landed while it was out.
func (a *settingsAdapter) reread() {
	a.mu.Lock()
	gen := a.gen
	a.mu.Unlock()

	cfg, err := a.store.Settings()
	if err != nil {
		return
	}
	a.keep(cfg, gen)
}

// keep takes what a poll read, unless a setter wrote while that read was in
// flight — which is what gen counts.
//
// It is a method of its own so that the guard can be stated in a test
// without a race to win: reread's own window, between letting go of mu and
// taking it again, is a few microseconds wide and no amount of repetition
// reliably lands inside it. Here the two orders are just two calls.
//
// Whatever a setter wrote is newer than whatever a straddling read saw,
// whichever order the two file operations happened to take: the setter's
// SaveSettings finished before it bumped gen, so a read that started before
// that bump either predates the write on disk or is about to be overwritten
// by the setter's own assignment.
func (a *settingsAdapter) keep(cfg store.Settings, gen uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.gen != gen {
		return
	}
	a.last = cfg
}

// write changes one field and puts the whole file back, which is what keeps
// a setting this window does not know about — an engine, a model — from
// being erased by a switch being flipped on screen.
//
// It refreshes the cache itself rather than waiting for the next poll: a
// switch flipped on screen that took half a second to read back as flipped
// would be a window arguing with the reader.
func (a *settingsAdapter) write(change func(*store.Settings)) error {
	a.writing.Lock()
	defer a.writing.Unlock()

	cfg, err := a.store.Settings()
	if err != nil {
		return err
	}
	change(&cfg)
	if err := a.store.SaveSettings(cfg); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.gen++
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

// poll is the window's Reader with the settings file on the same clock.
//
// internal/ui asks its Reader for the board every board.RefreshEvery, from a
// Cmd — its own goroutine, off the event loop and off the render — and that
// is the only clock this process has that is neither a keystroke nor a
// frame. It is where a file this window re-reads belongs, and hanging the
// settings on it costs one os.ReadFile beside a walk of the whole state
// tree.
//
// It is a wrapper here rather than a sixth method on ui.Settings and a
// second Cmd in internal/ui because which files are re-read, and how often,
// is a fact about where the state lives — which is this layer's business and
// not the window's. The window keeps asking exactly what it asked before.
type poll struct {
	ui.Reader
	cfg *settingsAdapter
}

// Refresh re-reads the settings, and then the board.
//
// The settings first, so that a frame drawn from the board this call returns
// is drawn against the configuration that was in force when it was read. The
// order costs nothing and the other one would show a cap one poll behind the
// tasks it is being applied to.
func (p poll) Refresh() (board.Board, board.Changed, error) {
	p.cfg.reread()
	return p.Reader.Refresh()
}
