package ui

// The fenced code block, drawn: which language a fence says it is, and that
// the well keeps its own paper under every colour the theme paints on it.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

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
	keyword := theme.Surface(theme.Sunken).Foreground(theme.Paint(theme.Bad).GetForeground()).Render("return")
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
	probe := theme.Surface(theme.Sunken).Render("x")

	paper, _, _ := strings.Cut(strings.TrimPrefix(probe, "\x1b["), "m")
	if paper == "" || paper == probe {
		t.Fatalf("the sunken surface sets nothing: %q", probe)
	}

	if got, want := strings.Count(row, paper), len(theme.LexCode(line, "go")); got < want {
		t.Errorf("the well names its paper %d times over %d runs: %q", got, want, row)
	}
}

// TestAFenceWithNoLanguageStillReadsItsQuotes. Most of what a model fences is
// output rather than source, and the quotes and the numbers in it are the
// shape it has.
