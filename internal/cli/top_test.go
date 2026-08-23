package cli

// The wiring, and the one frame that proves it.
//
// Nothing here opens a window. `orbit top -once` builds every piece the
// full-screen window is built from — the store, the board reader, the
// settings adapter, the printer, the four ports — renders a single frame as
// plain text and returns. The event loop is never started, no terminal is
// opened, and no engine is reached: the gestures that spend money are on the
// other half of this command, and a test that exercised them would spend it.
//
// The assertions are about the wiring rather than about the pixels. What a
// frame looks like is asserted in internal/ui, against goldens and a width
// matrix; what this file asks is whether the repositories were found, the
// records folded, the counts right, and the language the one that was asked
// for.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

// esc is the byte a pipe must never be handed. A frame drawn for `less`, for
// CI or for TERM=dumb that still carries colour is a frame that reads as
// line noise, and one control byte is enough to say so.
const esc = 0x1b

// quietLocale takes the developer's own language out of the test.
// words.Resolve falls back to $LANG, so a machine set to Spanish would
// render a frame these tests assert English against — and the failure would
// be one nobody else could reproduce.
func quietLocale(t *testing.T) {
	t.Helper()
	t.Setenv("LANG", "")
	t.Setenv("ORBIT_LANG", "")
}

// emptyHome points $ORBIT_HOME at a state root that does not exist yet and
// answers where it will be.
func emptyHome(t *testing.T) string {
	t.Helper()
	quietLocale(t)
	home := filepath.Join(t.TempDir(), "orbit")
	t.Setenv("ORBIT_HOME", home)
	return home
}

// twoRepos builds a root holding two repositories, each with one task
// written down, over an empty state root.
//
// Two repositories and not one, because the repository column is dropped
// when every row would carry the same value: a board of one repository
// passes an assertion about that column without ever having drawn it.
func twoRepos(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	emptyHome(t)
	for _, r := range []struct{ dir, id, text string }{
		{"billing", "ACME-2", "reconcile the ledger nightly"},
		{"payments", "ACME-1", "retry the webhook on 5xx"},
	} {
		dir := filepath.Join(root, r.dir)
		initRepo(t, dir)
		if code, _, errOut := run(t, "new", "-repo", dir, "-id", r.id, r.text); code != 0 {
			t.Fatalf("new %s exited %d: %s", r.id, code, errOut)
		}
	}
	return root
}

// bandLine is the one line of a frame whose band is named, so a test can ask
// what the count above a list says without asserting the whole frame.
func bandLine(frame, band string) string {
	for _, line := range strings.Split(frame, "\n") {
		if strings.Contains(line, band) {
			return line
		}
	}
	return ""
}

func TestTopDrawsOneFrameOverEveryRepository(t *testing.T) {
	root := twoRepos(t)

	code, out, errOut := run(t, "top", root, "-once")
	if code != 0 {
		t.Fatalf("top exited %d: %s", code, errOut)
	}
	for _, want := range []string{"payments", "billing", "ACME-1", "ACME-2", "TO DO"} {
		if !strings.Contains(out, want) {
			t.Errorf("the frame does not mention %q:\n%s", want, out)
		}
	}
	if line := bandLine(out, "TO DO"); !strings.Contains(line, "2") {
		t.Errorf("the band over two tasks says %q, want the count in it", line)
	}
	if strings.ContainsRune(out, esc) {
		t.Error("the frame carries terminal escapes; -once is what a pipe reads")
	}
}

// A band nobody can open is a band whose rows nothing will ever show. The
// window opens on NEEDS YOU and RUNNING and leaves the other two shut,
// because the reader can press o — and in a pipe there is no o to press, so
// two fresh tasks would be a heading over nothing.
func TestOneFrameOpensEveryBandBecauseAPipeHasNoKeyboard(t *testing.T) {
	root := twoRepos(t)

	code, out, errOut := run(t, "top", root, "-once")
	if code != 0 {
		t.Fatalf("top exited %d: %s", code, errOut)
	}
	head := bandLine(out, "TO DO")
	rows := strings.Split(out, "\n")
	var after int
	var seen bool
	for _, line := range rows {
		if line == head {
			seen = true
			continue
		}
		if seen && strings.Contains(line, "ACME-") {
			after++
		}
	}
	if after != 2 {
		t.Errorf("%d task rows under the band, want 2:\n%s", after, out)
	}
}

