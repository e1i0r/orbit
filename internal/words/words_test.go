package words

// words_test.go is task 2's whole point: from the moment a screen draws a
// string through T or P, this file fails the build the instant es.json
// falls out of step with the code, in either direction. Today it passes
// trivially, because nothing outside this package calls T or P yet.

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
	"unicode/utf8"
)

// placeholderPattern matches one named placeholder, {like this}.
var placeholderPattern = regexp.MustCompile(`\{[a-zA-Z0-9_]+\}`)

func placeholders(s string) []string {
	found := placeholderPattern.FindAllString(s, -1)
	slices.Sort(found)

	return found
}

// hasCombiningMark reports whether s contains a combining diacritical mark
// (U+0300–U+036F) — the signature of a decomposed, non-NFC string.
// unicode/norm needs x/text, which is not on the allowlist; this range is
// sufficient to catch the mistake in English and Spanish, the only two
// languages this package ships.
func hasCombiningMark(s string) bool {
	for _, r := range s {
		if r >= 0x0300 && r <= 0x036F {
			return true
		}
	}

	return false
}

// loadRepoCatalog reads a checked-in catalogue file directly, rather than
// through loadCatalog: the six checks are about what is committed, not
// about whatever a developer's $ORBIT_HOME happens to overlay on top.
func loadRepoCatalog(t *testing.T, filename string) catalog {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root(t), "internal", "words", "lang", filename))
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}

	cat, err := parseCatalog(raw)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}

	if cat.keys == nil {
		cat.keys = map[string]entry{}
	}

	return cat
}

// TestEveryTranslationKeyIsHonest is the whole suite: it walks the module
// for every call to T or P, and checks es.json against what it finds.
func TestEveryTranslationKeyIsHonest(t *testing.T) {
	sites := collectCallSites(t)
	es := loadRepoCatalog(t, "es.json")
	en := loadRepoCatalog(t, "en.json")

	used := map[string]bool{}
	for _, site := range sites {
		checkCallSite(t, site, es, used)
	}

	for key, e := range es.keys {
		if !used[key] {
			t.Errorf("es.json has key %q, which no T or P call in the module uses", key)
		}

		budget := en.keys[key].Cells
		checkQuality(t, "es.json", key, "value.one", e.Value.One, budget)
		checkQuality(t, "es.json", key, "value.other", e.Value.Other, budget)
		checkQuality(t, "es.json", key, "value", e.Value.Single, budget)
	}
	// Check 6 applies to every catalogue's values, not only es.json's — an
	// en.json override is still a value a reader sees, and its budget is
	// its own declared Cells, since en.json is where that number lives.
	for key, e := range en.keys {
		checkQuality(t, "en.json", key, "value.one", e.Value.One, e.Cells)
		checkQuality(t, "en.json", key, "value.other", e.Value.Other, e.Cells)
		checkQuality(t, "en.json", key, "value", e.Value.Single, e.Cells)
	}
}

// checkCallSite runs checks 1, 2, 4 and 5 against one call site.
func checkCallSite(t *testing.T, site callSite, es catalog, used map[string]bool) {
	t.Helper()

	if !site.keyOK {
		t.Errorf("%s:%d: %s's key argument is not a string literal — a dynamic key cannot be verified against es.json", site.file, site.line, site.method)
		return
	}

	used[site.key] = true

	e, ok := es.keys[site.key]
	if !ok {
		t.Errorf("%s:%d: key %q is used but has no entry in es.json", site.file, site.line, site.key)
		return
	}

	if !site.literal {
		// The English at this call site is computed, not written out, so
		// there is nothing left to compare it against statically.
		return
	}

	switch site.method {
	case "T":
		if e.Source.Single != site.english {
			t.Errorf("%s:%d: es.json source for %q is %q, want %q — the English changed and the Spanish was not revisited", site.file, site.line, site.key, e.Source.Single, site.english)
		}

		if e.Value.Single == "" {
			t.Errorf("%s:%d: key %q has no Spanish translation in es.json", site.file, site.line, site.key)
			return
		}

		want, got := placeholders(site.english), placeholders(e.Value.Single)
		if !slices.Equal(want, got) {
			t.Errorf("%s:%d: es.json placeholders for %q are %v, want %v", site.file, site.line, site.key, got, want)
		}
	case "P":
		if e.Source.One != site.english || e.Source.Other != site.other {
			t.Errorf("%s:%d: es.json source for %q does not match the English at the call site", site.file, site.line, site.key)
		}

		if e.Value.One == "" || e.Value.Other == "" {
			t.Errorf("%s:%d: key %q has no Spanish translation in es.json for both plural forms", site.file, site.line, site.key)
			return
		}

		if want, got := placeholders(site.english), placeholders(e.Value.One); !slices.Equal(want, got) {
			t.Errorf("%s:%d: es.json placeholders for %q (one) are %v, want %v", site.file, site.line, site.key, got, want)
		}

		if want, got := placeholders(site.other), placeholders(e.Value.Other); !slices.Equal(want, got) {
			t.Errorf("%s:%d: es.json placeholders for %q (other) are %v, want %v", site.file, site.line, site.key, got, want)
		}
	}
}

// checkQuality runs check 6 against one translated form, in whichever
// catalogue it came from: NFC always, and the declared budget when one
// applies to this key.
func checkQuality(t *testing.T, catalogName, key, form, value string, budget int) {
	t.Helper()

	if value == "" {
		return
	}

	if hasCombiningMark(value) {
		t.Errorf("%s key %q (%s) is not NFC — it contains a combining mark instead of a precomposed character", catalogName, key, form)
	}

	if budget > 0 {
		if n := utf8.RuneCountInString(value); n > budget {
			t.Errorf("%s key %q (%s) is %d cells, over its budget of %d — shorten the translation, not the column", catalogName, key, form, n, budget)
		}
	}
}
