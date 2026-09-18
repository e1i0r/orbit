package knowledge

// Where a rule lands on disk, and whether it comes back the same.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// roots is a state root and a repository, both temporary.
func roots(t *testing.T) (state, repo string) {
	t.Helper()

	return t.TempDir(), t.TempDir()
}

// TestWhatWasWrittenComesBack is the whole of the format: a rule saved and
// loaded is the same rule.
func TestWhatWasWrittenComesBack(t *testing.T) {
	state, repo := roots(t)
	s := NewStore(state)

	want := Rule{
		Scope:  Scope{Kind: File, Repo: repo, Path: "internal/ui/bar.go"},
		Source: Human,
		Ref:    "ACME-115",
		Phrase: "The bar drops hints from the end: what matters goes first.",
		Stops:  true,
		Check:  "go test ./internal/ui/ -run TestTheFlowsKey",
	}

	if _, err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("loaded %d rules, want 1", len(got))
	}

	if got[0].Phrase != want.Phrase || got[0].Check != want.Check || !got[0].Stops {
		t.Errorf("what came back is not what went in:\ngot  %+v\nwant %+v", got[0], want)
	}

	if got[0].Scope != want.Scope {
		t.Errorf("the scope came back as %+v, want %+v", got[0].Scope, want.Scope)
	}

	if got[0].Source != Human || got[0].Ref != "ACME-115" {
		t.Errorf("the source came back as %v ref %q", got[0].Source, got[0].Ref)
	}
}

// TestLoadBringsBothRootsTogether. Two places on disk, one answer: the
// caller asks what is known while working in a repository and gets the
// general rules, the ones of its languages, and the repository's own.
func TestLoadBringsBothRootsTogether(t *testing.T) {
	state, repo := roots(t)
	s := NewStore(state)

	for _, f := range []Rule{
		{Scope: Scope{Kind: General}, Source: Human, Phrase: "of everything"},
		{Scope: Scope{Kind: Language, Lang: "go"}, Source: Human, Phrase: "of Go"},
		{Scope: Scope{Kind: Repo, Repo: repo}, Source: Human, Phrase: "of the repository"},
	} {
		if _, err := s.Save(f); err != nil {
			t.Fatalf("Save %q: %v", f.Phrase, err)
		}
	}

	got, err := s.Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	said := make([]string, 0, len(got))
	for _, f := range got {
		said = append(said, f.Phrase)
	}

	slices.Sort(said)

	if want := []string{"of Go", "of everything", "of the repository"}; !slices.Equal(said, want) {
		t.Errorf("Load answered %v, want %v", said, want)
	}
}

// TestLoadRepoBringsBackOnlyTheRepositorys is the other half of the pair.
//
// A caller walking every repository the record knows about already holds the
// state root's rules, and Load would hand them back once per repository — a
// walk and a decode each time, thrown away each time.
func TestLoadRepoBringsBackOnlyTheRepositorys(t *testing.T) {
	state, repo := roots(t)
	s := NewStore(state)

	for _, f := range []Rule{
		{Scope: Scope{Kind: General}, Source: Human, Phrase: "of everything"},
		{Scope: Scope{Kind: Language, Lang: "go"}, Source: Human, Phrase: "of Go"},
		{Scope: Scope{Kind: Repo, Repo: repo}, Source: Human, Phrase: "of the repository"},
	} {
		if _, err := s.Save(f); err != nil {
			t.Fatalf("Save %q: %v", f.Phrase, err)
		}
	}

	got, err := s.LoadRepo(repo)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}

	said := make([]string, 0, len(got))
	for _, f := range got {
		said = append(said, f.Phrase)
	}

	if want := []string{"of the repository"}; !slices.Equal(said, want) {
		t.Errorf("LoadRepo answered %v, want %v", said, want)
	}
}

// TestLoadRepoOnACheckoutNobodyHasWrittenAbout. Every repository starts as
// one, and it is not a failure — the same answer Load gives.
func TestLoadRepoOnACheckoutNobodyHasWrittenAbout(t *testing.T) {
	state, repo := roots(t)

	got, err := NewStore(state).LoadRepo(repo)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("a checkout with nothing written about it answered %v", got)
	}
}

