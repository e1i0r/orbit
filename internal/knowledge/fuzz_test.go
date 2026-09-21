package knowledge

// The two readings a rule's reach depends on, and the file it is written in.
//
// A rule file travels with the clone it is about, so it is edited by hand,
// written by an older Orbit, and merged by git like any other file. And what
// it says about where it applies decides which runs are told about it — a
// rule that leaks into the module next door is worse than no rule, because
// the reader who wrote it believes it is somewhere else.

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzARuleIsInsideTheDirectoryOrItIsNot.
//
// A prefix test alone makes backend/ledger the owner of backend/ledgerfoo.
// The separator is what tells a directory from a name that merely starts the
// same way, and there is no spelling of either that may get past it.
func FuzzARuleIsInsideTheDirectoryOrItIsNot(f *testing.F) {
	f.Add("backend/ledger", "backend/ledger/postings.go")
	f.Add("backend/ledger", "backend/ledgerfoo/postings.go")
	f.Add("", "anything.go")
	f.Add("/backend/", "/backend/a.go")
	f.Add("a", "a")
	f.Add("a", "ab")

	f.Fuzz(func(t *testing.T, dir, path string) {
		if !under(dir, path) {
			return
		}

		clean := strings.Trim(dir, "/")
		if clean == "" {
			// A rule about no directory is about the whole checkout, which
			// is what an empty scope already means.
			return
		}

		inside := strings.Trim(path, "/")
		if !strings.HasPrefix(inside, clean+"/") {
			t.Errorf("under(%q, %q) said yes, and %q does not sit under %q", dir, path, inside, clean)
		}

		// The directory itself is not inside itself: a rule about a folder
		// is about what the folder holds.
		if inside == clean {
			t.Errorf("under(%q, %q) read the directory as being inside itself", dir, path)
		}
	})
}

// FuzzAScopeStaysInsideItsCheckout.
//
// The path is whatever a reader typed, and `internal/../../etc` is a path
// out of the checkout written as one inside it. What catches that is asking
// how to get there from the repository rather than reading the spelling, so
// what comes back is either a scope that names somewhere inside or a
// refusal — never a scope pointing at somebody else's disk.
func FuzzAScopeStaysInsideItsCheckout(f *testing.F) {
	f.Add("/w/acme", "internal/db")
	f.Add("/w/acme", "../../etc/passwd")
	f.Add("/w/acme", "")
	f.Add("/w/acme", ".")
	f.Add("", "internal/db")
	f.Add("/w/acme", "/etc/passwd")
	f.Add("/w/acme", "internal/db/pr.go")

	f.Fuzz(func(t *testing.T, repo, path string) {
		sc, err := At(repo, path)
		if err != nil {
			return
		}

		if sc.Repo != repo {
			t.Errorf("At(%q, %q) answered a scope in %q", repo, path, sc.Repo)
		}

		if sc.Path == "" {
			// The whole checkout, which is what nothing and a dot mean.
			return
		}

		if filepath.IsAbs(sc.Path) {
			t.Errorf("At(%q, %q) answered the absolute path %q", repo, path, sc.Path)
		}

		if sc.Path == ".." || strings.HasPrefix(sc.Path, "../") {
			t.Errorf("At(%q, %q) answered %q, which climbs out of the checkout", repo, path, sc.Path)
		}

		if strings.Contains(sc.Path, "\\") {
			t.Errorf("At(%q, %q) answered %q, which is not written the way a scope is", repo, path, sc.Path)
		}
	})
}

// FuzzARuleFileReadsBackAsWhatItSays.
//
// The file is edited by hand, written by an older Orbit, and merged by git.
// A reading that fell over on one of them would take the checkout's whole
// knowledge with it — so what it answers is either a rule or a refusal, and
// a rule it answers is one it could write again.
func FuzzARuleFileReadsBackAsWhatItSays(f *testing.F) {
	for _, seed := range []string{
		"",
		"---\nid: abc12345\nsource: human\n---\n\nAmounts are cents.\n",
		"---\n---\n",
		"---\nid: abc12345\nscope: dir\nwhere: internal/db\n---\n\nEvery query is a constant.\n",
		"no front matter at all",
		"---\nid: abc12345\nstate: off\nwhy: too wide\n---\n\nA rule.\n",
		"---\nid: \n---\n\n\n",
		"--- \n id : x \n---\nbody",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, body string) {
		got, err := decode(body, "internal/db", "/w/acme")
		if err != nil {
			return
		}

		if !utf8.ValidString(body) {
			return
		}

		// A rule with nothing to say is not a rule: it would reach every
		// phase's prompt as a blank line somebody has to wonder about.
		if strings.TrimSpace(got.Phrase) == "" {
			t.Errorf("decode(%q) answered a rule that says nothing", body)
		}

		// And what it answered is something it can write again, which is
		// what says a file survives being read and saved.
		again, err := decode(encode(got), "internal/db", "/w/acme")
		if err != nil {
			t.Fatalf("what decode(%q) answered cannot be read back: %v", body, err)
		}

		if again.Phrase != got.Phrase {
			t.Errorf("decode(%q) reads as %q and writes back as %q", body, got.Phrase, again.Phrase)
		}

		if again.Scope != got.Scope {
			t.Errorf("decode(%q) is scoped %+v and writes back as %+v", body, got.Scope, again.Scope)
		}
	})
}
