package ui

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
	"github.com/e1i0r/orbit/internal/words"
)

func TestEveryLineOfTheHelpScreenAnswersToTheLanguage(t *testing.T) {
	english, spanish := printers(t)

	rows := func(p *words.Printer) []string {
		t.Helper()
		m := modelWith(t, p, fixtureBoard(fixtureTasks(), 4), 100, 30, &recorder{})

		return m.openHelp().helpRows(60, 100)
	}

	en, es := rows(english), rows(spanish)
	if len(en) != len(es) {
		t.Fatalf("the help screen is %d lines in English and %d in Spanish, want the same shape in both", len(en), len(es))
	}

	same := 0

	for i := range en {
		a := strings.TrimSpace(ansi.Strip(en[i]))
		if a == "" {
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
