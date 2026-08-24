package cli

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// Asking a program what it takes is not a failure. Both subcommands here go
// through the same parse, and both return before openBoth, so no state root
// is touched and `run -h` never reaches an engine.

func TestTheHelpFlagPrintsTheFlagsAndSucceeds(t *testing.T) {
	for _, cmd := range []string{"new", "run"} {
		t.Setenv("ORBIT_HOME", t.TempDir())
		code, out, errOut := run(t, cmd, "-h")
		if code != 0 {
			t.Errorf("%s -h exited %d, want 0: %s", cmd, code, errOut)
		}
		if errOut != "" {
			t.Errorf("%s -h wrote to stderr: %s", cmd, errOut)
		}
		if !strings.Contains(out, "-repo") {
			t.Errorf("%s -h does not show its flags:\n%s", cmd, out)
		}
		if !strings.Contains(out, "orbit "+cmd) {
			t.Errorf("%s -h does not show the shape of the command:\n%s", cmd, out)
		}
	}
}

func TestAMistypedFlagPrintsTheErrorAndTheFlags(t *testing.T) {
	for _, tc := range []struct{ cmd, bad string }{
		{"list", "-repos"},
		{"show", "-r"},
	} {
		t.Setenv("ORBIT_HOME", t.TempDir())
		code, _, errOut := run(t, tc.cmd, tc.bad, ".")
		if code == 0 {
			t.Errorf("%s %s exited 0", tc.cmd, tc.bad)
		}
		if !strings.Contains(errOut, tc.bad) {
			t.Errorf("the error does not name the flag that was wrong:\n%s", errOut)
		}
		if !strings.Contains(errOut, "-repo string") {
			t.Errorf("%s %s says what was wrong and then offers nothing:\n%s", tc.cmd, tc.bad, errOut)
		}
	}
}

// TestTheUsageTableLinesUp reads the columns rather than the spaces. The
// table was aligned by hand and the `new` line is longer than the rest, so
// its description started three columns to the right of everyone else's. It
// is printed from the command table now, and a translated description is a
// width nobody can count in advance — which is the same failure with more
// ways to reach it.
func TestTheUsageTableLinesUp(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	_, out, _ := run(t, "help")
	p := words.For("")
	col := -1
	for _, c := range commands() {
		line := ""
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, c.Usage()) {
				line = l
			}
		}
		if line == "" {
			t.Fatalf("usage does not list %q:\n%s", c.Usage(), out)
		}
		at := strings.Index(line, c.About(p))
		if at < 0 {
			t.Fatalf("usage does not say what %q does:\n%s", c.Usage(), out)
		}
		if col == -1 {
			col = at
			continue
		}
		if at != col {
			t.Errorf("%q describes itself at column %d, the others at %d:\n%s", c.Usage(), at, col, out)
		}
	}
}
