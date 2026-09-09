//go:build integration

package cli

// The cockpit, driven the way a person drives it.
//
// The window here is the one `orbit top` builds — window() and ui.New, over
// a state root this test made and a board read out of it — so a port the
// command stops wiring is a port these tests stop finding. Building an
// Options by hand would test a window nobody ships.
//
// What is asserted is the frame: the reader presses a key and the screen
// they asked for is the screen that comes up. That is the whole promise the
// landing makes about the cockpit.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui"
	"github.com/e1i0r/orbit/internal/words"
)

// cockpitBoard is a state root with one repository and the tasks a test needs.
func cockpitBoard(t *testing.T) (Context, string) {
	t.Helper()

	root := t.TempDir()
	t.Setenv("ORBIT_HOME", filepath.Join(root, "state"))
	t.Setenv("HOME", filepath.Join(root, "home"))

	code := filepath.Join(root, "code")
	repo := filepath.Join(code, "ledger")

	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("make the repository: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "ledger.go"), []byte("package ledger\n"), 0o600); err != nil {
		t.Fatalf("write a file: %v", err)
	}

	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "test@orbit"},
		{"config", "user.name", "orbit test"},
		{"add", "-A"},
		{"commit", "-m", "the ledger"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo

		if said, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, said)
		}
	}

	var out, errOut bytes.Buffer

	ctx := Context{Out: &out, Err: &errOut, Words: words.For("en")}

	ranCommand(t, ctx, "new", "-repo", repo, "-id", "LED-1", "the refund lands on the total twice")

	return ctx, code
}

// ranCommand runs one orbit command, found in the table the command line
// dispatches on so that a test cannot call something the binary would not.
//
// Not called verb: internal/verb is the package this one asks for what a
// verb means, and a helper by that name shadows it.
func ranCommand(t *testing.T, ctx Context, name string, args ...string) {
	t.Helper()

	for _, c := range commands() {
		if c.Name == name {
			if err := c.Run(ctx, args); err != nil {
				t.Fatalf("orbit %s %v: %v", name, args, err)
			}

			return
		}
	}

	t.Fatalf("orbit has no verb %q", name)
}

// opened is the window, sized, with the board already read into it.
func opened(t *testing.T, ctx Context, dir string) tea.Model {
	t.Helper()

	opts, _, err := window(ctx, dir, "en")
	if err != nil {
		t.Fatalf("build the window: %v", err)
	}

	var m tea.Model = ui.New(opts)

	m = pump(t, m, m.Init())
	m = update(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})

	return m
}

// pump runs a command the window asked for and folds what it answers back
// in, the way the program's own loop does — a batch and all.
//
// A command that does not answer inside promptly is left alone. The window
// starts several clocks on Init — the frame tick, the rescan, the elapsed
// column — and each of them is a sleep before a message. The program waits
// on them because it has a terminal to keep; a test that waited on them
// would be timing the clocks rather than pressing keys.
func pump(t *testing.T, m tea.Model, cmd tea.Cmd) tea.Model {
	t.Helper()

	if cmd == nil {
		return m
	}

	answered := make(chan tea.Msg, 1)

	go func() { answered <- cmd() }()

	select {
	case msg := <-answered:
		switch msg := msg.(type) {
		case nil:
			return m
		case tea.BatchMsg:
			for _, one := range msg {
				m = pump(t, m, one)
			}

			return m
		default:
			return update(t, m, msg)
		}
	case <-time.After(promptly):
		return m
	}
}

// promptly is how long a command has to answer before a test stops waiting
// for it, which is longer than a read of the record and far shorter than any
// of the window's clocks.
const promptly = time.Second

// update hands the window one message and answers the window that came back.
func update(t *testing.T, m tea.Model, msg tea.Msg) tea.Model {
	t.Helper()

	next, _ := m.Update(msg)
	if next == nil {
		t.Fatal("the window answered with nothing")
	}

	return next
}

// press sends the keys of a tape, in order.
func press(t *testing.T, m tea.Model, keys ...tea.KeyPressMsg) tea.Model {
	t.Helper()

	for _, k := range keys {
		m = update(t, m, k)
	}

	return m
}

// typed is one printable key, and named one that has no text.
func typed(s string) tea.KeyPressMsg { return tea.KeyPressMsg{Text: s} }
func named(c rune) tea.KeyPressMsg   { return tea.KeyPressMsg{Code: c} }

// drawn is the frame the window would put on the terminal.
func drawn(t *testing.T, m tea.Model) string {
	t.Helper()

	v, ok := m.(interface{ View() tea.View })
	if !ok {
		t.Fatalf("the window does not draw: %T", m)
	}

	return strings.Join(strings.Fields(ansi.Strip(v.View().Content)), " ")
}

// lines is the frame as the terminal would show it, row by row, so that a
// test can find where something was drawn before it clicks on it.
func lines(t *testing.T, m tea.Model) []string {
	t.Helper()

	v, ok := m.(interface{ View() tea.View })
	if !ok {
		t.Fatalf("the window does not draw: %T", m)
	}

	return strings.Split(ansi.Strip(v.View().Content), "\n")
}

// at is where something is drawn: the cell its first character sits on.
//
// Found in the frame rather than written down as a coordinate. A test that
// clicks on a number is an anchor on the layout — it breaks when the bar
// moves and, worse, goes on passing when the thing it meant to click moved
// somewhere else. Asked this way, the test says the one thing that matters:
// what you can see, you can click, where you see it.
func at(t *testing.T, m tea.Model, needle string) (int, int) {
	t.Helper()

	for y, line := range lines(t, m) {
		if i := strings.Index(line, needle); i >= 0 {
			// Cells and not bytes: the header and the bar are full of
			// emoji, and a byte offset into one of those lines names a
			// column some way to the right of what it meant.
			return lipgloss.Width(line[:i]), y
		}
	}

	t.Fatalf("%q is nowhere in the frame:\n%s", needle, strings.Join(lines(t, m), "\n"))

	return 0, 0
}

// clicked is one whole click on a cell: the button down and up again on the
// same one. mouse.go acts on the release — a press with nothing after it is
// a reader still deciding — so a test that sent only the press would be
// asserting that nothing happens.
func clicked(t *testing.T, m tea.Model, x, y int) tea.Model {
	t.Helper()

	m = update(t, m, tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})

	return update(t, m, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
}

// clickOn finds something in the frame and clicks it.
func clickOn(t *testing.T, m tea.Model, needle string) tea.Model {
	t.Helper()

	x, y := at(t, m, needle)

	return clicked(t, m, x, y)
}
