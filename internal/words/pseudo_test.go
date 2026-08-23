package words

// pseudo_test.go builds the qps pseudolocale: an artificial language
// generated at test time from the English fallbacks, never shipped and
// never offered by Available (see TestAvailableDoesNotOfferThePseudolocale
// in locale_test.go). It exists purely inside the test binary — that is
// what "registered only under go test" means — so it costs the production
// binary nothing and needs no translator to catch two different bugs:
// anything that overflows a golden assumed English-length text, and
// anything still rendering as plain ASCII never went through T.

import (
	"strings"
	"testing"
)

// forPseudo builds a Printer that speaks qps: every string T or P would
// otherwise show in English, expanded and accented instead.
func forPseudo() *Printer {
	return &Printer{lang: "qps", pseudo: pseudoTransform}
}

// pseudoTransform expands s by roughly 40%, replaces its vowels with
// accented lookalikes, and wraps the result in brackets.
func pseudoTransform(s string) string {
	return "[" + pad(accentVowels(s), 0.4) + "]"
}

var vowelReplacer = strings.NewReplacer(
	"a", "á", "e", "é", "i", "í", "o", "ó", "u", "ú",
	"A", "Á", "E", "É", "I", "Í", "O", "Ó", "U", "Ú",
)

func accentVowels(s string) string {
	return vowelReplacer.Replace(s)
}

// pad grows s by repeating its own runes, so a pseudo string is reliably
// longer than the English it was built from without needing filler text
// of its own.
func pad(s string, ratio float64) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	extra := int(float64(len(runes)) * ratio)
	if extra == 0 {
		extra = 1
	}
	fill := make([]rune, extra)
	for i := range fill {
		fill[i] = runes[i%len(runes)]
	}
	return s + string(fill)
}

func TestPseudolocaleWrapsAndAccentsAndGrows(t *testing.T) {
	p := forPseudo()
	got := p.T("band.needs", "NEEDS YOU")
	if !strings.HasPrefix(got, "[") || !strings.HasSuffix(got, "]") {
		t.Errorf("T() = %q, want it wrapped in brackets", got)
	}
	if got == "["+"NEEDS YOU"+"]" {
		t.Errorf("T() = %q, want vowels replaced with accented lookalikes", got)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(got, "["), "]")
	if len(inner) <= len("NEEDS YOU") {
		t.Errorf("T() = %q, want it longer than the English original", got)
	}
}

func TestPseudolocaleStillExpandsPlaceholders(t *testing.T) {
	p := forPseudo()
	got := p.T("repo.tasks", "{repo} · {n} tasks", Arg{Name: "repo", Value: "orbit"}, Arg{Name: "n", Value: "4"})
	if strings.Contains(got, "{repo}") || strings.Contains(got, "{n}") {
		t.Errorf("T() = %q, a placeholder was never substituted", got)
	}
	if !strings.Contains(got, "órbít") {
		t.Errorf("T() = %q, want it to contain the substituted repo name, itself pseudo-transformed", got)
	}
}

func TestPseudolocaleAffectsPToo(t *testing.T) {
	p := forPseudo()
	got := p.P("task.count", 2, "{n} task", "{n} tasks")
	if !strings.HasPrefix(got, "[") || !strings.HasSuffix(got, "]") {
		t.Errorf("P() = %q, want it wrapped in brackets", got)
	}
}
