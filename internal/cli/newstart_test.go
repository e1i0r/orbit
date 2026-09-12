package cli

// `orbit new -run=true`: written down and run, in one command.

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestNewStartsWhatItWroteWhenAskedTo. The window's Save and start button
// builds this command line, and it saved without starting: the task was
// written down, the window went back to the board, and the row sat in to do
// with nobody able to say why.
func TestNewStartsWhatItWroteWhenAskedTo(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	code, out, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-9", "-run=true", "do the thing")
	if code != 0 {
		t.Fatalf("new -run=true exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "started") {
		t.Errorf("new -run=true said %q", out)
	}
}

// TestNewWithoutTheFlagWritesAndStops, which is what it did before the flag
// and what it must go on doing without it.
func TestNewWithoutTheFlagWritesAndStops(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	code, out, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-10", "do the thing")
	if code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	if strings.Contains(out, "started") {
		t.Errorf("new without the flag started something: %q", out)
	}
}
