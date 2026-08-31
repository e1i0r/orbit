package ui

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// clipboardTimeout is how long one clipboard helper is given before the
// window gives up on it.
//
// Every caller of readClipboard is inside Update, so the read happens on the
// thread that draws: while it is out, nothing renders and no key is
// answered. exec.Command carries no deadline, and the two helpers below are
// exactly the kind that hang rather than fail — wl-paste with no compositor
// answering, xclip waiting on an X server that is gone or on a selection
// owner that never replies. A paste on a machine in that state froze the
// whole window, with no way out of it but killing the process.
//
// It is a variable so a test can shorten it, the way internal/tracker makes
// Linear's endpoint one.
var clipboardTimeout = 2 * time.Second

// readClipboard queries the system clipboard across macOS, Wayland, and X11.
func readClipboard() string {
	// On darwin pbpaste is the answer, and an empty one is still the
	// answer: neither helper below is installed there, so falling through
	// to them only spends two more process spawns to be told so twice.
	if runtime.GOOS == "darwin" {
		out, _ := clipboardFrom("pbpaste")

		return out
	}

	if out, ok := clipboardFrom("wl-paste"); ok {
		return out
	}

	out, _ := clipboardFrom("xclip", "-out", "-selection", "clipboard")

	return out
}

// clipboardFrom runs one clipboard helper under that deadline.
//
// It reports false when the helper is not installed, failed, or ran out of
// time — the three cases the caller answers the same way, by asking the next
// helper along.
//
// What is bounded is the wait and not only the call, the way internal/ui
// bounds a repository's base branch lookup. exec.CommandContext kills the
// child it started, and that is not always enough: Output waits for the
// standard output pipe to close, and a helper that leaves anything of its
// own behind — a daemon it forked, a `sleep &` in a wrapper script — leaves
// that pipe open, so the read went on waiting for a process nobody had been
// told to stop. It waited the full five seconds under CI on Linux while the
// same code returned at the deadline on darwin, which is the shape of a
// guarantee that is not one.
//
// The disclosed cost is the same one boundedBaseOf discloses: a helper that
// never returns leaves its goroutine, and whatever it forked, running until
// it does. The channel is buffered, so the goroutine ends either way, and
// the caller is a keystroke rather than a two-second tick.
func clipboardFrom(name string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()

	type answer struct {
		out []byte
		err error
	}

	got := make(chan answer, 1)

	cmd := exec.CommandContext(ctx, name, args...)
	// WaitDelay bounds the second wait, the one for the pipes. Output does
	// not return when the process ends; it returns when standard output
	// reaches its end, and a helper's own child holds that open after the
	// helper is gone. Without this the goroutine below outlives the window's
	// interest in it for as long as that child runs.
	cmd.WaitDelay = clipboardTimeout

	go func() {
		out, err := cmd.Output()
		got <- answer{out: out, err: err}
	}()

	select {
	case a := <-got:
		if a.err != nil {
			return "", false
		}

		return string(a.out), true
	case <-ctx.Done():
		return "", false
	}
}

// writeClipboard puts one string on the system clipboard, through the same
// three platforms readClipboard reads it back from.
//
// It reports whether a helper took it. Nothing else in the window can tell
// whether pbcopy is on this machine, and a copy that quietly went nowhere
// is a copy the reader will paste from somewhere else and lose.
func writeClipboard(text string) bool {
	if runtime.GOOS == "darwin" {
		return clipboardTo(text, "pbcopy")
	}

	if clipboardTo(text, "wl-copy") {
		return true
	}

	return clipboardTo(text, "xclip", "-in", "-selection", "clipboard")
}

// clipboardTo hands the text to one helper on its standard input, under the
// deadline reading is given and for the same reason: this runs inside
// Update, so a helper that never returns is a window that never draws
// again.
func clipboardTo(text, name string, args ...string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(text)
	cmd.WaitDelay = clipboardTimeout

	done := make(chan error, 1)

	go func() { done <- cmd.Run() }()

	select {
	case err := <-done:
		return err == nil
	case <-ctx.Done():
		return false
	}
}
