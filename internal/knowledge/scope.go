// Package knowledge holds what Orbit has learned: facts about the code, with
// a source and a scope, that reach the agent before it works.
//
// A fact is not a memory of the model's. The model forgets between sessions
// and forgets when it is swapped for another one; every CLI keeps its own
// notes in its own file — CLAUDE.md, AGENTS.md — and each of those is a silo
// that empties the day the engine changes. What Orbit knows is Orbit's, kept
// outside all of them, which is the whole reason this package exists.
package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Kind is how far a fact reaches, from everything to one symbol.
//
// The order is the order the agent reads them in, and it is not arbitrary:
// the last thing read is the most specific to what is about to be touched, so
// a fact about one file has the last word over a fact about every repository.
// Two of these — General and Language — are not path prefixes at all, which
// is why the chain has a shape rather than being a single ladder.
type Kind int

const (
	// General is everything, everywhere: "the PRs are written in English".
	// It belongs to no repository, so it is kept in the state root and does
	// not travel.
	General Kind = iota
	// Language is every file written in one language, in every repository:
	// "in Go, never discard an error with _". It cuts across the path chain
	// rather than hanging from it, and it does not travel either.
	Language
	// Repo is one checkout and everything in it.
	Repo
	// Dir is one directory and everything below it. Downwards only: a rule
	// about the ledger is not a rule about the module beside it.
	Dir
	// File is one file.
	File
	// Symbol is one function or type inside a file, and nothing else in it.
	Symbol
)

// A Scope is what a fact is about.
//
// Which fields are read depends on Kind, and the ones that are not read are
// empty rather than ignored: a Language scope has no repository, and saying
// so with an empty field is what keeps "every Go file" from quietly becoming
// "every Go file in this checkout".
type Scope struct {
	Kind Kind
	// Lang is the language, for Language: "go", "ts", "py".
	Lang string
	// Repo is the checkout's path, for Repo and everything under it.
	Repo string
	// Path is relative to the repository, for Dir, File and Symbol.
	Path string
	// Symbol is the function or type, for Symbol.
	Symbol string
}

// A Target is what is about to be worked on, and what the facts are asked
// for. Symbol is empty when the question is about a whole file.
type Target struct {
	Repo   string
	Path   string
	Symbol string
}

// Covers is whether this fact reaches that target.
func (s Scope) Covers(t Target) bool {
	switch s.Kind {
	case General:
		return true
	case Language:
		return s.Lang != "" && s.Lang == LanguageOf(t.Path)
	case Repo:
		return s.Repo == t.Repo
	case Dir:
		return s.Repo == t.Repo && under(s.Path, t.Path)
	case File:
		return s.Repo == t.Repo && s.Path == t.Path
	case Symbol:
		return s.Repo == t.Repo && s.Path == t.Path && s.Symbol != "" && s.Symbol == t.Symbol
	default:
		return false
	}
}

// Depth is how specific a scope is, and the order facts are read in: the
// deeper the number, the later it is read and the more it has the last word.
//
// The path kinds carry their own depth on top of their kind's, so that two
// directory rules over one file are ordered by which of them is closer to it.
// The gap between kinds is wide enough that no directory nesting can reach
// the next kind up: a rule about a file always outranks a rule about the
// directory holding it, however deeply that directory is buried.
func (s Scope) Depth() int {
	const perKind = 1000

	depth := int(s.Kind) * perKind
	if s.Kind == Dir {
		depth += strings.Count(strings.Trim(s.Path, "/"), "/") + 1
	}

	return depth
}

// under is whether a path lies inside a directory.
//
// The separator is appended before comparing, which is the whole of it: a
// prefix test alone makes backend/ledger the owner of backend/ledgerfoo, and
// a rule that leaks into the module next door is worse than no rule.
func under(dir, path string) bool {
	dir = strings.Trim(dir, "/")
	if dir == "" {
		return true
	}

	return strings.HasPrefix(strings.Trim(path, "/"), dir+"/")
}

// languages is the extensions this understands, mapped to the name a fact
// is written against. The variants of one language answer to one name — a
// rule about TypeScript is about .ts and .tsx both, and nobody writing it
// down should have to say so twice.
var languages = map[string]string{
	".go":   "go",
	".ts":   "ts",
	".tsx":  "ts",
	".js":   "js",
	".jsx":  "js",
	".py":   "py",
	".rb":   "rb",
	".rs":   "rs",
	".java": "java",
	".sql":  "sql",
	".sh":   "sh",
	".md":   "md",
	".json": "json",
	".yaml": "yaml",
	".yml":  "yaml",
}

// At is the scope a path names inside a checkout: the directory and
// everything below it, or the one file.
//
// Which of the two is read off the checkout rather than guessed from the
// spelling. A name with no dot in it is usually a directory and sometimes a
// file, and a rule filed as the wrong one is a rule that reaches half a
// project or none of it.
//
// A path the checkout does not have is refused, and the refusal says the
// rule was not written down rather than what would have happened to it. The
// alternative was keeping it for a file that might appear later, and the
// price is worse than the convenience: a typo becomes a rule that quietly
// applies to nothing, and there is nothing on any screen that would ever say
// so. Writing it after the file exists costs one more gesture; a rule that
// reaches nothing costs whatever it was supposed to prevent.
func At(repo, path string) (Scope, error) {
	path = strings.Trim(strings.TrimSpace(filepath.ToSlash(path)), "/")
	if repo == "" {
		return Scope{}, fmt.Errorf("a rule about %s needs the repository it is in", path)
	}

	// Nothing, and the root written as a dot: both are the whole checkout.
	// The dot is how somebody says so out loud, which is what they need
	// when the place arrived filled in and is wrong.
	if path == "" || path == "." {
		return Scope{Kind: Repo, Repo: repo}, nil
	}

	// Where it lands rather than what was typed: `internal/../../etc` is a
	// path out of the checkout written as one inside it, and what catches
	// that is asking how to get there from the repository rather than
	// reading the spelling.
	full := filepath.Join(repo, filepath.FromSlash(path))

	inside, err := filepath.Rel(repo, full)
	if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return Scope{}, fmt.Errorf("%s is outside the repository, so the rule was not written down", path)
	}

	at, err := os.Stat(full)
	if err != nil {
		return Scope{}, fmt.Errorf("there is no %s in this checkout, so the rule was not written down", path)
	}

	// The path as the walk found it, not as it was typed: `./internal//db`
	// and `internal/db` are the same directory, and two facts filed under
	// two spellings of it are two rules nobody can tell apart.
	where := filepath.ToSlash(inside)

	if at.IsDir() {
		return Scope{Kind: Dir, Repo: repo, Path: where}, nil
	}

	return Scope{Kind: File, Repo: repo, Path: where}, nil
}

// LanguageOf is what a file is written in, read from its extension, and
// empty when that says nothing.
//
// The extension is enough to start with and it is all there is before
// anything has been parsed. A file with no extension — a Makefile, a script
// named for what it does — answers nothing rather than guessing, and a fact
// about a language simply does not reach it.
func LanguageOf(path string) string {
	return languages[strings.ToLower(filepath.Ext(path))]
}
