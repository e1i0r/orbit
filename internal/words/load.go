package words

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
)

//go:embed lang/*.json
var embedded embed.FS

// entry is one catalogue record, keyed by translation key in catalog.keys.
// Cells is meaningful only in en.json: a catalogue for another language
// declares no budget of its own, because a budget stated twice is a budget
// that can disagree with itself.
type entry struct {
	Value  text `json:"value"`
	Source text `json:"source"`
	Cells  int  `json:"cells,omitempty"`
}

// text is either a plain string — a key used through T — or an object with
// "one" and "other" — a key used through P. Both languages this package
// ships reduce plurals to those two forms, so the object shape costs
// nothing today and a language with "few" and "many" becomes a selector
// function later rather than a migration of every call site.
type text struct {
	Single   string
	One      string
	Other    string
	IsPlural bool
}

// UnmarshalJSON accepts a bare string or a {"one":...,"other":...} object,
// so a singular entry in lang/*.json reads as plainly as its English.
func (t *text) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		t.Single = single
		return nil
	}

	var forms struct {
		One   string `json:"one"`
		Other string `json:"other"`
	}
	if err := json.Unmarshal(data, &forms); err != nil {
		return fmt.Errorf("decode catalogue text %s: %w", data, err)
	}

	t.One = forms.One
	t.Other = forms.Other
	t.IsPlural = true

	return nil
}

// catalog is one language's parsed lang/*.json: its own display name — how
// a reader of that language writes it — and every key it carries.
type catalog struct {
	language string
	keys     map[string]entry
}

// parseCatalog decodes one catalogue file. An unknown field is rejected
// rather than silently ignored, the same discipline internal/flow already
// applies to flows/*.json: a typo'd field name should be loud, not quiet.
func parseCatalog(raw []byte) (catalog, error) {
	var f struct {
		Language string           `json:"language"`
		Keys     map[string]entry `json:"keys"`
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&f); err != nil {
		return catalog{}, fmt.Errorf("parse catalogue: %w", err)
	}

	return catalog{language: f.Language, keys: f.Keys}, nil
}

// loadCatalog reads one language's catalogue: the copy embedded in the
// binary, overlaid by $ORBIT_HOME/lang/<lang>.json when that file exists.
// That overlay is how a third language is one file and no code, and how a
// reader can fix a bad translation without a build.
//
// Any failure along the way — a missing file, a trailing comma, an
// unreadable $ORBIT_HOME — yields an empty or partial catalogue rather than
// an error. A window that will not open because a translation file has a
// trailing comma is a worse outcome than a window in English.
func loadCatalog(lang string) catalog {
	result := catalog{keys: map[string]entry{}}

	if raw, err := embedded.ReadFile(path.Join("lang", lang+".json")); err == nil {
		if c, err := parseCatalog(raw); err == nil {
			result = c
			if result.keys == nil {
				result.keys = map[string]entry{}
			}
		}
	}

	if p, ok := overlayPath(lang); ok {
		if raw, err := os.ReadFile(p); err == nil {
			if c, err := parseCatalog(raw); err == nil {
				if c.language != "" {
					result.language = c.language
				}

				for k, v := range c.keys {
					result.keys[k] = v
				}
			}
		}
	}

	return result
}

// loadBudgets reads the cell budgets declared in en.json — embedded and
// overlaid the same way any catalogue is — regardless of which language a
// Printer speaks. Budgets are declared once, in English: en.json is the
// file whose job is to say how wide a column may be, not es.json's.
func loadBudgets() map[string]int {
	en := loadCatalog("en")

	budgets := make(map[string]int, len(en.keys))
	for k, e := range en.keys {
		if e.Cells > 0 {
			budgets[k] = e.Cells
		}
	}

	return budgets
}

// overlayPath is $ORBIT_HOME/lang/<lang>.json, or ~/.orbit/lang/<lang>.json
// when $ORBIT_HOME is unset. internal/words may import nothing of Orbit's
// own packages (arch.layers lists it with an empty allowance), so
// store.RootPath's fallback is written out again here rather than shared.
func overlayPath(lang string) (string, bool) {
	dir, ok := overlayDir()
	if !ok {
		return "", false
	}

	return filepath.Join(dir, lang+".json"), true
}

// overlayDir is $ORBIT_HOME/lang, or ~/.orbit/lang when $ORBIT_HOME is
// unset. It returns false only when the home directory cannot be found —
// there is no error to report it through, because a Printer never fails.
func overlayDir() (string, bool) {
	root := os.Getenv("ORBIT_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}

		root = filepath.Join(home, ".orbit")
	}

	return filepath.Join(root, "lang"), true
}
