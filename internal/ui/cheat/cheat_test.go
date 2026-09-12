package cheat

// The cheat sheet, drawn twice, once per language this repository ships.
//
// Every line of it is compared against its opposite number. A line that
// reads the same in Spanish as in English is a line whose words never
// reached the catalogue — and that is not a hypothetical: this whole screen
// was written out in Spanish, literal by literal, under a window whose badge
// said EN, and the only string on it that answered to the language switch
// was the "scroll · back" footer. Nothing in the suite noticed, because
// internal/words can only hold to account the strings that call it.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/words"
)

// world is the Env in one language, with a verb and a tab of its own: the
// two lists the sheet is handed rather than keeping.
func world(t *testing.T, p *words.Printer) Env {
	t.Helper()

	return Env{
		Words: p,
		Keys:  keymap.New(p),
		// The build the masthead names. The language comparison below
		// skips the masthead's rows by these exact lines, so the version
		// here is part of what that exemption names.
		Version: "dev",
		// Both lists arrive already in the reader's language: they are the
		// window's to translate, and what this test is about is every other
		// line on the sheet.
		Verbs: []Verb{{Key: "r", Says: p.T("help.board.run", "Start a new run of the selected task")}},
		Tabs: []Tab{{
			Glyph:  "1",
			Title:  p.T("tab.overview", "overview"),
			Detail: p.T("tab_desc.overview", "general status, live activity and metrics summary"),
		}},
	}
}

func TestEveryLineOfTheHelpScreenAnswersToTheLanguage(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	rows := func(p *words.Printer) []string {
		t.Helper()

		return Open(0).View(60, 100, world(t, p))
	}

	en, es := rows(words.For("en")), rows(words.For("es"))
	if len(en) != len(es) {
		t.Fatalf("the help screen is %d lines in English and %d in Spanish, want the same shape in both", len(en), len(es))
	}

	same := 0

	for i := range en {
		a := strings.TrimSpace(ansi.Strip(en[i]))
		if a == "" {
			continue
		}

		// The masthead is the mark with the build's name: art with no
		// words in it, and a proper noun with a number. Neither has a
		// Spanish to differ into, so exactly these rows sit out the
		// comparison — art-only rows, and the row carrying the build.
		if isMasthead(a, "dev") {
			continue
		}

		if b := strings.TrimSpace(ansi.Strip(es[i])); a == b {
			same++

			t.Errorf("help line %d reads %q in both languages, want Spanish there", i, a)
		}
	}

	if same == 0 && len(en) == 0 {
		t.Fatal("the help screen drew nothing, so the comparison above proved nothing")
	}
}

// isMasthead is whether a stripped help line is the logo block: one of the
// mark's own art rows, or the row carrying the build's name. It is named
// from markRows rather than written out, so the exemption cannot drift from
// the drawing — and it names rows, not the title beside them, so a sentence
// smuggled into the block still fails the comparison above.
func isMasthead(stripped, version string) bool {
	for _, row := range markRows {
		if stripped == strings.TrimSpace(row) {
			return true
		}
	}

	return strings.HasPrefix(stripped, strings.TrimSpace(markRows[0])) &&
		strings.Contains(stripped, "orbit "+version)
}

// TestTheMastheadNamesTheMarkAndTheBuild. The sheet opens the way `orbit
// version` reads: the mark with its rings on, and the build beside it.
func TestTheMastheadNamesTheMarkAndTheBuild(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	rows := Open(0).View(60, 100, world(t, words.For("en")))
	joined := strings.Join(rows, "\n")

	if !strings.Contains(joined, "orbit dev") {
		t.Errorf("the help screen does not name the build beside its mark:\n%s", joined)
	}

	for _, row := range markRows {
		if !strings.Contains(ansi.Strip(joined), strings.TrimSpace(row)) {
			t.Errorf("the help screen lost the mark's row %q:\n%s", row, joined)
		}
	}
}