func TestOneFrameSpeaksTheLanguageItWasAskedFor(t *testing.T) {
	root := twoRepos(t)

	code, out, errOut := run(t, "top", root, "-once", "-lang", "es")
	if code != 0 {
		t.Fatalf("top exited %d: %s", code, errOut)
	}
	if !strings.Contains(out, "POR HACER") {
		t.Errorf("the frame is not in Spanish:\n%s", out)
	}
	if strings.Contains(out, "TO DO") {
		t.Errorf("the frame is in two languages at once:\n%s", out)
	}
}

// flag stops at the first argument that is not a flag, so a directory typed
// before -once would leave the flag unread and open a full-screen window
// where a frame was asked for. That is the one flag mistake in this command
// whose failure is not a message but a terminal nobody expected.
func TestTheDirectoryMayComeBeforeOrAfterTheFlags(t *testing.T) {
	root := twoRepos(t)

	before, first, errOut := run(t, "top", root, "-once")
	if before != 0 {
		t.Fatalf("top <dir> -once exited %d: %s", before, errOut)
	}
	after, second, errOut := run(t, "top", "-once", root)
	if after != 0 {
		t.Fatalf("top -once <dir> exited %d: %s", after, errOut)
	}
	if first != second {
		t.Errorf("the two orders drew different frames:\n%s\n---\n%s", first, second)
	}
}

func TestTopRefusesASecondDirectory(t *testing.T) {
	root := twoRepos(t)

	code, _, errOut := run(t, "top", root, root, "-once")
	if code == 0 {
		t.Error("top took two directories and picked one of them silently")
	}
	if !strings.Contains(errOut, "one directory") {
		t.Errorf("the refusal is %q, want it to say how many directories it takes", errOut)
	}
}

// A state root with nothing in it is the first thing a new reader sees, and
// it has to be a sentence they can act on rather than a blank screen.
func TestTopOverAStateRootWithNothingInItSaysWhereItLooked(t *testing.T) {
	emptyHome(t)
	root := t.TempDir()

	code, out, errOut := run(t, "top", root, "-once")
	if code != 0 {
		t.Fatalf("top over an empty state root exited %d: %s", code, errOut)
	}
	if !strings.Contains(out, "No repositories under") {
		t.Errorf("the empty frame does not say what is missing:\n%s", out)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("the frame is blank")
	}
}

// The brake, and the one way it is silently disabled: UnreadCap 0 means no
// cap, deliberately, and a settings adapter that answered 0 for "never
// configured" or for "could not be read" would let runs pile up unread with
// nothing on screen having changed.
func TestTheUnreadCapNeverBecomesNoCapByAccident(t *testing.T) {
	emptyHome(t)
	s, err := store.Open()
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	cfg, err := newSettings(s)
	if err != nil {
		t.Fatalf("newSettings: %v", err)
	}
	if got := cfg.UnreadCap(); got <= 0 {
		t.Errorf("a settings file nobody has written answers %d, which is no cap at all", got)
	}

	if code, _, errOut := run(t, "set", "unread-cap", "0"); code != 0 {
		t.Fatalf("set unread-cap exited %d: %s", code, errOut)
	}
	if got := cfg.UnreadCap(); got != 0 {
		t.Errorf("the cap is %d after somebody chose no cap at all, want 0", got)
	}
}

// The setters go through the store, and the getters read what the store now
// holds — not a copy this adapter is keeping to itself.
func TestTheSettingsAdapterWritesThroughToTheFile(t *testing.T) {
	emptyHome(t)
	s, err := store.Open()
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	cfg, err := newSettings(s)
	if err != nil {
		t.Fatalf("newSettings: %v", err)
	}
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

	second, err := newSettings(s)
	if err != nil {
		t.Fatalf("newSettings again: %v", err)
	}
	if !second.Autopilot() || second.Language() != "es" {
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

// The synopsis is the whole interface on one screen, and a command missing
// from it is a command nobody finds.
func TestTopIsInTheSynopsis(t *testing.T) {
	var found bool
	for _, s := range synopsis {
		if strings.HasPrefix(s[0], "orbit top") {
			found = true
		}
	}
	if !found {
		t.Error("orbit top is not in the synopsis")
	}
}
