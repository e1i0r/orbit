//go:build darwin

package cli

import "strings"

// notifier is the command macOS shows a notification with.
//
// osascript, which is in every install, rather than a helper somebody has to
// install first. The text is passed as a script, so both halves are quoted:
// a task's own title is a sentence somebody typed and it will one day have a
// quotation mark in it.
func notifier(title, text string) (string, []string) {
	script := "display notification " + quoted(text) + " with title " + quoted(title)

	return "osascript", []string{"-e", script}
}

// quoted is one string as AppleScript reads one.
//
// Both halves are quoted and both are escaped: a task's own title is a
// sentence somebody typed, and it will one day have a quotation mark in it.
func quoted(text string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(text) + `"`
}
