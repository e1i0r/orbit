package ui

import (
	"os/exec"
	"runtime"
)

// readClipboard queries the system clipboard across macOS, Wayland, and X11.
func readClipboard() string {
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("pbpaste").Output(); err == nil {
			return string(out)
		}
	}
	if out, err := exec.Command("wl-paste").Output(); err == nil {
		return string(out)
	}
	if out, err := exec.Command("xclip", "-out", "-selection", "clipboard").Output(); err == nil {
		return string(out)
	}
	return ""
}
