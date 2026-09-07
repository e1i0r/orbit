package markdown

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestRenderMarkdownRichAndRaw(t *testing.T) {
	md := `# Main Title
## Subtitle
### Section
> Important quote
- [x] Done task
- [ ] Todo task
- Bullet item
1. Numbered item
---
` + "```go\nfmt.Println(\"hello\")\n```"

	// 1. Rich formatting (raw = false)
	rich := Render(md, 80, false)
	if len(rich) == 0 {
		t.Fatal("Render rich returned empty")
	}

	joined := strings.Join(rich, "\n")
	if !strings.Contains(joined, "Main Title") || !strings.Contains(joined, "Subtitle") {
		t.Errorf("expected headings in rendered markdown, got:\n%s", joined)
	}

	if !strings.Contains(joined, "✔") || !strings.Contains(joined, "☐") {
		t.Errorf("expected checklist symbols in rendered markdown, got:\n%s", joined)
	}

	// 2. Raw formatting (raw = true)
	raw := Render(md, 80, true)
	if len(raw) == 0 {
		t.Fatal("Render raw returned empty")
	}

	rawJoined := strings.Join(raw, "\n")
	if !strings.Contains(rawJoined, "# Main Title") {
		t.Errorf("expected raw markdown '# Main Title', got:\n%s", rawJoined)
	}
}

// TestAWrappedItemIsStillOneItem. A bullet is the mark that one item began;
// putting it on every row a long item wrapped onto turns a sentence into a
// list of the fragments it was broken into.
func TestAWrappedItemIsStillOneItem(t *testing.T) {
	long := "- " + strings.Repeat("a reason this had to be done this way and not the other, ", 4)

	rows := Render(long, 90, false)
	if len(rows) < 2 {
		t.Fatalf("the item was drawn as %d rows, want it wrapped", len(rows))
	}

	for i, r := range rows {
		if got := strings.Count(ansi.Strip(r), "•"); (i == 0) != (got == 1) {
			t.Errorf("row %d has %d bullets: %q", i, got, ansi.Strip(r))
		}
	}

	if !strings.HasPrefix(ansi.Strip(rows[1]), Indent+"  ") {
		t.Errorf("the second row is %q, want it hanging under the first", ansi.Strip(rows[1]))
	}
}

// TestMarkdownIsSetToTheMeasure. The panes are as wide as the terminal and
// the eye is not: a line of 140 cells loses its reader at every break.
func TestMarkdownIsSetToTheMeasure(t *testing.T) {
	md := "# A title long enough that it would run past the measure if nothing cut it, and then some\n" +
		strings.Repeat("prose that keeps going and going. ", 12) + "\n" +
		"> a quotation that is also much longer than the measure allows for, several times over\n" +
		"- an item that is likewise far too long to sit on one line of a pane this wide, and more"

	ceiling := lipgloss.Width(Indent) + Measure

	for _, r := range Render(md, 200, false) {
		if got := lipgloss.Width(r); got > ceiling {
			t.Errorf("a row is %d cells wide, want at most %d: %q", got, ceiling, ansi.Strip(r))
		}
	}
}

// TestACodeWellIsSquare. The well is a shape, and a shape with a ragged edge
// reads as damage. Tabs are what makes it ragged: one character, four
// columns, and padded before the terminal widens them.
func TestACodeWellIsSquare(t *testing.T) {
	md := "```go\nfunc f() {\n\tif x {\n\t\treturn\n\t}\n}\n```"

	rows := Render(md, 100, false)
	if len(rows) != 6 {
		t.Fatalf("the fence was drawn as %d rows, want a language and five of code", len(rows))
	}

	want := lipgloss.Width(rows[1])
	for _, r := range rows[1:] {
		if got := lipgloss.Width(r); got != want {
			t.Errorf("a row of the well is %d cells wide, want %d: %q", got, want, ansi.Strip(r))
		}
	}

	if !strings.Contains(ansi.Strip(rows[2]), codeTab+"if x {") {
		t.Errorf("the tab was not opened out: %q", ansi.Strip(rows[2]))
	}
}

// TestANumberedItemNeedsANumber, or every sentence with a full stop in the
// middle of it becomes a list.
func TestANumberedItemNeedsANumber(t *testing.T) {
	for _, c := range []struct {
		line string
		list bool
	}{
		{"1. the first thing", true},
		{"12. the twelfth thing", true},
		{"e.g. this is prose", false},
		{"Fig. 4 shows it", false},
		{"1.no space, no list", false},
	} {
		if _, got := listItem(c.line); got != c.list {
			t.Errorf("listItem(%q) = %v, want %v", c.line, got, c.list)
		}
	}
}

// TestProseKeepsItsColourAcrossASpan. A style rendered inside another closes
// with a reset that ends the outer one too, so a paragraph painted once
// around an inline span goes back to the terminal's own foreground halfway
// through the sentence.
func TestProseKeepsItsColourAcrossASpan(t *testing.T) {
	got := Inline("plain `code` plain **bold** and the tail")

	for _, run := range strings.Split(got, "\x1b[m") {
		if run != "" && !strings.HasPrefix(run, "\x1b[") {
			t.Errorf("%q is drawn with no colour of its own", run)
		}
	}
}
