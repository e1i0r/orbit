package cli

// The settings port: what it answers, where the answer comes from, and when
// it goes back to the file for a new one.
//
// The frame is asserted next door in top_test.go. These tests are about the
// adapter behind it — getters that touch no file because several of them are
// called per rendered frame, a poll that touches it on the board's clock so
// that `orbit set` in another terminal still reaches an open window, and the
// one number on this screen that must never become zero by accident.

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/store"
)

// openSettings builds the pair internal/cli hands the window: a Settings
// that answers from memory, and the Reader whose Refresh is what puts a new
// answer in it. They are built together because neither is correct alone —
// an adapter with nothing polling it never sees the file change again.
func openSettings(t *testing.T) (*settingsAdapter, poll) {
	t.Helper()
	s, err := store.Open()
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	cfg, err := newSettings(s)
	if err != nil {
		t.Fatalf("newSettings: %v", err)
	}
	// An empty directory to be the board's root: these tests are about the
	// settings file the poll carries, and a board with nothing on it is the
	// cheapest one to poll.
	return cfg, poll{Reader: board.NewReader(s, t.TempDir()), cfg: cfg}
}

// The header reads the cap once a frame, every band header reads it again
// beside atUnreadCap, and the start dialog reads it twice more. At the
// board's poll a file read plus a JSON parse in each of those is five to ten
// blocking reads a second in the middle of a render — the thing this program
// refuses lipgloss.HasDarkBackground by name for doing.
//
// So the getters answer from memory, and the file is read on the poll's
// clock. Both halves are asserted here, because either alone is a bug: a
// getter that still reads is the finding, and a poll that does not read is
// a window that never notices `orbit set` again.
func TestTheGettersAnswerFromMemoryAndThePollIsWhatRefreshesThem(t *testing.T) {
	emptyHome(t)
	cfg, p := openSettings(t)
	before := cfg.UnreadCap()

	// Another process — which is what `orbit set` is, and what an editor
	// saving settings.json is.
	if code, _, errOut := run(t, "set", "unread-cap", "9"); code != 0 {
		t.Fatalf("set unread-cap exited %d: %s", code, errOut)
	}

	for i := range 3 {
		if got := cfg.UnreadCap(); got != before {
			t.Fatalf("getter %d answered %d: it read the settings file, and several of these are drawn per frame", i, got)
		}
	}

	if _, _, err := p.Refresh(); err != nil {
		t.Fatalf("the poll: %v", err)
	}
	if got := cfg.UnreadCap(); got != 9 {
		t.Errorf("the cap is %d after a poll over a file another process changed, want 9", got)
	}
}

// The wiring, not the pieces: the Reader the window is actually handed has
// to be the one that keeps the settings in step. An adapter and a poll that
// both work while window() hands over the bare board reader is a window
// whose settings freeze the moment it opens.
func TestTheWindowIsHandedAReaderThatKeepsItsSettingsInStep(t *testing.T) {
	emptyHome(t)
	root := t.TempDir()

	opts, _, err := window(root, "")
	if err != nil {
		t.Fatalf("window: %v", err)
	}
	if code, _, errOut := run(t, "set", "unread-cap", "7"); code != 0 {
		t.Fatalf("set unread-cap exited %d: %s", code, errOut)
	}
	if _, _, err := opts.Reader.Refresh(); err != nil {
		t.Fatalf("the window's poll: %v", err)
	}
	if got := opts.Settings.UnreadCap(); got != 7 {
		t.Errorf("the cap is %d after the window polled, want 7", got)
	}
}

