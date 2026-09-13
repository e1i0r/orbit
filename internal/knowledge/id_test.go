package knowledge

// A rule's name: coined once, and surviving everything else about it.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aRepo is a checkout with nothing written down about it yet.
func aRepo(t *testing.T) (*Store, string) {
	t.Helper()

	return NewStore(t.TempDir()), t.TempDir()
}

// TestARuleIsNamedOnceAndKeepsItsName.
//
// Everything else about a fact can change. The file is named after the
// sentence, so rewording one renames it, and what the record wrote down
// about the old name stops being findable — which would make a cycle that
// cannot say "this is the same thing you said differently three weeks ago".
func TestARuleIsNamedOnceAndKeepsItsName(t *testing.T) {
	s, repo := aRepo(t)

	was := Fact{
		Scope: Scope{Kind: Repo, Repo: repo}, Source: Human,
		Phrase: "amounts are cents",
	}

	if _, err := s.Save(was); err != nil {
		t.Fatalf("save: %v", err)
	}

	written := only(t, s, repo)
	if written.ID == "" {
		t.Fatal("a fact Orbit wrote has no name")
	}

	// Reworded, which renames the file: the name has to come through it.
	now := written
	now.Phrase = "every amount in this project is in cents"

	if _, err := s.Replace(written, now); err != nil {
		t.Fatalf("replace: %v", err)
	}

	after := only(t, s, repo)
	if after.ID != written.ID {
		t.Errorf("rewording it renamed the rule from %q to %q", written.ID, after.ID)
	}

	if !strings.Contains(after.Phrase, "every amount") {
		t.Errorf("the sentence reads %q", after.Phrase)
	}
}

// TestTwoRulesAreNotTheSameRule, which is the other half of a name being
// worth anything.
func TestTwoRulesAreNotTheSameRule(t *testing.T) {
	s, repo := aRepo(t)

	for _, phrase := range []string{"amounts are cents", "errors are wrapped"} {
		if _, err := s.Save(Fact{
			Scope: Scope{Kind: Repo, Repo: repo}, Source: Human, Phrase: phrase,
		}); err != nil {
			t.Fatalf("save %q: %v", phrase, err)
		}
	}

	facts, err := s.LoadRepo(repo)
	if err != nil {
		t.Fatal(err)
	}

	if len(facts) != 2 {
		t.Fatalf("the checkout holds %d facts", len(facts))
	}

	if facts[0].ID == facts[1].ID {
		t.Errorf("two rules are both called %q", facts[0].ID)
	}
}

// TestAFactWrittenByHandIsReadWithoutAName.
//
// Writing these by hand is half the reason they are files, so a header of
// two lines has to work. It gets a name the first time Orbit writes it, and
// not while reading: a walk that wrote to every file it read would put a
// diff in somebody's checkout for having opened a list.
func TestAFactWrittenByHandIsReadWithoutAName(t *testing.T) {
	s, repo := aRepo(t)

	at := filepath.Join(repo, ".orbit", "knowledge", "by-hand.md")
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}

	body := "---\nscope: repo\nsource: human\n---\n\nthe migrations are generated\n"
	if err := os.WriteFile(at, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	written := only(t, s, repo)
	if written.ID != "" {
		t.Errorf("reading a hand-written fact named it %q", written.ID)
	}

	// And the file on disk is untouched by the reading.
	again, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}

	if string(again) != body {
		t.Errorf("reading the fact rewrote it:\n%s", again)
	}

	// Edited through a screen, it gets one, and the file says so.
	now := written
	now.Phrase = "the migrations are generated, never hand-edited"

	if _, err := s.Replace(written, now); err != nil {
		t.Fatalf("replace: %v", err)
	}

	if after := only(t, s, repo); after.ID == "" {
		t.Error("a hand-written fact Orbit has now written still has no name")
	}
}

// TestTheNameIsInTheFileAndNotInItsName.
//
// If the id were the file name the directory becomes unreadable, and half of
// why these are files is that somebody can open the folder and see what
// Orbit thinks it knows.
func TestTheNameIsInTheFileAndNotInItsName(t *testing.T) {
	s, repo := aRepo(t)

	if _, err := s.Save(Fact{
		Scope: Scope{Kind: Repo, Repo: repo}, Source: Human,
		Phrase: "amounts are cents", Ref: "PAY-1",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	at := filepath.Join(repo, ".orbit", "knowledge", "PAY-1-amounts-are-cents.md")

	body, err := os.ReadFile(at)
	if err != nil {
		t.Fatalf("the file is not named after what it says: %v", err)
	}

	if !strings.Contains(string(body), "id: ") {
		t.Errorf("the file does not carry a name:\n%s", body)
	}
}

// only is the single fact the checkout holds.
func only(t *testing.T, s *Store, repo string) Fact {
	t.Helper()

	facts, err := s.LoadRepo(repo)
	if err != nil {
		t.Fatalf("read what the checkout knows: %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("the checkout holds %d facts, want one", len(facts))
	}

	return facts[0]
}

// TestEditingAFactSomebodyWroteByHandLeavesNoCopyBehind.
//
// A file written by hand is called whatever they called it, and a
// replacement that worked the old name out of the fact's own fields put the
// new copy somewhere else and left the original where it was — still read,
// still told to every phase, still refusing work over a sentence nobody
// meant to keep.
func TestEditingAFactSomebodyWroteByHandLeavesNoCopyBehind(t *testing.T) {
	s, repo := aRepo(t)

	at := filepath.Join(repo, ".orbit", "knowledge", "whatever-they-called-it.md")
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(at,
		[]byte("---\nscope: repo\nsource: human\n---\n\nnever push on a Friday\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	was := only(t, s, repo)

	now := was
	now.Phrase = "never push on a Friday afternoon"

	if _, err := s.Replace(was, now); err != nil {
		t.Fatalf("replace: %v", err)
	}

	// One fact, and only takes care of saying so.
	after := only(t, s, repo)
	if !strings.Contains(after.Phrase, "afternoon") {
		t.Errorf("the one left reads %q", after.Phrase)
	}

	if _, err := os.Stat(at); err == nil {
		t.Error("the file it was read out of is still there")
	}
}
