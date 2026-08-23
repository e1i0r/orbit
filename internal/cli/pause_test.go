package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// control is what a reader would cat to see what they asked a run for.
func control(t *testing.T, orbitHome string) string {
	t.Helper()
	body, err := os.ReadFile(findFile(t, orbitHome, "control"))
	if err != nil {
		t.Fatalf("read the control word: %v", err)
	}
	return string(body)
}

func TestPauseLeavesOneWordARunWillFind(t *testing.T) {
	root, orbitHome := workspace(t)
	dir := writeTask(t, root)

	code, out, errOut := run(t, "pause", "-repo", dir, "ACME-1")
	if code != 0 {
		t.Fatalf("pause exited %d: %s", code, errOut)
	}
	if got := control(t, orbitHome); got != "pause\n" {
		t.Errorf("the control file holds %q, want %q", got, "pause\n")
	}
	// It has not paused anything yet, and saying so would be a claim about
	// a run that may be in the middle of an hour-long phase.
	if strings.Contains(out, "paused") {
		t.Errorf("pause said %q, which claims more than it did", out)
	}
	if !strings.Contains(out, "ACME-1") || !strings.Contains(out, "pause") {
		t.Errorf("pause said %q, which does not say what was asked of what", out)
	}
}

// resume writes through the same door with a different word, which is the
// whole of the difference between the two commands.
func TestResumeLeavesTheOtherWord(t *testing.T) {
	root, orbitHome := workspace(t)
	dir := writeTask(t, root)

	if code, _, errOut := run(t, "pause", "-repo", dir, "ACME-1"); code != 0 {
		t.Fatalf("pause exited %d: %s", code, errOut)
	}
	if code, _, errOut := run(t, "resume", "-repo", dir, "ACME-1"); code != 0 {
		t.Fatalf("resume exited %d: %s", code, errOut)
	}
	if got := control(t, orbitHome); got != "resume\n" {
		t.Errorf("the control file holds %q, want %q", got, "resume\n")
	}
}

// A word left for a task nobody is running is not a mistake — it waits for
// the run that starts next — so neither command asks whether one is live.
func TestPausingATaskNoRunHoldsIsNotARefusal(t *testing.T) {
	root, orbitHome := workspace(t)
	dir := writeTask(t, root)
	if code, _, errOut := run(t, "pause", "-repo", dir, "ACME-1"); code != 0 {
		t.Fatalf("pause exited %d: %s", code, errOut)
	}
	if _, err := os.Stat(findFile(t, orbitHome, "control")); err != nil {
		t.Errorf("the word was not kept: %v", err)
	}
}

func TestPauseAndResumeNeedAnID(t *testing.T) {
	for _, word := range []string{"pause", "resume"} {
		root, _ := workspace(t)
		dir := filepath.Join(root, "payments")
		code, _, errOut := run(t, word, "-repo", dir)
		if code == 0 {
			t.Fatalf("%s with no id exited 0", word)
		}
		if !strings.Contains(errOut, word+" needs the id of a task") {
			t.Errorf("the refusal is %q, and does not name the command that refused", errOut)
		}
	}
}

// The flag set is named after the word, so each command's help shows its own
// line out of the synopsis. A single `pause|resume` row would match neither.
func TestEachOfTheTwoShowsItsOwnLine(t *testing.T) {
	for _, tc := range []struct{ word, says string }{
		{"pause", "stop a run at its next phase"},
		{"resume", "let a stopped run carry on"},
	} {
		t.Setenv("ORBIT_HOME", t.TempDir())
		code, out, errOut := run(t, tc.word, "-h")
		if code != 0 {
			t.Fatalf("%s -h exited %d: %s", tc.word, code, errOut)
		}
		if !strings.Contains(out, tc.says) {
			t.Errorf("%s -h does not say what it does:\n%s", tc.word, out)
		}
		if !strings.Contains(out, "-repo") {
			t.Errorf("%s -h does not show its flags:\n%s", tc.word, out)
		}
	}
}
