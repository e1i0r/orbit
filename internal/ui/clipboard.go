package ui

import (
	"context"
	"os/exec"
	"runtime"
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
func clipboardFrom(name string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return "", false
	}

	return string(out), true
}
