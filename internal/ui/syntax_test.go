package ui

// Colour inside a code well. What a reader looks for in a block somebody
// else's model wrote is its shape — where the strings are, which line is a
// comment — and one flat colour on darker paper carries none of it.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// parts is what the lexer made of a line, as the runs it found.
func parts(t *testing.T, line, family string) map[codePart][]string {
	t.Helper()

	out := map[codePart][]string{}
	for _, tok := range lexCode(line, family) {
		out[tok.part] = append(out[tok.part], tok.text)
	}

	return out
}

// has says whether one of the runs of that part is exactly want.
func has(got map[codePart][]string, part codePart, want string) bool {
	for _, s := range got[part] {
		if s == want {
			return true
		}
	}

	return false
}

// TestThreeVolumesOfWord. The keyword, the type and the named literal are
// three sets and not one because a block that shouts all of them equally is a
// block with no shape.
func TestThreeVolumesOfWord(t *testing.T) {
	got := parts(t, `func run(n int) bool { return n > 0 && ok != nil }`, "go")

	for part, want := range map[codePart]string{
		codeKeyword: "func", codeType: "int", codeConst: "nil", codeNumber: "0",
	} {
		if !has(got, part, want) {
			t.Errorf("%q was not read as part %d: %v", want, part, got)
		}
	}

	// An identifier the language did not name is the well's own ink: a
	// vocabulary that also held every function a language ships would paint
	// most of a line loud and say nothing by it.
	if !has(got, codePlain, " run(n ") {
		t.Errorf("the names the line invented were not left plain: %v", got)
	}

	// Each of the four is painted, and each in its own colour, or the sets
	// are three names for one thing.
	seen := map[Role]bool{}

	for _, part := range []codePart{codeKeyword, codeType, codeConst, codeComment} {
		role, painted := codeRole(part)
		if !painted {
			t.Errorf("part %d is drawn in the well's own ink, so it is not told apart", part)
		}

		if seen[role] {
			t.Errorf("part %d shares its colour with a part above it", part)
		}

		seen[role] = true
	}
}

// TestAWordInsideAWordIsNotAWord. A keyword found in the middle of an
// identifier is a block where half the names are painted as syntax.
func TestAWordInsideAWordIsNotAWord(t *testing.T) {
	got := parts(t, "iffy := myint + sha256sum", "go")

	for _, part := range []codePart{codeKeyword, codeType, codeNumber} {
		if len(got[part]) > 0 {
			t.Errorf("part %d was found inside an identifier: %v", part, got[part])
		}
	}
}

// TestACommentTakesTheRestOfTheLine, whatever is written in it: a quote or a
// brace inside a comment is prose, and reading it as syntax is how the colour
// of a block comes apart halfway down.
func TestACommentTakesTheRestOfTheLine(t *testing.T) {
	got := parts(t, `x := 1 // set x to "one", func, nil`, "go")

	if !has(got, codeComment, `// set x to "one", func, nil`) {
		t.Errorf("the comment did not take the rest of the line: %v", got)
	}

	if len(got[codeString]) > 0 || len(got[codeKeyword]) > 0 {
		t.Errorf("what was written in the comment was read as syntax: %v", got)
	}

	// The opener is the family's own. A hash is a comment in a shell and an
	// anchor in a Go string, and one table for both would grey out half the
	// Go on the pane.
	if hash := parts(t, `n := 5 # not a comment here`, "go"); len(hash[codeComment]) > 0 {
		t.Errorf("a shell comment opened a Go line: %v", hash)
	}

	if hash := parts(t, `echo one # and the rest`, "shell"); !has(hash, codeComment, "# and the rest") {
		t.Errorf("the shell comment was not read: %v", hash)
	}
}

// TestAStringHoldsWhatItQuotes. An escaped quote does not close a string, and
// a string never closed runs to the end of the line — which is the whole of
// what a lexer reading one line at a time can honestly say about it.
func TestAStringHoldsWhatItQuotes(t *testing.T) {
	escaped := parts(t, `a := "one \" two" + b`, "go")
	if !has(escaped, codeString, `"one \" two"`) {
		t.Errorf("the escaped quote closed the string early: %v", escaped)
	}

	if !has(escaped, codePlain, " + b") {
		t.Errorf("what came after the string was taken into it: %v", escaped)
	}

	open := parts(t, `a := "never closed`, "go")
	if !has(open, codeString, `"never closed`) {
		t.Errorf("the unclosed string did not run to the end of the line: %v", open)
	}

	// A backquoted string in Go holds its backslashes: reading an escape in
	// one is how a raw regexp ends up unterminated.
	raw := parts(t, "p := `one \\` + rest", "go")
	if !has(raw, codeString, "`one \\`") {
		t.Errorf("a backslash closed a raw string early: %v", raw)
	}
}

