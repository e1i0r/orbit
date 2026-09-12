package knowledge

// Where a fact lands on disk, and what it is called there.
//
// Both halves are the design and not an implementation detail. Where it
// lands is what makes a fact travel: one about a checkout lives inside that
// checkout, so it arrives with a clone and goes through review, rather than
// existing on one machine in silence. What it is called is what keeps two of
// them apart.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	want := filepath.Join(repo, ".orbit", "knowledge", "backend", "ledger",
		"REF-9-no-update-or-delete-in-ledger.md")
	if where != want {
		t.Errorf("the fact was written to %s,\nwant %s", where, want)
	}

	if _, err := os.Stat(want); err != nil {
		t.Errorf("nothing is there: %v", err)
	}
}

// TestOneTaskCanTeachMoreThanOneThing.
//
// A fact is named after what it came out of, and every fact an agent writes
// mid-task comes out of that task. Named by the reference alone, the second
// thing it learned wrote over the first and said "written down" about it.
func TestOneTaskCanTeachMoreThanOneThing(t *testing.T) {
	state, repo := roots(t)
	s := NewStore(state)

	var where []string

	for _, phrase := range []string{
		"amounts are cents, never floats",
		"the fuzz test hangs on a fixed seed",
	} {
		at, err := s.Save(Fact{
			Scope:  Scope{Kind: Repo, Repo: repo},
			Source: FromRecord,
			Ref:    "PAY-1",
			Phrase: phrase,
		})
		if err != nil {
			t.Fatalf("Save: %v", err)
		}

		where = append(where, at)
	}

	if where[0] == where[1] {
		t.Fatalf("both went to %s, and one of them is gone", where[0])
	}

	got, err := s.LoadRepo(repo)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}

	if len(got) != 2 {
		t.Errorf("the repository holds %d of the two things that task learned", len(got))
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
