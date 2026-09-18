package cli

// The adapters the window and the browser are handed, and the one rule they
// all follow: they decide nothing.
//
// Each of these read zero. The settings adapter used to keep the window's own
// list of rows and fell six behind the declaration, which is the whole reason
// it now goes straight through — and nothing was checking that it still does.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/ui/known"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// adapterOn is the settings adapter over a state root of the test's own.
func adapterOn(t *testing.T) (*settingsAdapter, *store.Store) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	s, err := store.New(home)
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() }) //nolint:errcheck // the test is over

	a, err := newSettings(s)
	if err != nil {
		t.Fatalf("open the settings: %v", err)
	}

	return a, s
}

// TestTheWindowsRowsAreTheDeclarationsRows. The window used to keep its own
// list and fell six behind: every setting the vocabulary declares has to
// arrive on the screen, and none that it does not.
func TestTheWindowsRowsAreTheDeclarationsRows(t *testing.T) {
	a, _ := adapterOn(t)
	p := words.For("en")

	rows := a.Kept(p)
	declared := verb.Kept(p, store.Settings{})

	if len(rows) != len(declared) {
		t.Fatalf("the window draws %d settings and the declaration has %d",
			len(rows), len(declared))
	}

	for i, row := range rows {
		if row.Name != declared[i].Name {
			t.Errorf("row %d is %q, want %q", i, row.Name, declared[i].Name)
		}
	}
}

// TestWhatASettingReadsAsIsDeclaredOnce. A second copy of a default is the
// one that goes stale, so the adapter keeps none of its own.
func TestWhatASettingReadsAsIsDeclaredOnce(t *testing.T) {
	a, _ := adapterOn(t)

	for _, row := range a.Kept(words.For("en")) {
		if got, want := a.Fresh(row.Name), verb.Fresh(row.Name); got != want {
			t.Errorf("%s reads as %q when nobody has chosen, want the declared %q",
				row.Name, got, want)
		}
	}
}

// TestARowChangedOnScreenTakesTheSameLockATypedSetterDoes. The window must
// not argue with the reader about what it just set, whichever door the value
// came through.
func TestARowChangedOnScreenTakesTheSameLockATypedSetterDoes(t *testing.T) {
	a, s := adapterOn(t)
	p := words.For("en")

	shown, err := a.Choose(p, "language", "es")
	if err != nil {
		t.Fatalf("choose a language: %v", err)
	}

	if !strings.Contains(shown, "es") {
		t.Errorf("the row now reads %q, want the value that was chosen", shown)
	}

	// The file, not the adapter's memory: a screen that agreed with itself
	// and not with the disk is a screen that lies after a restart.
	cfg, err := s.Settings()
	if err != nil {
		t.Fatalf("read the settings back: %v", err)
	}

	if cfg.Language != "es" {
		t.Errorf("the file says %q, want the value the screen set", cfg.Language)
	}
}

// TestAValueTheVocabularyWillNotHaveLeavesTheRowAlone. The refusal travels
// out of the adapter rather than out of the write, and the field keeps what
// it had — a row that took a value nothing validates is a setting nobody can
// trust the shape of.
func TestAValueTheVocabularyWillNotHaveLeavesTheRowAlone(t *testing.T) {
	a, s := adapterOn(t)
	p := words.For("en")

	// unread-cap is a number, and a word is not one.
	if _, err := a.Choose(p, "unread-cap", "plenty"); err == nil {
		t.Fatal("a cap that is not a number was accepted")
	}

	cfg, err := s.Settings()
	if err != nil {
		t.Fatalf("read the settings back: %v", err)
	}

	if cfg.UnreadCap != store.Shipped().UnreadCap {
		t.Errorf("a refused value left the cap at %d, want the %d it was",
			cfg.UnreadCap, store.Shipped().UnreadCap)
	}

	// And a setting nobody declared is refused by name rather than written
	// into a file under a key nothing reads.
	if _, err := a.Choose(p, "not-a-setting", "1"); err == nil {
		t.Error("a setting the vocabulary does not declare was accepted")
	}
}

