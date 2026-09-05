package knowledge

// Where a fact lands on disk, and whether it comes back the same.

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

// TestAFactOfARepositoryLivesInTheRepository is what makes knowledge travel.
//
// It goes under `.orbit/knowledge/` inside the checkout, so it moves with the
// push, whoever clones the project gets what Orbit learned about it, and a
// rule that is about to start steering the agent shows up in a diff somebody
// reviews rather than appearing on one machine in silence.
func TestAFactOfARepositoryLivesInTheRepository(t *testing.T) {
	state, repo := roots(t)
	s := NewStore(state)

	f := Fact{
		Scope:  Scope{Kind: Dir, Repo: repo, Path: "backend/ledger"},
		Source: Human,
		Ref:    "REF-9",
		Phrase: "No UPDATE or DELETE in ledger. Reconcile marks, it does not correct.",
	}

	where, err := s.Save(f)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	want := filepath.Join(repo, ".orbit", "knowledge", "backend", "ledger", "REF-9.md")
	if where != want {
		t.Errorf("the fact was written to %s,\nwant %s", where, want)
	}

	if _, err := os.Stat(want); err != nil {
		t.Errorf("nothing is there: %v", err)
	}
}

// TestAGeneralFactLivesInTheStateRootAndNoRepository. It is about no
// checkout in particular, so putting it in one would be picking a repository
// at random and making it apply only while somebody works there.
func TestAGeneralFactLivesInTheStateRootAndNoRepository(t *testing.T) {
	state, repo := roots(t)
	s := NewStore(state)

	where, err := s.Save(Fact{
		Scope:  Scope{Kind: General},
		Source: Human,
		Phrase: "The PRs and the commits are written in English.",
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !strings.HasPrefix(where, state) {
		t.Errorf("a general fact was written to %s, outside the state root %s", where, state)
	}

	if entries, err := os.ReadDir(filepath.Join(repo, ".orbit")); err == nil && len(entries) > 0 {
		t.Error("a general fact left something inside a repository")
	}
}

// TestALanguageFactIsFiledUnderItsLanguage, beside the general ones and in
// the state root for the same reason: it belongs to every checkout at once.
func TestALanguageFactIsFiledUnderItsLanguage(t *testing.T) {
	state, _ := roots(t)
	s := NewStore(state)

	where, err := s.Save(Fact{
		Scope:  Scope{Kind: Language, Lang: "go"},
		Source: Human,
		Phrase: "Never discard what a call answered with _.",
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if want := filepath.Join(state, "knowledge", "lang", "go"); filepath.Dir(where) != want {
		t.Errorf("a Go fact went to %s, want it under %s", filepath.Dir(where), want)
	}
}

// TestWhatWasWrittenComesBack is the whole of the format: a fact saved and
// loaded is the same fact.
func TestWhatWasWrittenComesBack(t *testing.T) {
	state, repo := roots(t)
	s := NewStore(state)

	want := Fact{
		Scope:  Scope{Kind: File, Repo: repo, Path: "internal/ui/bar.go"},
		Source: Human,
		Ref:    "ORB-115",
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
		t.Fatalf("loaded %d facts, want 1", len(got))
	}

	if got[0].Phrase != want.Phrase || got[0].Check != want.Check || !got[0].Stops {
		t.Errorf("what came back is not what went in:\ngot  %+v\nwant %+v", got[0], want)
	}

	if got[0].Scope != want.Scope {
		t.Errorf("the scope came back as %+v, want %+v", got[0].Scope, want.Scope)
	}

	if got[0].Source != Human || got[0].Ref != "ORB-115" {
		t.Errorf("the source came back as %v ref %q", got[0].Source, got[0].Ref)
	}
}

// TestLoadBringsBothRootsTogether. Two places on disk, one answer: the
// caller asks what is known while working in a repository and gets the
// general facts, the ones of its languages, and the repository's own.
func TestLoadBringsBothRootsTogether(t *testing.T) {
	state, repo := roots(t)
	s := NewStore(state)

	for _, f := range []Fact{
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
		t.Fatalf("loaded %d facts, want the one written by hand", len(got))
	}

	if got[0].Phrase != "The cockpit is checked at 100 columns, not 180." {
		t.Errorf("the sentence came back as %q", got[0].Phrase)
	}

	if got[0].Scope.Kind != Dir || got[0].Scope.Path != "internal/ui" {
		t.Errorf("the scope was read as %+v, want the directory it was filed in", got[0].Scope)
	}
}
