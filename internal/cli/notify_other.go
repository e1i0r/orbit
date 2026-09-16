//go:build !darwin

package cli

import "os/exec"

// notifier is the command this desktop shows a notification with, and
// nothing when it has none.
//
// notify-send is the freedesktop one and is what a Linux desktop has; a
// server has neither it nor anybody to read what it would show, and
// answering nothing is the whole handling of that case.
func notifier(title, text string) (string, []string) {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return "", nil
	}

	return "notify-send", []string{title, text}
}
