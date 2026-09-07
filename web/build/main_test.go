package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// root is the module root, two directories up from this one.
func root(t *testing.T) string {
	t.Helper()

	here, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	return filepath.Dir(filepath.Dir(here))
}

// TestTheCommittedPagesAreWhatTheTemplateSays.
//
// The pages under site/ are generated and committed, because GitHub Pages
// serves that directory as it finds it and nothing runs this program at
// deploy time. So the one thing that can go wrong is the one this catches: a
// sentence changed in the template or a catalogue, and the page the world is
// served still saying the old thing. It fails here rather than being noticed
// on the site.
func TestTheCommittedPagesAreWhatTheTemplateSays(t *testing.T) {
	r := root(t)

	pages, err := render(r)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if len(pages) != len(languages) {
		t.Fatalf("rendered %d pages for %d languages", len(pages), len(languages))
	}

	for name, want := range pages {
		path := filepath.Join(r, "site", name)

		got, err := os.ReadFile(path) //nolint:gosec // a path this test builds itself
		if err != nil {
			t.Fatalf("read %q: %v", path, err)
		}

		if string(got) != string(want) {
			t.Errorf("site/%s is not what the template says — run `make site` and commit it", name)
		}
	}
}

// TestBothLanguagesSayTheSameThings is the other half: a page that renders
// and is committed can still be a page with an empty bullet in it, if a key
// was added to one catalogue and not the other.
func TestBothLanguagesSayTheSameThings(t *testing.T) {
	web := filepath.Join(root(t), "web")

	catalogs := map[string]map[string]string{}

	for _, l := range languages {
		words, err := catalog(filepath.Join(web, l.catalog))
		if err != nil {
			t.Fatalf("read %s: %v", l.catalog, err)
		}

		if len(words) == 0 {
			t.Fatalf("%s has nothing in it", l.catalog)
		}

		catalogs[l.code] = words
	}

	if err := sameKeys(catalogs); err != nil {
		t.Error(err)
	}
}

// TestASentenceInOnlyOneLanguageIsRefused. The check above is only worth
// having if it fails when it should, and it is the kind of check that is
// written once and never exercised again.
func TestASentenceInOnlyOneLanguageIsRefused(t *testing.T) {
	err := sameKeys(map[string]map[string]string{
		"en": {"tagline": "a cockpit", "status": "young"},
		"es": {"tagline": "una cabina"},
	})
	if err == nil {
		t.Fatal("a catalogue missing a sentence was accepted")
	}

	if !strings.Contains(err.Error(), "status") {
		t.Errorf("the error does not name the sentence that is missing: %v", err)
	}
}

// TestBuildWritesBothPages covers the writing itself, against a copy of the
// sources rather than the repository: a test that rewrites the committed
// pages would make the test above pass by changing what it compares against.
func TestBuildWritesBothPages(t *testing.T) {
	tmp := t.TempDir()

	if err := os.CopyFS(filepath.Join(tmp, "web"), os.DirFS(filepath.Join(root(t), "web"))); err != nil {
		t.Fatalf("copy the sources: %v", err)
	}

	written, err := build(tmp)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	for _, l := range languages {
		path := filepath.Join(tmp, "site", l.out)

		body, err := os.ReadFile(path) //nolint:gosec // a path this test builds itself
		if err != nil {
			t.Fatalf("read %q: %v", path, err)
		}

		if string(body) != string(written[l.out]) {
			t.Errorf("%s on disk is not what build answered", l.out)
		}

		if !strings.Contains(string(body), `<html lang="`+l.code+`">`) {
			t.Errorf("%s does not say what language it is in", l.out)
		}
	}
}

// TestAMissingSourceIsAnError, because the program is run from a Makefile and
// a silent empty page is worse than a message.
func TestAMissingSourceIsAnError(t *testing.T) {
	if _, err := render(t.TempDir()); err == nil {
		t.Error("a directory with no template rendered a page")
	}

	if _, err := catalog(filepath.Join(t.TempDir(), "nothing.json")); err == nil {
		t.Error("a catalogue that is not there was read")
	}
}

// TestADamagedCatalogueIsAnError. The catalogues are edited by hand and JSON
// is unforgiving about a trailing comma; the message has to name the file.
func TestADamagedCatalogueIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "es.json")
	if err := os.WriteFile(path, []byte(`{"tagline": }`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := catalog(path)
	if err == nil {
		t.Fatal("a catalogue that is not JSON was read")
	}

	if !strings.Contains(err.Error(), "es.json") {
		t.Errorf("the error does not name the file: %v", err)
	}
}

// TestASentenceNothingSaysIsAnError: the template asking for a key neither
// catalogue has renders as the empty string in every other templating
// arrangement, and here it stops the build.
func TestASentenceNothingSaysIsAnError(t *testing.T) {
	tmp := t.TempDir()
	web := filepath.Join(tmp, "web")

	if err := os.CopyFS(web, os.DirFS(filepath.Join(root(t), "web"))); err != nil {
		t.Fatalf("copy the sources: %v", err)
	}

	page := filepath.Join(web, "page.tmpl.html")

	body, err := os.ReadFile(page) //nolint:gosec // a path this test builds itself
	if err != nil {
		t.Fatalf("read the template: %v", err)
	}

	asked := string(body) + "{{.T.a_sentence_nobody_wrote}}"
	if err := os.WriteFile(page, []byte(asked), 0o600); err != nil {
		t.Fatalf("write the template: %v", err)
	}

	if _, err := render(tmp); err == nil {
		t.Error("the page asked for a sentence nothing says, and rendered anyway")
	}
}

// TestAPlaceItCannotWriteIsAnError, because the Makefile's exit status is the
// only thing watching.
func TestAPlaceItCannotWriteIsAnError(t *testing.T) {
	tmp := t.TempDir()

	if err := os.CopyFS(filepath.Join(tmp, "web"), os.DirFS(filepath.Join(root(t), "web"))); err != nil {
		t.Fatalf("copy the sources: %v", err)
	}

	// A file where the site directory has to go: MkdirAll cannot make a
	// directory out of it, and the write never starts.
	if err := os.WriteFile(filepath.Join(tmp, "site"), nil, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := build(tmp); err == nil {
		t.Error("a site directory that is a file was written into")
	}
}
