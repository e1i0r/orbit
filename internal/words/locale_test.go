package words

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveStripsAPOSIXLocale is the seventh check: Resolve must reduce
// a POSIX-style locale name to the two-letter code the catalogues use,
// wherever in the precedence chain it appears.
func TestResolveStripsAPOSIXLocale(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{"territory, encoding and variant", "es_MX.UTF-8@euro", "es"},
		{"the C locale", "C", "en"},
		{"the POSIX locale", "POSIX", "en"},
		{"empty", "", "en"},
		{"BCP 47 region tag", "es-419", "es"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("LANG", "")
			if got := Resolve(c.value, "", ""); got != c.want {
				t.Errorf("Resolve(%q, \"\", \"\") = %q, want %q", c.value, got, c.want)
			}
		})
	}
}

// TestResolveFollowsPrecedence checks that flag beats env beats setting
// beats $LANG, and that an unset chain falls back to English.
func TestResolveFollowsPrecedence(t *testing.T) {
	t.Setenv("LANG", "es_ES.UTF-8")
	if got := Resolve("", "", ""); got != "es" {
		t.Errorf("Resolve with only $LANG set = %q, want es", got)
	}
	if got := Resolve("", "", "fr"); got != "fr" {
		t.Errorf("setting did not override $LANG: got %q, want fr", got)
	}
	if got := Resolve("", "de", "fr"); got != "de" {
		t.Errorf("env did not override setting: got %q, want de", got)
	}
	if got := Resolve("it", "de", "fr"); got != "it" {
		t.Errorf("flag did not override env: got %q, want it", got)
	}

	t.Setenv("LANG", "")
	if got := Resolve("", "", ""); got != "en" {
		t.Errorf("Resolve with nothing set = %q, want en", got)
	}
}

// TestAvailableNamesEachLanguageInItself checks that the two shipped
// catalogues are offered, each under the name its own file declares —
// "English" and "Español", not "en" and "es".
func TestAvailableNamesEachLanguageInItself(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	got := Available()
	want := []string{"English", "Español"}
	if len(got) != len(want) {
		t.Fatalf("Available() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Available()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestAvailableDoesNotOfferThePseudolocale confirms that qps, which exists
// only inside the test binary, is never something Available offers a user
// — even if a qps.json file were dropped into an overlay by mistake.
// Checking Available() against the literal string "qps" would not catch
// that: Available returns each catalogue's self-declared display name, not
// its code, so a qps.json claiming to be called "Pseudo" would sail
// straight through a check that only ever looked for "qps". This test
// plants exactly that file and asserts its display name never appears.
func TestAvailableDoesNotOfferThePseudolocale(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)
	langDir := filepath.Join(home, "lang")
	if err := os.MkdirAll(langDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{"language": "Pseudo", "keys": {}}`
	if err := os.WriteFile(filepath.Join(langDir, pseudoCode+".json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	for _, name := range Available() {
		if name == "Pseudo" {
			t.Errorf("Available() = %v, includes the qps overlay's display name — the pseudolocale must never reach the picker, no matter what it calls itself", Available())
		}
	}
}

// TestAvailableFindsAnOverlaidThirdLanguage is the "one file and no code"
// claim, checked: dropping fr.json under $ORBIT_HOME/lang is enough for
// Available to list it, under the name the file itself declares.
func TestAvailableFindsAnOverlaidThirdLanguage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)
	langDir := filepath.Join(home, "lang")
	if err := os.MkdirAll(langDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{"language": "Français", "keys": {}}`
	if err := os.WriteFile(filepath.Join(langDir, "fr.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := Available()
	found := false
	for _, name := range got {
		if name == "Français" {
			found = true
		}
	}
	if !found {
		t.Errorf("Available() = %v, want it to include Français from the overlay", got)
	}
}
