package words

import (
	"os"
	"sort"
	"strings"
)

// Resolve picks a language from four sources, in priority order: an
// explicit flag, an explicit environment variable, a saved setting, and
// finally the process's own locale, $LANG. English is the answer when none
// of them says otherwise.
//
// flag, env and setting are already resolved by the caller — the --lang
// flag's value, $ORBIT_LANG, the "lang" line of settings — because a
// package that holds no language of its own should not decide where one
// comes from either. $LANG is the one source Resolve reads for itself: it
// is a property of the process, not a setting Orbit owns.
func Resolve(flag, env, setting string) string {
	for _, v := range []string{flag, env, setting, os.Getenv("LANG")} {
		if lang := normalizeLocale(v); lang != "" {
			return lang
		}
	}

	return "en"
}

// normalizeLocale reduces a POSIX-style locale name — es_MX.UTF-8@euro,
// es-419, C, POSIX — to the lowercase language code this package's
// catalogues are keyed by. It returns "" for a value that names no usable
// language: "C" and "POSIX" are the locale that means "no locale", and an
// empty source is treated the same as one that was never set.
func normalizeLocale(v string) string {
	v = strings.TrimSpace(v)
	if i := strings.IndexByte(v, '@'); i >= 0 {
		v = v[:i]
	}

	if i := strings.IndexByte(v, '.'); i >= 0 {
		v = v[:i]
	}

	if i := strings.IndexAny(v, "_-"); i >= 0 {
		v = v[:i]
	}

	switch strings.ToLower(v) {
	case "", "c", "posix":
		return ""
	default:
		return strings.ToLower(v)
	}
}

// pseudoCode is the language code the qps pseudolocale would use if it
// were ever a file — it never is (pseudo_test.go builds it entirely
// inside the test binary), but Available refuses this code by name too,
// so a qps.json dropped into an overlay by mistake still could not reach
// a real user, regardless of what display name that file claimed for
// itself.
const pseudoCode = "qps"

// Available lists the languages the picker may offer: every catalogue this
// binary embeds, plus any $ORBIT_HOME/lang/*.json overlay, each returned
// the way a reader of that language writes its own name. Available reads
// that name out of the file itself rather than a table baked into this
// package — which is what keeps a third language to one file and no code.
func Available() []string {
	codes := map[string]bool{}

	if entries, err := embedded.ReadDir("lang"); err == nil {
		for _, e := range entries {
			if code, ok := languageCode(e.Name()); ok && code != pseudoCode {
				codes[code] = true
			}
		}
	}

	if dir, ok := overlayDir(); ok {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}

				if code, ok := languageCode(e.Name()); ok && code != pseudoCode {
					codes[code] = true
				}
			}
		}
	}

	sorted := make([]string, 0, len(codes))
	for code := range codes {
		sorted = append(sorted, code)
	}

	sort.Strings(sorted)

	names := make([]string, 0, len(sorted))
	for _, code := range sorted {
		name := loadCatalog(code).language
		if name == "" {
			name = code
		}

		names = append(names, name)
	}

	return names
}

// languageCode reports the language code a catalogue filename names, i.e.
// "es" for "es.json".
func languageCode(filename string) (string, bool) {
	if !strings.HasSuffix(filename, ".json") {
		return "", false
	}

	return strings.TrimSuffix(filename, ".json"), true
}
