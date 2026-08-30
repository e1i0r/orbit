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
	"regexp"
	"strings"
	"testing"
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

	var (
		after int
		seen  bool
	)

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

// The two orders draw the same frame end to end.
//
// This is not what guards -once. Every test in this file reaches the plain
// branch through interactive(out) — run() hands the command a buffer — so
// both invocations below would be equal whether or not the flag was ever
// read. What guards -once is TestTopReadsItsFlagsOnEitherSideOfTheDirectory,
// which asks parseTop what it parsed. What this one adds is that everything
// downstream of the flags agrees too: the same root, the same board, the
// same words, byte for byte.
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

	if stripTheClock(first) != stripTheClock(second) {
		t.Errorf("the two orders drew different frames:\n%s\n---\n%s", first, second)
	}
}

// stripTheClock takes out of a frame the two numbers a passing second
// changes: how long the read took, and how long ago each task was written.
//
// The second of those is why this exists. The frames are drawn a fraction
// of a second apart and the age is whole seconds, so two runs either side
// of a tick differ by 0s against 1s in a comparison that is otherwise byte
// for byte — which has nothing to do with the order the flags were typed.
//
// The padding in front of an age goes with it. The column is right aligned,
// so 9s and 10s are not the same width, and normalising the number without
// the space before it would still leave two lines that differ.
func stripTheClock(s string) string {
	read := regexp.MustCompile(`\d+ms read`)
	age := regexp.MustCompile(`(?m) +\d+[smhd]$`)

	return age.ReplaceAllString(read.ReplaceAllString(s, "Xms read"), " Xs")
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

// A directory with nothing in it is a sentence a reader can act on rather
// than a blank screen — and it is the one case where "no repositories" is
// the true thing to say. The other two empty states are next door in
// topdir_test.go; this is the one that must survive them.
func TestTopOverADirectoryWithNoRepositoryInItSaysWhereItLooked(t *testing.T) {
	emptyHome(t)
	root := t.TempDir()

	code, out, errOut := run(t, "top", root, "-once")
	if code != 0 {
		t.Fatalf("top over an empty directory exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "No repositories under") {
		t.Errorf("the empty frame does not say what is missing:\n%s", out)
	}

	if strings.Contains(out, "no tasks written") {
		t.Errorf("the frame says there are repositories with no tasks, over a directory with no repository:\n%s", out)
	}

	if strings.TrimSpace(out) == "" {
		t.Error("the frame is blank")
	}
}

// The table is the whole interface on one screen, and a command missing from
// it is a command nobody finds and nothing dispatches.
func TestTopIsInTheTable(t *testing.T) {
	if _, ok := lookup("top"); !ok {
		t.Error("orbit top is not in the command table")
	}
}

// The four rows of the choice between the two ways out of this command.
//
// The row that matters is the first: a terminal, and -once typed anyway. It
// is the only place in this package where -once is the deciding term, and
// the only reason drawsOneFrame is a function rather than the expression it
// replaced — every test here hands the command a buffer, so interactive() is
// false in all of them and the second term decides the branch on its own.
func TestOnceDrawsAFrameEvenWhenThereIsATerminalToOpen(t *testing.T) {
	cases := []struct {
		name           string
		once, terminal bool
		want           bool
	}{
		{"-once, at a terminal", true, true, true},
		{"-once, down a pipe", true, false, true},
		{"no flag, down a pipe", false, false, true},
		{"no flag, at a terminal", false, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := drawsOneFrame(c.once, c.terminal); got != c.want {
				t.Errorf("drawsOneFrame(once=%v, terminal=%v) = %v, want %v", c.once, c.terminal, got, c.want)
			}
		})
	}
}

// TestTopSaysWhenItCouldNotOpenItsLog. The window is drawn either way —
// nothing on it is read out of the log — but a diagnostic log that has been
// going nowhere since the first frame is the thing a reader finds out about a
// week later, from an empty file, while looking for why something else broke.
func TestTopSaysWhenItCouldNotOpenItsLog(t *testing.T) {
	root, orbitHome := workspace(t)
	// A file where the log directory goes. os.MkdirAll refuses to make a
	// directory over one, which is the way to make Init fail that does not
	// depend on being able to take a permission away from whoever is running
	// the tests.
	if err := os.MkdirAll(orbitHome, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(orbitHome, "logs"), []byte("not a directory\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := run(t, "top", "-once", root)
	if !strings.Contains(errOut, "logs") {
		t.Errorf("top opened no log and said nothing about it: %q", errOut)
	}

	if !strings.Contains(out, "1 repo") {
		t.Errorf("a log that would not open took the board with it: %q", out)
	}

	if code == 0 {
		t.Error("top exited 0 with a log that was never opened")
	}
}