// A read that fails leaves the last good answer standing, because
// ui.Settings has no error to return and the alternative — the zero
// Settings — is UnreadCap 0, which means no cap at all. The brake would come
// off with nothing on screen having changed.
func TestASettingsReadThatFailsLeavesTheLastGoodAnswerStanding(t *testing.T) {
	home := emptyHome(t)
	cfg, p := openSettings(t)
	good := cfg.UnreadCap()
	if good <= 0 {
		t.Fatalf("the fixture starts at %d, which is no cap at all", good)
	}

	// A directory where the file goes fails the read for every user,
	// including root — a mode of 0000 does not.
	if err := os.MkdirAll(filepath.Join(home, "settings.json"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, _, err := p.Refresh(); err != nil {
		t.Fatalf("the poll: %v", err)
	}
	if got := cfg.UnreadCap(); got != good {
		t.Errorf("the cap is %d after a read that failed, want the last good answer %d", got, good)
	}
}

// A poll reads the file without holding the lock, so a write can land while
// its read is out. What it read is then the older answer, and taking it
// would put the switch the reader just flipped back for half a second.
//
// The two orders are two calls rather than a race to win: reread's window is
// microseconds wide and repetition does not reliably land in it — twenty
// runs of the concurrency test below pass with the guard deleted. keep is
// where the guard lives, so keep is what is asked.
func TestAPollThatStraddledAWriteDropsWhatItRead(t *testing.T) {
	emptyHome(t)
	cfg, _ := openSettings(t)
	// What a poll read off the disk before the reader pressed A. gen is 0:
	// this adapter has written nothing yet.
	stale := cfg.read()

	if err := cfg.SetAutopilot(true); err != nil {
		t.Fatalf("SetAutopilot: %v", err)
	}
	cfg.keep(stale, 0)
	if !cfg.Autopilot() {
		t.Error("a poll whose read straddled the write put the switch back")
	}

	// And the guard is what dropped it, rather than something else: the same
	// value, offered against the generation the write left behind, is taken.
	cfg.keep(stale, 1)
	if cfg.Autopilot() {
		t.Error("keep refused a read that no write straddled; the poll would never see the file again")
	}
}

// The getters run in the render and the setters and the poll in Cmds on
// other goroutines. -race is the whole of this test's assertion; the guard
// above is what the correctness of the outcome rests on.
func TestTheSettingsAreReadWrittenAndPolledAtOnce(t *testing.T) {
	emptyHome(t)
	cfg, p := openSettings(t)

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(3)
		go func() { defer wg.Done(); _, _ = cfg.UnreadCap(), cfg.Autopilot() }()
		go func() {
			defer wg.Done()
			if _, _, err := p.Refresh(); err != nil {
				t.Errorf("the poll: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			if err := cfg.SetAutopilot(true); err != nil {
				t.Errorf("SetAutopilot: %v", err)
			}
		}()
	}
	wg.Wait()

	if !cfg.Autopilot() {
		t.Error("autopilot reads as off after twenty writes turning it on")
	}
}

// The brake, and the one way it is silently disabled: UnreadCap 0 means no
// cap, deliberately, and a settings adapter that answered 0 for "never
// configured" or for "could not be read" would let runs pile up unread with
// nothing on screen having changed.
func TestTheUnreadCapNeverBecomesNoCapByAccident(t *testing.T) {
	emptyHome(t)
	cfg, p := openSettings(t)
	if got := cfg.UnreadCap(); got <= 0 {
		t.Errorf("a settings file nobody has written answers %d, which is no cap at all", got)
	}

	if code, _, errOut := run(t, "set", "unread-cap", "0"); code != 0 {
		t.Fatalf("set unread-cap exited %d: %s", code, errOut)
	}
	if _, _, err := p.Refresh(); err != nil {
		t.Fatalf("the poll: %v", err)
	}
	if got := cfg.UnreadCap(); got != 0 {
		t.Errorf("the cap is %d after somebody chose no cap at all, want 0", got)
	}
}

// The setters go through the store, and what they wrote is readable at once
// — not on the next poll: a switch flipped on screen that took half a second
// to read back as flipped would be a window arguing with the reader.
func TestTheSettingsAdapterWritesThroughToTheFile(t *testing.T) {
	emptyHome(t)
	cfg, _ := openSettings(t)
	if err := cfg.SetAutopilot(true); err != nil {
		t.Fatalf("SetAutopilot: %v", err)
	}
	if err := cfg.SetLanguage("es"); err != nil {
		t.Fatalf("SetLanguage: %v", err)
	}
	if !cfg.Autopilot() {
		t.Error("autopilot was switched on and reads as off")
	}
	if cfg.Language() != "es" {
		t.Errorf("the language reads as %q after being set to es", cfg.Language())
	}

	// A second adapter over its own store is what another process gets, and
	// it is the only witness that the file was written rather than a field.
	second, err := store.Open()
	if err != nil {
		t.Fatalf("open the store again: %v", err)
	}
	other, err := newSettings(second)
	if err != nil {
		t.Fatalf("newSettings again: %v", err)
	}
	if !other.Autopilot() || other.Language() != "es" {
		t.Error("the settings did not reach the file: a second adapter cannot see them")
	}
}

// A settings file that cannot be read is where the cap would silently become
// no cap, so the command refuses to open at all and names the file. The file
// is made a directory because that fails for every user, including root —
// a mode of 0000 does not.
func TestTopRefusesASettingsFileItCannotRead(t *testing.T) {
	home := emptyHome(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "settings.json"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	code, _, errOut := run(t, "top", root, "-once")
	if code == 0 {
		t.Error("top opened over settings it could not read")
	}
	if !strings.Contains(errOut, "settings.json") {
		t.Errorf("the refusal is %q, want it to name the file", errOut)
	}
}
