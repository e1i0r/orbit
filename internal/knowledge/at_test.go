package knowledge

// The scope a path names: a directory and everything under it, or one file.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aCheckout is a directory with a file in a folder, which is the whole of
// what a scope is read off.
func aCheckout(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()

	if err := os.MkdirAll(filepath.Join(repo, "backend", "ledger"), 0o750); err != nil {
		t.Fatalf("make the folder: %v", err)
	}

	at := filepath.Join(repo, "backend", "ledger", "write.go")
	if err := os.WriteFile(at, []byte("package ledger\n"), 0o600); err != nil {
		t.Fatalf("write the file: %v", err)
	}

	return repo
}

// TestAPathIsADirectoryOrAFile, read off the checkout rather than guessed
// from the spelling.
//
// A name with no dot in it is usually a directory and sometimes a file, and
// a rule filed as the wrong one is a rule that reaches half a project or
// none of it.
func TestAPathIsADirectoryOrAFile(t *testing.T) {
	repo := aCheckout(t)

	folder, err := At(repo, "backend/ledger")
	if err != nil {
		t.Fatalf("a folder that is there: %v", err)
	}

	if folder.Kind != Dir || folder.Path != "backend/ledger" {
		t.Errorf("the folder reads as %+v", folder)
	}

	file, err := At(repo, "backend/ledger/write.go")
	if err != nil {
		t.Fatalf("a file that is there: %v", err)
	}

	if file.Kind != File || file.Path != "backend/ledger/write.go" {
		t.Errorf("the file reads as %+v", file)
	}

	// And a rule about the whole checkout is still a rule about the whole
	// checkout: naming no path is not an error, it is the widest place
	// inside a repository. The root written as a dot is the same place said
	// out loud, which is what somebody needs when the place arrived filled
	// in and is wrong.
	for _, typed := range []string{"", "."} {
		whole, err := At(repo, typed)
		if err != nil || whole.Kind != Repo || whole.Path != "" {
			t.Errorf("%q read as %+v, %v", typed, whole, err)
		}
	}
}

// TestTheFolderCoversWhatIsUnderIt, which is the whole point of putting a
// rule on one: the tree is the inheritance.
func TestTheFolderCoversWhatIsUnderIt(t *testing.T) {
	repo := aCheckout(t)

	folder, err := At(repo, "backend/ledger")
	if err != nil {
		t.Fatal(err)
	}

	inside := Target{Repo: repo, Path: "backend/ledger/write.go"}
	if !folder.Covers(inside) {
		t.Error("a rule on the folder does not reach the file in it")
	}

	beside := Target{Repo: repo, Path: "backend/ledgerfoo/write.go"}
	if folder.Covers(beside) {
		t.Error("a rule on the folder reaches the module next door")
	}
}

// TestAPathThatIsNotThereIsRefused.
//
// The alternative was keeping it for a file that might appear later, and the
// price is worse than the convenience: a typo becomes a rule that quietly
// applies to nothing, and no screen would ever say so.
func TestAPathThatIsNotThereIsRefused(t *testing.T) {
	repo := aCheckout(t)

	_, err := At(repo, "backend/ledgre")
	if err == nil {
		t.Fatal("a path that is not in the checkout was accepted")
	}

	if !strings.Contains(err.Error(), "backend/ledgre") {
		t.Errorf("the refusal reads %q, without saying what was typed", err)
	}

	// And it says what happened to the rule, not what would have happened
	// to it. "would reach nothing" leaves the reader working out whether it
	// was written; "was not written down" is the answer.
	if !strings.Contains(err.Error(), "not written down") {
		t.Errorf("the refusal reads %q, without saying the rule was not kept", err)
	}
}

// TestAPathCannotLeaveTheCheckout, however it is spelled. A rule filed
// outside the repository is a rule that travels with nothing and reaches
// somebody else's code.
func TestAPathCannotLeaveTheCheckout(t *testing.T) {
	repo := aCheckout(t)

	for _, out := range []string{"../elsewhere", "backend/../../elsewhere", "/etc"} {
		if _, err := At(repo, out); err == nil {
			t.Errorf("%q was accepted as a path inside the repository", out)
		}
	}
}

// TestOneFolderHasOneSpelling. `./backend//ledger` and `backend/ledger` are
// the same folder, and two facts filed under two spellings of it are two
// rules nobody can tell apart.
func TestOneFolderHasOneSpelling(t *testing.T) {
	repo := aCheckout(t)

	for _, typed := range []string{"backend/ledger", "./backend/ledger", "backend/ledger/", "/backend/ledger"} {
		got, err := At(repo, typed)
		if err != nil {
			t.Errorf("%q: %v", typed, err)
			continue
		}

		if got.Path != "backend/ledger" {
			t.Errorf("%q was filed as %q", typed, got.Path)
		}
	}
}

// TestARuleAboutAPathNeedsTheRepositoryItIsIn, because a path with no
// checkout under it names nothing at all.
func TestARuleAboutAPathNeedsTheRepositoryItIsIn(t *testing.T) {
	if _, err := At("", "backend/ledger"); err == nil {
		t.Error("a path with no repository was accepted")
	}
}
