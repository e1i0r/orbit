package clip

import (
	"os"
	"path/filepath"
	"runtime"
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
// Read is called from Update — from compose, from the note, from
// the supervisor, from a middle click and from the flow builder — so it runs
// on the thread that draws. It shelled out with exec.Command, which waits
// for as long as the helper takes, and xclip waiting on a selection owner
// that never replies takes for ever. The window stopped rendering and
// stopped answering keys, and the only way out was killing it.
func TestAClipboardThatNeverAnswersDoesNotFreezeTheWindow(t *testing.T) {
	// The helper leaves a child of its own holding the standard output
	// pipe and exits. Killing what was started is not enough to end that
	// read: CI on Linux waited the full five seconds where darwin returned
	// at once, so the deadline has to bound the wait and not only the call.
	helperNamed(t, "wl-paste", "sleep 5 &")

	timeout = 150 * time.Millisecond

	t.Cleanup(func() { timeout = 2 * time.Second })

	done := time.Now()
	out, ok := from("wl-paste")

	took := time.Since(done)
	if took > 2*time.Second {
		t.Errorf("the read took %s; a helper that never answers has to be abandoned at the deadline", took)
	}

	if ok || out != "" {
		t.Errorf("from = (%q, %v), want nothing: a helper that ran out of time said nothing", out, ok)
	}
}

// TestAHelperThatAnsweredIsBelieved is the other half: the deadline must not
// cost the ordinary case, where the helper prints the selection and exits.
func TestAHelperThatAnsweredIsBelieved(t *testing.T) {
	helperNamed(t, "wl-paste", "printf 'pegado'")

	out, ok := from("wl-paste")
	if !ok || out != "pegado" {
		t.Errorf("from = (%q, %v), want (\"pegado\", true)", out, ok)
	}
}

// TestAHelperThatIsNotInstalledIsNotAnAnswer. On a machine with no clipboard
// tool at all every helper fails this way, and Read has to reach the
// end of the list rather than believe the first empty answer.
func TestAHelperThatIsNotInstalledIsNotAnAnswer(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if out, ok := from("wl-paste"); ok || out != "" {
		t.Errorf("from = (%q, %v), want nothing for a helper that is not there", out, ok)
	}
}

// TestWhatIsCopiedReachesTheHelper. A cut takes the text out of the field,
// so a copy that went nowhere while saying it worked is text nobody gets
// back. The helper is handed it on standard input, which is the half of
// this the read tests above never exercise.
func TestWhatIsCopiedReachesTheHelper(t *testing.T) {
	taken := filepath.Join(t.TempDir(), "taken")
	helperNamed(t, "wl-copy", "cat > "+taken)

	if !to("una tarea", "wl-copy") {
		t.Fatal("the helper refused what it was handed")
	}

	got, err := os.ReadFile(taken)
	if err != nil {
		t.Fatalf("read what the helper was given: %v", err)
	}

	if string(got) != "una tarea" {
		t.Errorf("the helper was given %q, want %q", got, "una tarea")
	}
}

// A machine with no clipboard helper on it says so, rather than reporting a
// copy that never happened.
func TestACopyWithNoHelperToTakeItIsNotACopy(t *testing.T) {
	helperNamed(t, "unrelated", "true")

	if to("una tarea", "wl-copy") {
		t.Error("to said yes with no helper installed")
	}
}

// TestTheHelpersAreTriedInTheOrderThisMachineIsLikelyToHaveThem.
//
// On darwin pbpaste is the answer and an empty one is still the answer:
// neither of the others is installed there, so falling through to them only
// spends two more process spawns to be told so twice. Everywhere else
// Wayland is tried before X11, and the first helper that answers wins.
func TestTheHelpersAreTriedInTheOrderThisMachineIsLikelyToHaveThem(t *testing.T) {
	if runtime.GOOS == "darwin" {
		helperNamed(t, "pbpaste", "printf 'from the mac'")

		if got := Read(); got != "from the mac" {
			t.Errorf("Read() = %q, want what pbpaste answered", got)
		}

		return
	}

	helperNamed(t, "wl-paste", "printf 'from wayland'")

	if got := Read(); got != "from wayland" {
		t.Errorf("Read() = %q, want what wl-paste answered", got)
	}
}

// TestACopyThatWentNowhereSaysSo. Nothing else in the window can tell
// whether a helper is on this machine, and a copy that quietly went nowhere
// is a copy the reader will paste from somewhere else and lose.
func TestACopyThatWentNowhereSaysSo(t *testing.T) {
	// A PATH with no helper on it at all: every one of the three is
	// missing, and Write says so rather than claiming the text was taken.
	t.Setenv("PATH", t.TempDir())

	if Write("something") {
		t.Error("a machine with no clipboard helper said it took the text")
	}

	if got := Read(); got != "" {
		t.Errorf("a machine with no clipboard helper read %q", got)
	}
}

// TestAHelperThatTakesItSaysSo.
func TestAHelperThatTakesItSaysSo(t *testing.T) {
	name := "wl-copy"
	if runtime.GOOS == "darwin" {
		name = "pbcopy"
	}

	took := filepath.Join(t.TempDir(), "took")
	helperNamed(t, name, "cat > "+took)

	if !Write("what the reader selected") {
		t.Fatal("a helper that took the text said it had not")
	}

	got, err := os.ReadFile(took)
	if err != nil {
		t.Fatalf("the helper wrote nothing: %v", err)
	}

	if string(got) != "what the reader selected" {
		t.Errorf("the helper was handed %q", got)
	}
}
