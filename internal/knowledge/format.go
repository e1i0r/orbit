package knowledge

// What a rule looks like in a file.
//
// A header of plain `key: value` lines, and then the sentence. The sentence
// is the body rather than another field because it is the part written for a
// person: it is what shows up first when somebody opens the file, and what a
// reviewer reads in the diff.
//
// The header is parsed here rather than by a YAML library because these are
// a dozen known keys on one line each, and a dependency that could bring in
// anchors, multi-line scalars and type coercion is a larger surface than the
// thing it would parse.

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// fence is what opens and closes the header.
const fence = "---"

// The header's keys. Everything the scope needs is written out even when the
// file's own location implies it: a rule that has been moved by hand still
// says what it is about, and a reader of one file does not have to work out
// where it sits to know what it covers.
const (
	keyID     = "id"
	keyScope  = "scope"
	keySource = "source"
	keyRef    = "ref"
	keyAt     = "at"
	keyAction = "action"
	keyCheck  = "check"
	keyPath   = "path"
	keySymbol = "symbol"
	keyLang   = "lang"
	keyOff    = "off"
	keyState  = "state"
	keyWhy    = "why"
	keyReview = "review"
	keyUsed   = "used"
)

var kindNames = map[Kind]string{
	General: "general", Language: "lang", Repo: "repo",
	Dir: "dir", File: "file", Symbol: "symbol",
}

var stateNames = map[State]string{
	Active: "active", Paused: "paused", Off: "off",
}

var sourceNames = map[Source]string{
	FromCode: "code", Human: "human", FromRecord: "record",
	FromProduction: "production", FromDocs: "docs", FromHistory: "history",
}

// encode writes a rule out.
func encode(f Rule) string {
	var b strings.Builder

	b.WriteString(fence + "\n")
	line(&b, keyID, f.ID)
	line(&b, keyScope, kindNames[f.Scope.Kind])
	line(&b, keySource, sourceNames[f.Source])
	line(&b, keyRef, f.Ref)
	line(&b, keyPath, f.Scope.Path)
	line(&b, keySymbol, f.Scope.Symbol)
	line(&b, keyLang, f.Scope.Lang)

	if !f.At.IsZero() {
		line(&b, keyAt, f.At.UTC().Format(time.RFC3339))
	}

	if f.Stops {
		line(&b, keyAction, "stop")
	}

	line(&b, keyCheck, f.Check)

	// Nothing for a rule that applies: it is the ordinary case and the zero
	// value, and a header that said so on every file would be a line
	// somebody learns to stop reading.
	if f.State != Active {
		line(&b, keyState, stateNames[f.State])
		line(&b, keyWhy, f.Why)
	}

	if f.Review {
		line(&b, keyReview, "true")
	}

	if f.Used > 0 {
		line(&b, keyUsed, strconv.Itoa(f.Used))
	}

	b.WriteString(fence + "\n\n" + strings.TrimSpace(f.Phrase) + "\n")

	return b.String()
}

// line writes one header entry, and nothing for a value there is not.
func line(b *strings.Builder, key, value string) {
	if value != "" {
		b.WriteString(key + ": " + value + "\n")
	}
}

// decode reads a rule back. where is the file's path under its root and repo
// is the checkout it belongs to, both used to fill in what the header leaves
// out — which is how a file dropped in by hand works with a header of two
// lines.
func decode(body, where, repo string) (Rule, error) {
	head, phrase := split(body)
	if phrase == "" {
		return Rule{}, fmt.Errorf("says nothing")
	}

	f := Rule{ID: head[keyID], Phrase: phrase, Ref: head[keyRef], Check: head[keyCheck]}

	source, ok := sourceNamed(head[keySource])
	if !ok {
		return Rule{}, fmt.Errorf("comes from %q, which is not a source", head[keySource])
	}

	f.Source = source
	f.Stops = head[keyAction] == "stop"
	f.State = stateNamed(head)
	f.Why = head[keyWhy]
	f.Review = head[keyReview] == "true"
	f.Used, _ = strconv.Atoi(head[keyUsed]) //nolint:errcheck // a count nobody wrote is none

	if at := head[keyAt]; at != "" {
		when, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return Rule{}, fmt.Errorf("entered at %q, which is not a time: %w", at, err)
		}

		f.At = when
	}

	f.Scope = scopeFrom(head, where, repo)

	return f, f.Validate()
}

// stateNamed is where the header says the rule stands, and active when it
// says nothing — which is what a file somebody wrote by hand means.
//
// `off: true` is read as switched off, because that is what every file
// written before there were three states says, and a rule somebody
// disagreed with silently coming back on would be the worst way to find out
// this changed.
func stateNamed(head map[string]string) State {
	for state, spelled := range stateNames {
		if head[keyState] == spelled {
			return state
		}
	}

	if head[keyOff] == "true" {
		return Off
	}

	return Active
}

// scopeFrom builds the scope from what the header says, falling back to
// where the file sits for whatever it leaves out.
func scopeFrom(head map[string]string, where, repo string) Scope {
	sc := Scope{
		Kind:   kindNamed(head[keyScope], where, repo),
		Lang:   head[keyLang],
		Repo:   repo,
		Path:   head[keyPath],
		Symbol: head[keySymbol],
	}

	dir := strings.TrimSuffix(where, "/"+lastSegment(where))
	if dir == where {
		dir = ""
	}

	switch sc.Kind {
	case Language:
		if sc.Lang == "" {
			sc.Lang = lastSegment(dir)
		}

		sc.Repo = ""
	case General:
		sc.Repo = ""
	case Dir, File, Symbol:
		if sc.Path == "" {
			sc.Path = dir
		}
	}

	return sc
}

// kindNamed reads the kind the header names, and works it out from where the
// file sits when the header does not say.
func kindNamed(name, where, repo string) Kind {
	for kind, spelled := range kindNames {
		if name == spelled {
			return kind
		}
	}

	switch {
	case repo == "" && strings.HasPrefix(where, langDir+"/"):
		return Language
	case repo == "":
		return General
	case !strings.Contains(where, "/"):
		return Repo
	default:
		return Dir
	}
}

// sourceNamed reads a source, and says so when the name is not one. There is
// no default: a rule whose origin cannot be read is a rule nobody can trace,
// and the whole reason this is kept outside the model is that it can be.
func sourceNamed(name string) (Source, bool) {
	for source, spelled := range sourceNames {
		if name == spelled {
			return source, true
		}
	}

	return unsourced, false
}

// split takes the header apart from the sentence.
func split(body string) (map[string]string, string) {
	head := map[string]string{}

	rest := strings.TrimPrefix(strings.TrimSpace(body), fence)
	if rest == strings.TrimSpace(body) {
		// No header at all: the whole file is the sentence.
		return head, strings.TrimSpace(body)
	}

	end := strings.Index(rest, "\n"+fence)
	if end < 0 {
		return head, ""
	}

	for _, l := range strings.Split(rest[:end], "\n") {
		key, value, found := strings.Cut(l, ":")
		if !found {
			continue
		}

		head[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return head, strings.TrimSpace(rest[end+len("\n"+fence):])
}

// lastSegment is the last part of a slash path, and the whole of it when
// there is only one.
func lastSegment(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}

	return path
}
