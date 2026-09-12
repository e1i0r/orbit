package verb

// Every verb says what it is, in both languages, and so does every field
// it takes. This calls each of them: a sentence nobody reads is a sentence
// nobody checks, and the catalogue test can only hold to account the keys
// a call site names — a verb whose About is nil fails before any surface
// has to invent its words.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// TestEveryVerbSaysItInBothLanguages.
func TestEveryVerbSaysItInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		p := words.For(lang)

		for _, v := range Every() {
			if v.About == nil {
				t.Errorf("%q says nothing about what it is", v.Name)
				continue
			}

			if s := v.About(p); s == "" {
				t.Errorf("%q says %q in %s, which is nothing", v.Name, s, lang)
			}

			for _, f := range v.Takes {
				if f.About == nil {
					t.Errorf("%q takes a field that does not say what it is", v.Name)
					continue
				}

				if s := f.About(p); s == "" {
					t.Errorf("%q asks for %q in %s with no words", v.Name, f.Name, lang)
				}
			}
		}
	}
}