// TestAFactWrittenByHandIsRead. The file is the source for anything somebody
// typed themselves, so one dropped into the directory has to be picked up
// with no ceremony.
func TestAFactWrittenByHandIsRead(t *testing.T) {
	state, repo := roots(t)

	dir := filepath.Join(repo, ".orbit", "knowledge", "internal", "ui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	body := "---\nscope: dir\nsource: human\nref: BY-HAND\n---\n\nThe cockpit is checked at 100 columns, not 180.\n"
	if err := os.WriteFile(filepath.Join(dir, "columns.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := NewStore(state).Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("loaded %d rules, want the one written by hand", len(got))
	}

	if got[0].Phrase != "The cockpit is checked at 100 columns, not 180." {
		t.Errorf("the sentence came back as %q", got[0].Phrase)
	}

	if got[0].Scope.Kind != Dir || got[0].Scope.Path != "internal/ui" {
		t.Errorf("the scope was read as %+v, want the directory it was filed in", got[0].Scope)
	}
}

// TestChangingASentenceDoesNotLeaveTheOldOneBehind.
//
// A rule with no reference is filed under a slug of its own sentence, so
// editing the sentence writes to a different path. Saving alone would leave
// both, and the one nobody meant to keep would go on being told.
func TestChangingASentenceDoesNotLeaveTheOldOneBehind(t *testing.T) {
	state, repo := roots(t)
	s := NewStore(state)

	was := Rule{
		Scope:  Scope{Kind: Repo, Repo: repo},
		Source: Human,
		Phrase: "the fuxx tests hang sometimes",
	}

	if _, err := s.Save(was); err != nil {
		t.Fatalf("Save: %v", err)
	}

	now := was
	now.Phrase = "the fuzz tests hang sometimes"

	if _, err := s.Replace(was, now); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	got, err := s.Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got) != 1 {
		phrases := make([]string, 0, len(got))
		for _, f := range got {
			phrases = append(phrases, f.Phrase)
		}

		t.Fatalf("the repository holds %d rules: %v", len(got), phrases)
	}

	if got[0].Phrase != now.Phrase {
		t.Errorf("the rule says %q, want the sentence it was changed to", got[0].Phrase)
	}
}

// TestReplacingInPlaceKeepsTheOneFile, so that turning a rule off does not
// depend on the sentence having stayed the same.
func TestReplacingInPlaceKeepsTheOneFile(t *testing.T) {
	state, repo := roots(t)
	s := NewStore(state)

	was := Rule{Scope: Scope{Kind: Repo, Repo: repo}, Source: Human, Ref: "REF-9", Phrase: "no UPDATE in ledger"}
	if _, err := s.Save(was); err != nil {
		t.Fatalf("Save: %v", err)
	}

	now := was
	now.State = Off

	if _, err := s.Replace(was, now); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	got, err := s.Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got) != 1 || got[0].State != Off {
		t.Errorf("the repository holds %d rules and the first stands at %v", len(got), got[0].State)
	}
}

// TestARuleCorrectedKeepsItsName. The name survives the change, which is the
// whole of what it is for: a screen that rebuilt the rule from what was typed
// would coin a second name for one rule, and everything the record wrote
// about the first would be about something that no longer exists.
func TestARuleCorrectedKeepsItsName(t *testing.T) {
	s, repo := aRepo(t)

	was := Rule{Scope: Scope{Kind: Repo, Repo: repo}, Source: Human, Phrase: "amounts are cents"}
	if _, err := s.Save(was); err != nil {
		t.Fatalf("save: %v", err)
	}

	written := only(t, s, repo)
	if written.ID == "" {
		t.Fatal("a rule Orbit wrote has no name")
	}

	// What a screen hands back: the sentence changed, and no name on it.
	now := written
	now.ID = ""
	now.Phrase = "amounts are cents, never floats"

	if _, err := s.Replace(written, now); err != nil {
		t.Fatalf("correct it: %v", err)
	}

	after := only(t, s, repo)
	if after.ID != written.ID {
		t.Errorf("the corrected rule is called %q, want the %q it was", after.ID, written.ID)
	}

	if after.Phrase != now.Phrase {
		t.Errorf("it says %q, want the corrected sentence", after.Phrase)
	}
}

// TestAFileNameIsCutToItsFirstFewWords. The file is named after the sentence
// so a reader can find it in a listing, and a sentence of forty words would
// otherwise be a filename of forty words.
func TestAFileNameIsCutToItsFirstFewWords(t *testing.T) {
	// Six, which is what slug cuts at: enough that the name still reads as
	// the sentence it came from.
	const kept = 6

	long := "amounts are always cents and never floats because a float loses a penny somewhere"

	got := fileName(Rule{Phrase: long, Scope: Scope{Kind: General}, Source: Human})

	parts := strings.Split(strings.TrimSuffix(got, ".md"), "-")
	if len(parts) > kept {
		t.Errorf("the file is called %q, which is %d words and the cut is %d", got, len(parts), kept)
	}

	if !strings.HasPrefix(got, "amounts") {
		t.Errorf("the file is called %q, want it to start with the sentence's first word", got)
	}

	// And one already short enough keeps every word it has.
	short := fileName(Rule{Phrase: "amounts are cents", Scope: Scope{Kind: General}, Source: Human})
	for _, word := range []string{"amounts", "are", "cents"} {
		if !strings.Contains(short, word) {
			t.Errorf("the file is called %q, want %q in it", short, word)
		}
	}
}
