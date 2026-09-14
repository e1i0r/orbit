package learn

// Reading what a project already says about itself.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aProject is a checkout with the files a project keeps its expectations in.
func aProject(t *testing.T, files map[string]string) string {
	t.Helper()

	repo := t.TempDir()

	for name, body := range files {
		at := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
			t.Fatalf("make room for %s: %v", name, err)
		}

		if err := os.WriteFile(at, []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	return repo
}

const aContributing = `# Contributing

This project is written in Go.

Every change ships with a test that fails without it.
Never push on a Friday afternoon.
`

// TestARuleHasToPointAtALineOfTheFile.
//
// The whole reason this can be trusted. A model asked to summarise two years
// of CONTRIBUTING will produce plausible rules nobody ever wrote, and the
// only way to tell those from the real ones without reading it yourself is to
// make it point at the line.
func TestARuleHasToPointAtALineOfTheFile(t *testing.T) {
	s := root(t)
	repo := aProject(t, map[string]string{"CONTRIBUTING.md": aContributing})

	answer := "testing | a change comes with a test | Every change ships with a test that fails without it.\n" +
		"process | deploys go out on Tuesdays | Deploys go out on Tuesdays."

	got, err := Read(context.Background(), s, answering(answer, nil), repo)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("the answer became %d rules: %+v", len(got), got)
	}

	if !strings.Contains(got[0].Text, "comes with a test") {
		t.Errorf("the one that held up reads %q", got[0].Text)
	}

	// And it says where to go and look, which is half of why a rule from a
	// document can be trusted at all.
	if got[0].About != "CONTRIBUTING.md:5" {
		t.Errorf("it says it came from %q", got[0].About)
	}

	if got[0].By != FromAPaper {
		t.Errorf("it says it came from %q", got[0].By)
	}
}

// TestAFileAlreadyReadIsNotReadAgain, until it changes. Reading the same
// CONTRIBUTING twice offers the same rules twice, and the second time they
// were already answered.
func TestAFileAlreadyReadIsNotReadAgain(t *testing.T) {
	s := root(t)
	repo := aProject(t, map[string]string{"CONTRIBUTING.md": aContributing})

	answer := "testing | a change comes with a test | Every change ships with a test that fails without it."

	if _, err := Read(context.Background(), s, answering(answer, nil), repo); err != nil {
		t.Fatalf("reading: %v", err)
	}

	again, err := Read(context.Background(), s, answering(answer, nil), repo)
	if err != nil {
		t.Fatalf("reading again: %v", err)
	}

	if len(again) != 0 {
		t.Errorf("a file nobody has touched was read again: %+v", again)
	}

	// Rewritten, it is read again: what changed may be a rule.
	rewritten := aContributing + "\nAlways wrap an error with what you were doing.\n"
	if err := os.WriteFile(filepath.Join(repo, "CONTRIBUTING.md"), []byte(rewritten), 0o600); err != nil {
		t.Fatal(err)
	}

	after, err := Read(context.Background(), s, answering(answer, nil), repo)
	if err != nil {
		t.Fatal(err)
	}

	if len(after) == 0 {
		t.Error("a file somebody rewrote was not read again")
	}
}

// TestARuleInAFolderIsAboutThatFolder, because a rule in backend/README is
// about backend.
func TestARuleInAFolderIsAboutThatFolder(t *testing.T) {
	if got := alongside("backend/README.md"); got != "backend" {
		t.Errorf("a rule in backend/README.md is about %q", got)
	}

	if got := alongside("CONTRIBUTING.md"); got != "" {
		t.Errorf("a rule at the root is about %q", got)
	}
}

// TestAQuoteTheFileWrappedIsStillTheQuote.
//
// A model copying a sentence out of a wrapped paragraph rejoins it with one
// space where the file had a newline, and refusing that would throw away real
// quotes for a difference nobody can see.
func TestAQuoteTheFileWrappedIsStillTheQuote(t *testing.T) {
	body := "# Contributing\n\nEvery change ships with a test\nthat fails without it.\n"

	at, there := lineOf(body, "Every change ships with a test that fails without it.")
	if !there || at != 3 {
		t.Errorf("the wrapped sentence reads as line %d, there=%v", at, there)
	}

	if _, there := lineOf(body, "deploys go out on Tuesdays"); there {
		t.Error("a sentence that is not in the file was found in it")
	}
}

// TestReadingNeedsAnEngineAndACheckout, and says which is missing rather than
// answering with nothing.
func TestReadingNeedsAnEngineAndACheckout(t *testing.T) {
	s := root(t)

	if _, err := Read(context.Background(), s, nil, t.TempDir()); err == nil {
		t.Error("reading with no engine answered as though it had one")
	}

	if _, err := Read(context.Background(), s, answering("", nil), ""); err == nil {
		t.Error("reading no checkout answered as though there were one")
	}
}
