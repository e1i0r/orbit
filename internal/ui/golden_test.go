package ui

// golden_test.go holds the five frames this task is specified by, and the
// pseudolocale that proves every word in them went through the translation
// layer.
//
// The -update flag is registered here and nowhere else in the module, which
// is why it has to be run scoped: `go test ./... -update` fails in every
// package that does not know the flag. A golden regenerated without being
// read is a rubber stamp, so the flag writes files and nothing else — it
// never silences a failure.

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// update rewrites the golden files instead of comparing against them.
//
// This is not x/exp/golden's flag, and the helper below is not its
// RequireEqual, for one reason: that package derives a golden's path from
// tb.Name(), and the five files this task is specified by have names no test
// function could produce. The comparison it does — escape the control codes,
// diff, print both sides — is what is reimplemented here against a filename
// the caller chooses.
var update = flag.Bool("update", false, "rewrite the golden frames in testdata")

// golden compares one rendered frame against the file it is specified by.
func golden(t *testing.T, name string, rows []string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	got := strings.Join(rows, "\n") + "\n"
	if *update {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatalf("make testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v — run the suite with -update to write it", path, err)
	}
	if want := string(raw); want != got {
		t.Errorf("%s does not match the frame.\nwant:\n%s\ngot:\n%s", path, want, got)
	}
}

func TestTheBoardIsTheScreenItWasSpecifiedAs(t *testing.T) {
	cases := []struct {
		name string
		lang string
		w, h int
		full bool
	}{
		{"board-100x30-en", "en", 100, 30, true},
		{"board-100x30-es", "es", 100, 30, true},
		{"board-60x20-en", "en", 60, 20, true},
		{"board-empty-100x30-en", "en", 100, 30, false},
		{"board-qps-100x30", "qps", 100, 30, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := fixtureBoard(nil, 4)
			if c.full {
				b = fixtureBoard(fixtureTasks(), 4)
			}
			m := modelWith(t, printerFor(t, c.lang), b, c.w, c.h, nil)
			golden(t, c.name, renderAt(t, m, c.w, c.h))
		})
	}
}

// TestNoEnglishSurvivesThePseudolocale is the mechanical half of reading the
// qps frame. Every word below is one the window writes itself, as opposed to
// one it is handed — a repository's name, a task's id, a phase, a model, an
// elapsed time and the program's own name are data and stay as they are. A
// word from this list appearing in a pseudolocale frame is a string that
// never went through T, which is the one defect no reviewer reliably finds.
func TestNoEnglishSurvivesThePseudolocale(t *testing.T) {
	m := modelWith(t, printerFor(t, "qps"), fixtureBoard(fixtureTasks(), 4), 100, 30, nil)
	frame := strings.Join(renderAt(t, m, 100, 30), "\n")
	for _, english := range []string{
		"NEEDS YOU", "RUNNING", "TO DO", "DONE TODAY",
		"autopilot", "unread", "repos", "open", "new", "move", "filter",
		"failed", "waiting", "abandoned", "in ", "cap reached",
	} {
		if strings.Contains(frame, english) {
			t.Errorf("the pseudolocale frame still says %q, so that string never went through T:\n%s", english, frame)
		}
	}
}

// printerFor builds a Printer for one language. English and Spanish are
// embedded in internal/words; qps is generated below.
func printerFor(t *testing.T, lang string) *words.Printer {
	t.Helper()
	if lang == "qps" {
		writePseudoCatalogue(t)
	}
	return words.For(lang)
}

// writePseudoCatalogue generates qps into a temporary $ORBIT_HOME, using the
// overlay internal/words already reads a third language from.
//
// Its keys and its English come from es.json, which is the module's complete
// register of both: TestEveryTranslationKeyIsHonest fails the build if a key
// used through T is missing from that file or if its recorded source has
// drifted from the English at the call site. That is what makes reading it
// here sound rather than a second list to keep in step — and it is why a
// string that never went through T is precisely a string this catalogue has
// no entry for, and so stays English on screen.
func writePseudoCatalogue(t *testing.T) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "words", "lang", "es.json"))
	if err != nil {
		t.Fatalf("read the Spanish catalogue: %v", err)
	}
	var file struct {
		Keys map[string]struct {
			Source json.RawMessage `json:"source"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("parse the Spanish catalogue: %v", err)
	}

	keys := map[string]map[string]any{}
	for key, entry := range file.Keys {
		keys[key] = map[string]any{"value": pseudoValue(t, entry.Source)}
	}
	out, err := json.Marshal(map[string]any{"language": "Pseudo", "keys": keys})
	if err != nil {
		t.Fatalf("encode the pseudolocale: %v", err)
	}

	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "lang"), 0o750); err != nil {
		t.Fatalf("make the overlay directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "lang", "qps.json"), out, 0o600); err != nil {
		t.Fatalf("write the pseudolocale: %v", err)
	}
	t.Setenv("ORBIT_HOME", home)
}

// pseudoValue transforms one catalogue value, which is either a string or
// the {one, other} object a plural key carries.
func pseudoValue(t *testing.T, source json.RawMessage) any {
	t.Helper()
	var single string
	if err := json.Unmarshal(source, &single); err == nil {
		return pseudo(single)
	}
	var forms map[string]string
	if err := json.Unmarshal(source, &forms); err != nil {
		t.Fatalf("parse a catalogue source: %v", err)
	}
	out := map[string]any{}
	for form, text := range forms {
		out[form] = pseudo(text)
	}
	return out
}

// accents is the substitution table. Every English word has a vowel, so
// every translated string comes out of here visibly not English, while a
// placeholder — which is substituted after this runs — is left alone.
var accents = map[rune]rune{
	'a': 'á', 'e': 'é', 'i': 'í', 'o': 'ó', 'u': 'ú', 'y': 'ý',
	'c': 'ç', 'd': 'ð', 'l': 'ł', 'n': 'ñ', 'r': 'ŕ', 's': 'š', 't': 'ţ',
	'A': 'Á', 'E': 'É', 'I': 'Í', 'O': 'Ó', 'U': 'Ú', 'Y': 'Ý',
	'C': 'Ç', 'D': 'Ð', 'L': 'Ł', 'N': 'Ñ', 'R': 'Ŕ', 'S': 'Š', 'T': 'Ţ',
}

// pseudo accents one template, leaving {placeholders} whole so the values
// substituted into them stay readable.
func pseudo(s string) string {
	var out strings.Builder
	inside := false
	for _, r := range s {
		switch r {
		case '{':
			inside = true
		case '}':
			inside = false
		}
		if sub, ok := accents[r]; ok && !inside {
			out.WriteRune(sub)
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
