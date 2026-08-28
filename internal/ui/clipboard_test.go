package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// helperNamed puts an executable of that name, running that shell script, at
// the front of PATH for the length of the test.
//
// The two system directories stay on the path behind it because the scripts
// below call ordinary commands and a PATH holding only the temp directory
// leaves them unable to find one — a fake meant to hang that instead exits
// at once, and a test that proves nothing while passing.
func helperNamed(t *testing.T, name, script string) {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script+"\n"), 0o700); err != nil {
		t.Fatalf("write the fake %s: %v", name, err)
	}

	sep := string(os.PathListSeparator)
	t.Setenv("PATH", dir+sep+"/bin"+sep+"/usr/bin")
}

// TestAClipboardThatNeverAnswersDoesNotFreezeTheWindow.
//
// readClipboard is called from Update — from compose, from the note, from
// the supervisor, from a middle click and from the flow builder — so it runs
// on the thread that draws. It shelled out with exec.Command, which waits
// for as long as the helper takes, and xclip waiting on a selection owner
// that never replies takes for ever. The window stopped rendering and
// stopped answering keys, and the only way out was killing it.
func TestAClipboardThatNeverAnswersDoesNotFreezeTheWindow(t *testing.T) {
	helperNamed(t, "wl-paste", "sleep 5")

	clipboardTimeout = 150 * time.Millisecond

	t.Cleanup(func() { clipboardTimeout = 2 * time.Second })

	done := time.Now()
	out, ok := clipboardFrom("wl-paste")

	took := time.Since(done)
	if took > 2*time.Second {
		t.Errorf("the read took %s; a helper that never answers has to be abandoned at the deadline", took)
	}

	if ok || out != "" {
		t.Errorf("clipboardFrom = (%q, %v), want nothing: a helper that ran out of time said nothing", out, ok)
	}
}

// TestAHelperThatAnsweredIsBelieved is the other half: the deadline must not
// cost the ordinary case, where the helper prints the selection and exits.
func TestAHelperThatAnsweredIsBelieved(t *testing.T) {
	helperNamed(t, "wl-paste", "printf 'pegado'")

	out, ok := clipboardFrom("wl-paste")
	if !ok || out != "pegado" {
		t.Errorf("clipboardFrom = (%q, %v), want (\"pegado\", true)", out, ok)
	}
}

// TestAHelperThatIsNotInstalledIsNotAnAnswer. On a machine with no clipboard
// tool at all every helper fails this way, and readClipboard has to reach the
// end of the list rather than believe the first empty answer.
func TestAHelperThatIsNotInstalledIsNotAnAnswer(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if out, ok := clipboardFrom("wl-paste"); ok || out != "" {
		t.Errorf("clipboardFrom = (%q, %v), want nothing for a helper that is not there", out, ok)
	}
}
