package view

// The line beside a running task: which argument of a tool call it shows,
// and where it is cut.

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestAnActionIsTheArgumentAndNotTheDocument is the regression.
//
// The arguments used to be searched for `"command"` and then for `:"`, which
// wants the value to start immediately after the colon. Every JSON encoder
// that indents puts a space there, so the key matched, the value did not,
// and the row showed the whole document instead of the command in it.
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
			"a quotation mark inside the value used to cut it short",
			`{"command":"echo \"hello there\""}`,
			`Bash: echo "hello there"`,
		},
		{
			"the second key is looked for when the first is absent",
			`{"offset": 1, "path": "internal/view/fold.go"}`,
			"Bash: internal/view/fold.go",
		},
		{
			"a call about none of them shows what it was given",
			`{"pattern":"needle"}`,
			`Bash: {"pattern":"needle"}`,
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

// TestAnActionIsCutBetweenCharacters. The cut used to be head[:47], which is
// a byte offset. A character on that boundary is left as half of itself, and
// what a terminal draws for half a character is up to the terminal.
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
