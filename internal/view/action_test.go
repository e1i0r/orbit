package view

// The line beside a running task: which argument of a tool call it shows,
// and where it is cut.

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestAnActionIsTheArgumentAndNotTheDocument.
//
// Searching the arguments for `"command"` and then for `:"` wants the value
// to start immediately after the colon. Every JSON encoder that indents puts
// a space there, so the key matches, the value does not, and the row shows
// the whole document instead of the command in it.
func TestAnActionIsTheArgumentAndNotTheDocument(t *testing.T) {
	for _, c := range []struct {
		why  string
		args string
		want string
	}{
		{
			"a space after the colon is what an encoder writes",
			`{"command": "go test ./...", "timeout": 120}`,
			"Bash: go test ./...",
		},
		{
			"and one without it still works",
			`{"command":"go test ./..."}`,
			"Bash: go test ./...",
		},
		{
			"a file path arrives as it was written, not as it was escaped",
			`{"file_path":"C:\\Users\\ana\\main.go"}`,
			`Bash: C:\Users\ana\main.go`,
		},
		{
			"a quotation mark inside the value does not cut it short",
			`{"command":"echo \"hello there\""}`,
			`Bash: echo "hello there"`,
		},
		{
			"the second key is looked for when the first is absent",
			`{"offset": 1, "path": "internal/view/fold.go"}`,
			"Bash: internal/view/fold.go",
		},
		{
			"the pattern a search is for is one of them too",
			`{"pattern":"needle","-i":true}`,
			"Bash: needle",
		},
		{
			"a call about none of them shows what it was given",
			`{"style":"table"}`,
			`Bash: {"style":"table"}`,
		},
		{
			"arguments that are not JSON are shown as they stand",
			"go test ./...",
			"Bash: go test ./...",
		},
		{
			"a call with no arguments at all is just its name",
			"",
			"Bash",
		},
	} {
		if got := formatAction("Bash", c.args); got != c.want {
			t.Errorf("formatAction(%s) = %q, want %q — %s", c.args, got, c.want, c.why)
		}
	}
}

// TestAnActionIsCutBetweenCharacters. head[:47] is a byte offset: a
// character on that boundary is left as half of itself, and what a terminal
// draws for half a character is up to the terminal.
func TestAnActionIsCutBetweenCharacters(t *testing.T) {
	long := strings.Repeat("ñ", 80)

	got := formatAction("Bash", long)
	if !utf8.ValidString(got) {
		t.Fatalf("formatAction cut a character in half: %q", got)
	}

	cut := strings.TrimPrefix(got, "Bash: ")
	if n := utf8.RuneCountInString(cut); n != actionChars {
		t.Errorf("the action is %d characters long, want %d", n, actionChars)
	}

	if !strings.HasSuffix(cut, "…") {
		t.Errorf("the action reads %q, want it to end in an ellipsis", cut)
	}

	// And one that fits is not touched.
	if got := formatAction("Bash", "ls"); got != "Bash: ls" {
		t.Errorf("formatAction = %q, want a short action left alone", got)
	}
}

// TestAnActionNamesTheFileAndNotTheWorktree.
//
// Every path an engine reports inside a run begins with the worktree it was
// given — ~/.orbit/worktrees/<repo>/<task>/ — and that prefix is the same on
// every row of the run. It is also long enough that cutting the action to
// the measure cuts away the one part that differs: the file.
func TestAnActionNamesTheFileAndNotTheWorktree(t *testing.T) {
	for _, c := range []struct {
		why  string
		args string
		want string
	}{
		{
			"the worktree a run was given is the same on every row",
			`{"file_path":"/Users/ana/.orbit/worktrees/5a14f7401345/FRA-62/internal/ui/repoargs_test.go"}`,
			"Bash: internal/ui/repoargs_test.go",
		},
		{
			"a state directory somewhere else is still a worktree",
			`{"path":"/var/tmp/state/worktrees/02c3a714b58d/ACME-1/go.mod"}`,
			"Bash: go.mod",
		},
		{
			"a path that is not under a worktree is left as it stands",
			`{"file_path":"/etc/hosts"}`,
			"Bash: /etc/hosts",
		},
		{
			"and the worktree root itself still names itself",
			`{"path":"/Users/ana/.orbit/worktrees/5a14f7401345/FRA-62"}`,
			"Bash: /Users/ana/.orbit/worktrees/5a14f7401345/FRA-62",
		},
	} {
		if got := formatAction("Bash", c.args); got != c.want {
			t.Errorf("formatAction(%s) = %q, want %q — %s", c.args, got, c.want, c.why)
		}
	}
}

// TestAnActionOnMoreThanOneLineIsJoined.
//
// A shell command written over several lines is one command. Stopping at
// the first line break leaves the row reading `grep -rn \`, which is the
// same row for every grep in the run and says nothing about any of them.
func TestAnActionOnMoreThanOneLineIsJoined(t *testing.T) {
	for _, c := range []struct {
		why  string
		args string
		want string
	}{
		{
			"a continued line carries on, and the backslash goes",
			"{\"command\":\"grep -rn \\\\\\n  needle .\"}",
			"Bash: grep -rn needle .",
		},
		{
			"lines that are separate commands are still one row",
			"{\"command\":\"cd internal\\ngo test ./...\"}",
			"Bash: cd internal go test ./...",
		},
	} {
		if got := formatAction("Bash", c.args); got != c.want {
			t.Errorf("formatAction(%s) = %q, want %q — %s", c.args, got, c.want, c.why)
		}
	}
}
