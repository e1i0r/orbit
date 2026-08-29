package words

// words_test.go fails the build the instant es.json falls out of step with
// the code, in either direction. It was written when nothing outside this
// package called T or P; it now holds some six hundred call sites to
// account, and every one of the seven checks below was added because the
// mistake it looks for had already shipped.

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
// through loadCatalog: the checks are about what is committed, not
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

		checkTranslated(t, key, e)

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

// borrowedWholesale is every key whose Spanish is its English on purpose.
//
// It is short and it is named because the alternative is a check nobody
// can turn on: a translation identical to its source is what a Spanish
// string sitting in the English slot looks like from here, and that mistake
// shipped — "intento 1", "etapa", "costo", the whole cheat sheet — passing
// all six checks above, because a key whose source is already Spanish
// agrees with itself perfectly.
var borrowedWholesale = map[string]string{
	"state.phase_of":           "placeholders and a slash, no prose",
	"band.repo_filter_tag":     "placeholders and a colon, no prose",
	"mcp.install_wrote":        "placeholders and a dash, no prose",
	"mcp.install_failed":       "placeholders and a colon, no prose",
	"flows.thinking_badge":     "a label that is itself a loanword, plus a placeholder",
	"board.col_id":             "ID is ID",
	"compose.id":               "ID is ID",
	"compose.url":              "url is url",
	"board.col_repo":           "repo is repo, and it is a column head three cells wide",
	"compose.tab_manual":       "a tab number and a word Spanish spells the same",
	"tab.gates":                "the engines call them gates and so does the record",
	"tab.thinking":             "the dial, the flag and the engines all say thinking",
	"start.thinking_hint":      "the dial, the flag and the engines all say thinking",
	"compose.thinking":         "the dial, the flag and the engines all say thinking",
	"overview.thinking":        "the dial, the flag and the engines all say thinking",
	"key.supervisor":           "supervisor is supervisor",
	"overview.action_merge_pr": "merge and PR are what the forge calls them",
}

// checkTranslated is check 7: a value identical to its source is either a
// word Spanish borrowed whole, and named above with the reason, or it is
// English that was never translated — or, worse, Spanish that was written
// into the English slot and is therefore drawn to a reader of English.
func checkTranslated(t *testing.T, key string, e entry) {
	t.Helper()

	if _, borrowed := borrowedWholesale[key]; borrowed {
		return
	}

	const why = "es.json value for %q is identical to its source, %q — either the Spanish is missing, or the source is not English; if the word is the same in both, say so in borrowedWholesale"

	if e.Source.Single != "" && e.Source.Single == e.Value.Single {
		t.Errorf(why, key, e.Source.Single)
	}

	if e.Source.One != "" && e.Source.One == e.Value.One && e.Source.Other == e.Value.Other {
		t.Errorf(why, key, e.Source.One)
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