// TestAFenceSaysWhatToReadAndTheNextOneForgetsIt. The language belongs to the
// block it was written on; carried past the closing fence it would colour
// plain output as somebody's Go.
func TestAFenceSaysWhatToReadAndTheNextOneForgetsIt(t *testing.T) {
	rows := renderMarkdown("```go\nreturn nil\n```\n\n```\nreturn nil\n```\n", 60, false)

	var painted, plain string

	for _, r := range rows {
		switch {
		case !strings.Contains(ansi.Strip(r), "return nil"):
		case painted == "":
			painted = r
		default:
			plain = r
		}
	}

	if painted == "" || plain == "" {
		t.Fatalf("the two blocks were not both drawn:\n%s", strings.Join(rows, "\n"))
	}

	// The keyword is asked for by role, so what it should look like is
	// composed here the same way the well composes it.
	keyword := Surface(Sunken).Foreground(Paint(Bad).GetForeground()).Render("return")
	if !strings.Contains(painted, keyword) {
		t.Errorf("the Go fence did not paint its keyword: %q", painted)
	}

	if strings.Contains(plain, keyword) {
		t.Errorf("the fence with no language was read as the Go before it: %q", plain)
	}
}

// TestTheWellKeepsItsPaperUnderEveryColour. A style rendered inside another
// closes with a reset, so a token painted on the well would take the rest of
// the row back to the window's own paper — a block with a bite out of it.
func TestTheWellKeepsItsPaperUnderEveryColour(t *testing.T) {
	const w = 60

	line := `if n > 0 { // done`

	row := codeWell(line, "go", w)

	if got := lipgloss.Width(ansi.Strip(row)); got != w {
		t.Errorf("a row of the well is %d cells wide, want %d: %q", got, w, ansi.Strip(row))
	}

	// The paper is named once per run of the line, and the runs are what the
	// lexer found: a count short of that is a run drawn on the window's own
	// paper. What to look for is taken from the surface rather than written
	// out, so the check is about the well and not about one theme's hex.
	probe := Surface(Sunken).Render("x")

	paper, _, _ := strings.Cut(strings.TrimPrefix(probe, "\x1b["), "m")
	if paper == "" || paper == probe {
		t.Fatalf("the sunken surface sets nothing: %q", probe)
	}

	if got, want := strings.Count(row, paper), len(lexCode(line, "go")); got < want {
		t.Errorf("the well names its paper %d times over %d runs: %q", got, want, row)
	}
}

// TestAFenceWithNoLanguageStillReadsItsQuotes. Most of what a model fences is
// output rather than source, and the quotes and the numbers in it are the
// shape it has.
func TestAFenceWithNoLanguageStillReadsItsQuotes(t *testing.T) {
	got := parts(t, `level="warn" tries=3 msg=ok`, codeFamily("whatever-this-is"))

	if !has(got, codeString, `"warn"`) {
		t.Errorf("the quoted value was not read: %v", got)
	}

	if !has(got, codeNumber, "3") {
		t.Errorf("the number was not read: %v", got)
	}

	if len(got[codeKeyword]) > 0 {
		t.Errorf("a language nothing knows named a keyword anyway: %v", got)
	}
}

// TestEveryAliasAModelTypesIsRead. The name on a fence is whatever the model
// wrote, and a fence whose language is not recognised is a block with no
// shape at all.
func TestEveryAliasAModelTypesIsRead(t *testing.T) {
	for alias, want := range map[string]string{
		"Go": "go", "golang": "go", "bash": "shell", "console": "shell",
		"YAML": "data", "jsonc": "data", "py": "python", "tsx": "js", "sql": "sql",
		"": "", "brainfuck": "",
	} {
		if got := codeFamily(alias); got != want {
			t.Errorf("codeFamily(%q) = %q, want %q", alias, got, want)
		}
	}
}