// TestTheBrowserAsksTheDeclarationWhatAVerbIs. Read off the declaration
// itself, so the answer cannot go stale as verbs are added — a hand-written
// list here is the list that says no to a verb that exists.
func TestTheBrowserAsksTheDeclarationWhatAVerbIs(t *testing.T) {
	h := hands{}

	for _, v := range verb.Every() {
		if !h.Named(v.Path()) {
			t.Errorf("the browser does not know %q, and the declaration does", v.Path())
		}
	}

	for _, no := range []string{"", "board", "task fly", "rm -rf"} {
		if no != "" && h.Named(no) && !hasVerb(no) {
			t.Errorf("the browser answers to %q, and the declaration has no such verb", no)
		}
	}
}

// hasVerb says whether the declaration really holds that name, so the case
// above cannot fail over a verb somebody added since it was written.
func hasVerb(name string) bool {
	for _, v := range verb.Every() {
		if v.Path() == name {
			return true
		}
	}

	return false
}

// TestTheTrayThePageIsHandedIsTheOneTheWindowReads. Both surfaces answer
// from one place, so a sentence waiting in the window is a sentence waiting
// in the browser.
func TestTheTrayThePageIsHandedIsTheOneTheWindowReads(t *testing.T) {
	// A build with nothing behind the tray hands over nothing rather than
	// reaching for a reader it has not got.
	if rows := (knows{}).Waiting(); rows != nil {
		t.Errorf("a build with no tray behind it handed over %+v", rows)
	}

	waiting := []known.Said{
		{Text: "run the tests before you push", From: "operator", Where: "internal/db"},
	}

	k := knows{said: func() []known.Said { return waiting }}

	rows := k.Waiting()
	if len(rows) != 1 {
		t.Fatalf("the page was handed %d rows, want one per sentence waiting", len(rows))
	}

	if rows[0].Text != waiting[0].Text || rows[0].From != waiting[0].From ||
		rows[0].Where != waiting[0].Where {
		t.Errorf("the row arrived as %+v, want the sentence with where it came from", rows[0])
	}
}

// TestTheCheckoutsThePageIsHandedAreReadOffTheDisk. The browser may not
// reach the disk, which is the whole reason these arrive as data.
func TestTheCheckoutsThePageIsHandedAreReadOffTheDisk(t *testing.T) {
	// No board behind it is no checkouts, not a panic.
	if got := (knows{}).Checkouts(); got != nil {
		t.Errorf("a build with no board behind it handed over %+v", got)
	}

	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	s, err := store.New(home)
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}

	defer func() { _ = s.Close() }() //nolint:errcheck // the test is over

	root := t.TempDir()
	initRepo(t, root+"/payments")

	r := board.NewReader(s, root)
	if err := r.Rescan(); err != nil {
		t.Fatalf("look for repositories: %v", err)
	}

	got := (knows{board: r}).Checkouts()
	if len(got) != 1 {
		t.Fatalf("the page was handed %d checkouts, want the one on the board", len(got))
	}

	if got[0].Name != "payments" {
		t.Errorf("the checkout arrived as %q, want the name a reader calls it", got[0].Name)
	}

	if got[0].Path == "" {
		t.Error("the checkout arrived with no path, and the page files rules by it")
	}
}

// TestAKnowledgeReaderWithNothingBehindItAnswersNothing, which is the shape
// every port here takes: a build without one answers empty rather than
// reaching for it.
func TestAKnowledgeReaderWithNothingBehindItAnswersNothing(t *testing.T) {
	facts := []knowledge.Rule{{ID: "aaaa1111", Phrase: "never log a card number"}}

	k := knows{all: func() []knowledge.Rule { return facts }}

	if got := k.Waiting(); got != nil {
		t.Errorf("a build with rules and no tray handed over %+v", got)
	}
}
