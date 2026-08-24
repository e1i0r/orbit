// Package words is the translation layer every user-facing string in the
// window goes through. It holds no language of its own: a caller resolves
// the language once — from a flag, an environment variable, a setting, or
// $LANG, by way of Resolve — and gets back an immutable Printer that is
// then passed down as a parameter, never reached for as a package global.
//
// English lives at the call site, as the second argument to T and as the
// third and fourth to P. The catalogues under lang/*.json exist to
// override that English, never to hold it: this codebase argues with
// itself in a comment above nearly every string it draws, and moving the
// words into JSON would leave every one of those arguments pointing at
// nothing. A missing or corrupt catalogue therefore costs the reader
// nothing — it yields the English that was already written into the code.
package words

import (
	"strconv"
	"strings"
)

// Printer holds one resolved language and nothing else. It is immutable:
// build one with For and pass it down explicitly. There is no package
// variable holding a Printer and no init() that picks a default language.
type Printer struct {
	lang    string
	strs    catalog
	budgets map[string]int
	// pseudo, when set, transforms the chosen text instead of returning it
	// as-is. Production code never sets this field; only the qps
	// pseudolocale built inside the test binary does (see words_test.go).
	// A field that is always nil outside `go test` is how qps stays out of
	// the shipped binary's behaviour without needing a build tag.
	pseudo func(string) string
}

// Arg is one named placeholder substitution for T or P. It is a struct
// rather than a map or a run of variadic key/value strings so that a
// translated value containing a literal "{" can never be mistaken for a
// second placeholder — the predecessor's variadic key/value scheme crashed
// on exactly that, and the fix was renaming the placeholder everywhere it
// was used.
type Arg struct {
	Name  string
	Value string
}

// For resolves lang into an immutable Printer. It never fails: a missing or
// malformed catalogue simply leaves nothing to override, so every call to T
// or P falls back to the English written at the call site.
func For(lang string) *Printer {
	return &Printer{
		lang:    lang,
		strs:    loadCatalog(lang),
		budgets: loadBudgets(),
	}
}

// T translates one string. key names it for the catalogue; english is what
// is shown when nothing overrides it — including when the catalogue is
// missing, corrupt, or simply does not carry key yet. Placeholders in the
// chosen template are written {like this} and are replaced from args.
func (p *Printer) T(key, english string, args ...Arg) string {
	template := english
	if e, ok := p.strs.keys[key]; ok && !e.Value.IsPlural && e.Value.Single != "" {
		template = e.Value.Single
	}
	return p.finish(expand(template, args))
}

// P translates one string that varies with a count. n chooses the form —
// exactly one is "one", everything else is "other" — and one, other are
// the English fallbacks the catalogue may override. {n} is available to
// the template without being named in args.
func (p *Printer) P(key string, n int, one, other string, args ...Arg) string {
	template := other
	if n == 1 {
		template = one
	}
	if e, ok := p.strs.keys[key]; ok && e.Value.IsPlural {
		switch {
		case n == 1 && e.Value.One != "":
			template = e.Value.One
		case n != 1 && e.Value.Other != "":
			template = e.Value.Other
		}
	}
	return p.finish(expand(template, withCount(args, n)))
}

// Cells is the declared budget for key, in terminal cells, or 0 when
// en.json declares none. The budget is always English's, regardless of p's
// own language: en.json is the one catalogue whose job is to state it.
func (p *Printer) Cells(key string) int {
	return p.budgets[key]
}

// finish applies the pseudolocale transform, when this Printer carries one,
// after every placeholder has already been substituted — a translated
// {name} must never itself be mistaken for a placeholder pattern.
func (p *Printer) finish(s string) string {
	if p.pseudo == nil {
		return s
	}
	return p.pseudo(s)
}

// withCount adds {n} to args unless the caller already supplied one.
func withCount(args []Arg, n int) []Arg {
	for _, a := range args {
		if a.Name == "n" {
			return args
		}
	}
	return append(args, Arg{Name: "n", Value: strconv.Itoa(n)})
}

// expand replaces every {name} in template with its Arg's value, using
// strings.NewReplacer rather than text/template: this runs inside View,
// and View runs on every update.
func expand(template string, args []Arg) string {
	if len(args) == 0 {
		return template
	}
	pairs := make([]string, 0, len(args)*2)
	for _, a := range args {
		pairs = append(pairs, "{"+a.Name+"}", a.Value)
	}
	return strings.NewReplacer(pairs...).Replace(template)
}
