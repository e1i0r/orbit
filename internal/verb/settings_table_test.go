package verb

// The settings table in memory, which every surface that draws the settings
// now stands its fixture on.
//
// It read zero, and the reason it exists is the reason that matters: every
// surface had a table of its own written out by hand, and a fixture with its
// own table keeps passing while the screen it stands for falls behind. That
// is exactly how six settings came to be declared and never drawn.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// TestTheTableHoldsEverySettingTheVocabularyDeclares. A fixture over this
// one knows about a setting the afternoon it is declared, which is the whole
// of what it is for.
func TestTheTableHoldsEverySettingTheVocabularyDeclares(t *testing.T) {
	p := words.For("en")

	got := NewTable().Kept(p)

	declared := Kept(p, store.Shipped())
	if len(got) != len(declared) {
		t.Fatalf("the table holds %d settings and the vocabulary declares %d",
			len(got), len(declared))
	}

	for i, one := range got {
		if one.Name != declared[i].Name {
			t.Errorf("row %d is %q, want %q", i, one.Name, declared[i].Name)
		}

		if one.About == "" {
			t.Errorf("%s has nothing said about it, and a dial nobody can read is a dial nobody turns",
				one.Name)
		}
	}
}

// TestAFreshTableIsWhatOrbitShips, so a surface drawing one is drawing what
// a reader who has chosen nothing would see.
func TestAFreshTableIsWhatOrbitShips(t *testing.T) {
	tbl := NewTable()

	for _, one := range tbl.Kept(words.For("en")) {
		if got, want := tbl.Value(one.Name), one.Value; got != want {
			t.Errorf("%s holds %q and draws %q", one.Name, got, want)
		}

		if got, want := tbl.Fresh(one.Name), Fresh(one.Name); got != want {
			t.Errorf("%s reads fresh as %q, want the declared %q", one.Name, got, want)
		}
	}
}

// TestChoosingGoesThroughTheValidatorDeclaredBesideTheSetting. It refuses
// what the real table refuses, which is the other half of a fixture being
// worth standing on.
func TestChoosingGoesThroughTheValidatorDeclaredBesideTheSetting(t *testing.T) {
	p := words.For("en")
	tbl := NewTable()

	shown, err := tbl.Choose(p, "autopilot", "1")
	if err != nil {
		t.Fatalf("switch autopilot on: %v", err)
	}

	// The form worth printing and not what was typed: "1" is stored as true
	// and shown as "on", so the confirmation says what the table now holds
	// rather than repeating the argument.
	if shown == "1" {
		t.Errorf("it answered %q, want the form worth showing", shown)
	}

	if got := tbl.Value("autopilot"); got != shown {
		t.Errorf("the table holds %q and answered %q", got, shown)
	}
}

// TestAValueTheVocabularyWillNotHaveLeavesTheTableAlone. A table that took a
// value nothing validates is a fixture that passes where the screen refuses.
func TestAValueTheVocabularyWillNotHaveLeavesTheTableAlone(t *testing.T) {
	p := words.For("en")
	tbl := NewTable()

	was := tbl.Value("unread-cap")

	if _, err := tbl.Choose(p, "unread-cap", "plenty"); err == nil {
		t.Fatal("a cap that is not a number was accepted")
	}

	if got := tbl.Value("unread-cap"); got != was {
		t.Errorf("a refused value moved the cap from %q to %q", was, got)
	}
}

// TestASettingNobodyDeclaredIsRefusedByNameAndSaysWhichThereAre. A reader
// who mistyped one needs the list, not a no.
func TestASettingNobodyDeclaredIsRefusedByNameAndSaysWhichThereAre(t *testing.T) {
	p := words.For("en")
	tbl := NewTable()

	_, err := tbl.Choose(p, "not-a-setting", "1")
	if err == nil {
		t.Fatal("a setting nobody declared was accepted")
	}

	if !strings.Contains(err.Error(), "not-a-setting") {
		t.Errorf("the refusal is %q, want it to say back what was typed", err)
	}

	// And asked what it holds for one, it answers nothing rather than
	// inventing a value for a key that does not exist.
	if got := tbl.Value("not-a-setting"); got != "" {
		t.Errorf("a setting that does not exist holds %q", got)
	}
}

// TestTwoTablesDoNotShareWhatWasChosen. It is not concurrent and does not
// pretend to be, but a surface holding one must not find another surface's
// choice in it.
func TestTwoTablesDoNotShareWhatWasChosen(t *testing.T) {
	p := words.For("en")

	mine, theirs := NewTable(), NewTable()

	if _, err := mine.Choose(p, "language", "es"); err != nil {
		t.Fatalf("choose a language: %v", err)
	}

	if got := theirs.Value("language"); got == "es" {
		t.Error("a choice made in one table turned up in another")
	}
}
