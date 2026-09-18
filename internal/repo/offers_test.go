package repo

// What a checkout offers somebody filling in a form about it.
//
// Both read zero. Filing a rule under a folder used to mean knowing by heart
// which folders a project has and spelling one right; these are what
// replaced that, and a reading that quietly answers nothing puts the typing
// straight back.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// checkout is a directory with those entries in it: a name ending in / is a
// folder, anything else is a file with that content.
func checkout(t *testing.T, entries map[string]string) string {
	t.Helper()

	root := t.TempDir()

	for name, body := range entries {
		path := filepath.Join(root, name)

		if strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(path, 0o750); err != nil {
				t.Fatalf("make %s: %v", name, err)
			}

			continue
		}

		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	return root
}

// TestTheFoldersOfferedAreTheOnesAReaderWouldNameOutLoud. One level and not
// the tree: two levels of a real project is hundreds of entries, and the
// folder a rule is about is almost always one somebody could say.
func TestTheFoldersOfferedAreTheOnesAReaderWouldNameOutLoud(t *testing.T) {
	root := checkout(t, map[string]string{
		"internal/":     "",
		"cmd/":          "",
		"docs/":         "",
		"internal/db/":  "",
		".git/":         "",
		".github/":      "",
		"node_modules/": "",
		"vendor/":       "",
		"dist/":         "",
		"README.md":     "# orbit",
		"go.mod":        "module orbit",
		"internal/x.go": "package x",
		"coverage/":     "",
		"__pycache__/":  "",
		"build/":        "",
		"target/":       "",
	})

	got := Folders(root)

	want := []string{"cmd", "docs", "internal"}
	if !slices.Equal(got, want) {
		t.Errorf("it offers %v, want %v", got, want)
	}
}

// TestWhatIsNotOfferedIsWhatNobodyWouldFileARuleUnder. A dot directory is
// git's or an editor's, and the ignored ones are somebody else's code: a
// rule filed under vendor/ is a rule about work this project does not do.
func TestWhatIsNotOfferedIsWhatNobodyWouldFileARuleUnder(t *testing.T) {
	for _, name := range []string{
		".git", ".github", "node_modules", "vendor", "__pycache__", "build", "dist",
		"target", "coverage",
	} {
		root := checkout(t, map[string]string{name + "/": "", "internal/": ""})

		if got := Folders(root); slices.Contains(got, name) {
			t.Errorf("%s was offered: %v", name, got)
		}
	}
}

// TestACheckoutThatCannotBeReadOffersNothing, rather than a panic or a list
// somebody would act on.
func TestACheckoutThatCannotBeReadOffersNothing(t *testing.T) {
	if got := Folders(filepath.Join(t.TempDir(), "not-there")); got != nil {
		t.Errorf("a directory that is not there offered %v", got)
	}

	if got := Checks(filepath.Join(t.TempDir(), "not-there")); got != nil {
		t.Errorf("a checkout with no Makefile offered %v", got)
	}
}

// TestTheChecksOfferedAreTheCommandsTheTeamAlreadyTypes. The Makefile and
// not a guess at the language's usual command: a rule given one of the
// project's own targets is a rule whose gate the team already trusts.
func TestTheChecksOfferedAreTheCommandsTheTeamAlreadyTypes(t *testing.T) {
	root := checkout(t, map[string]string{"Makefile": strings.Join([]string{
		"GO ?= go",
		"",
		".PHONY: check test",
		"",
		"check: fmt vet test",
		"\t@echo ok",
		"",
		"test:",
		"\t$(GO) test ./...",
		"",
		"# a variable holding a colon is not a target",
		"LDFLAGS := -X main.Version=1:2",
		"",
		"# a pattern rule names no command somebody types",
		"%.o: %.c",
		"\tcc -c $<",
		"",
		"# and the same target twice is one offer",
		"test:",
		"\t$(GO) test -race ./...",
	}, "\n")})

	got := Checks(root)

	want := []string{"make check", "make test"}
	if !slices.Equal(got, want) {
		t.Errorf("it offers %v, want %v", got, want)
	}
}

// TestNeitherOfferIsUnbounded. A ceiling against a pathological repository
// and not a shortlist: the number is past what any real project reaches, so
// a reader never finds the one they wanted missing.
func TestNeitherOfferIsUnbounded(t *testing.T) {
	folders := map[string]string{}

	var lines []string

	for i := range atMostOffered + 10 {
		name := string(rune('a'+i%26)) + string(rune('a'+i/26)) + "-dir"
		folders[name+"/"] = ""

		lines = append(lines, string(rune('a'+i%26))+string(rune('a'+i/26))+"-target:")
	}

	if got := Folders(checkout(t, folders)); len(got) != atMostOffered {
		t.Errorf("a checkout with %d folders offered %d, want %d",
			len(folders), len(got), atMostOffered)
	}

	made := checkout(t, map[string]string{"Makefile": strings.Join(lines, "\n")})
	if got := Checks(made); len(got) != atMostOffered {
		t.Errorf("a Makefile with %d targets offered %d, want %d",
			len(lines), len(got), atMostOffered)
	}
}
