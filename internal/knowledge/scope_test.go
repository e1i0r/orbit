package knowledge

// Which facts reach a file, and in what order they are read.

import (
	"testing"
)

// TestGeneralReachesEverything: the widest scope there is, and the one that
// has nowhere else to live — "the PRs are written in English" belongs to no
// repository in particular.
func TestGeneralReachesEverything(t *testing.T) {
	general := Scope{Kind: General}

	for _, target := range []Target{
		{Repo: "/w/orbit", Path: "internal/ui/bar.go"},
		{Repo: "/w/payments", Path: "src/index.ts"},
		{Repo: "/w/payments"},
	} {
		if !general.Covers(target) {
			t.Errorf("general does not reach %s in %s", target.Path, target.Repo)
		}
	}
}

// TestALanguageReachesItsOwnFilesInEveryRepository. The key is not where the
// file is but what it is written in, which is the one scope that cuts across
// the path chain instead of hanging from it.
func TestALanguageReachesItsOwnFilesInEveryRepository(t *testing.T) {
	goFacts := Scope{Kind: Language, Lang: "go"}

	if !goFacts.Covers(Target{Repo: "/w/orbit", Path: "internal/ui/bar.go"}) {
		t.Error("a Go fact does not reach a .go file")
	}

	if !goFacts.Covers(Target{Repo: "/w/other", Path: "main.go"}) {
		t.Error("a Go fact stops at the repository it was written in")
	}

	if goFacts.Covers(Target{Repo: "/w/orbit", Path: "web/app.ts"}) {
		t.Error("a Go fact reaches a TypeScript file")
	}
}

// TestADirectoryReachesWhatIsInsideItAndNothingBeside. Scopes go downwards
// and never sideways: a rule about the ledger is not a rule about the
// module next to it.
func TestADirectoryReachesWhatIsInsideItAndNothingBeside(t *testing.T) {
	ledger := Scope{Kind: Dir, Repo: "/w/orbit", Path: "backend/ledger"}

	if !ledger.Covers(Target{Repo: "/w/orbit", Path: "backend/ledger/write.go"}) {
		t.Error("a directory does not reach a file inside it")
	}

	if !ledger.Covers(Target{Repo: "/w/orbit", Path: "backend/ledger/sql/rows.go"}) {
		t.Error("a directory does not reach a file below it")
	}

	if ledger.Covers(Target{Repo: "/w/orbit", Path: "backend/ledgerfoo/write.go"}) {
		t.Error("a directory reaches a sibling whose name merely starts the same")
	}

	if ledger.Covers(Target{Repo: "/w/orbit", Path: "backend/refund/write.go"}) {
		t.Error("a directory reaches the module beside it")
	}
}

// TestARepositoryDoesNotReachAnother is the whole of why the repository is
// part of the scope and not assumed from the caller.
func TestARepositoryDoesNotReachAnother(t *testing.T) {
	orbit := Scope{Kind: Repo, Repo: "/w/orbit"}

	if !orbit.Covers(Target{Repo: "/w/orbit", Path: "internal/ui/bar.go"}) {
		t.Error("a repository does not reach its own file")
	}

	if orbit.Covers(Target{Repo: "/w/payments", Path: "internal/ui/bar.go"}) {
		t.Error("a repository reaches a file of another repository at the same path")
	}
}

// TestASymbolReachesOnlyItself. The narrowest scope, and the one that says
// "this is true of Write and of nothing else in the file".
func TestASymbolReachesOnlyItself(t *testing.T) {
	write := Scope{Kind: Symbol, Repo: "/w/orbit", Path: "backend/ledger/write.go", Symbol: "Write"}

	if !write.Covers(Target{Repo: "/w/orbit", Path: "backend/ledger/write.go", Symbol: "Write"}) {
		t.Error("a symbol does not reach itself")
	}

	if write.Covers(Target{Repo: "/w/orbit", Path: "backend/ledger/write.go", Symbol: "Read"}) {
		t.Error("a symbol reaches its neighbour in the same file")
	}

	if write.Covers(Target{Repo: "/w/orbit", Path: "backend/ledger/write.go"}) {
		t.Error("a symbol reaches the whole file it lives in")
	}
}

// TestTheOrderIsFromTheWidestToTheNarrowest. What the agent reads last is
// what is most specific to what it is about to touch, so a fact about one
// file has the last word over a fact about every repository.
func TestTheOrderIsFromTheWidestToTheNarrowest(t *testing.T) {
	want := []Kind{General, Language, Repo, Dir, File, Symbol}

	for i := 1; i < len(want); i++ {
		wider, narrower := Scope{Kind: want[i-1]}, Scope{Kind: want[i]}
		if wider.Depth() >= narrower.Depth() {
			t.Errorf("%v is not read before %v: depths are %d and %d",
				want[i-1], want[i], wider.Depth(), narrower.Depth())
		}
	}
}

// TestADeeperDirectoryIsReadAfterAShallowerOne: two directory rules on the
// same file are ordered by how close each one is to it.
func TestADeeperDirectoryIsReadAfterAShallowerOne(t *testing.T) {
	backend := Scope{Kind: Dir, Repo: "/w/orbit", Path: "backend"}
	ledger := Scope{Kind: Dir, Repo: "/w/orbit", Path: "backend/ledger"}

	if backend.Depth() >= ledger.Depth() {
		t.Errorf("backend/ is not read before backend/ledger/: depths are %d and %d",
			backend.Depth(), ledger.Depth())
	}
}

// TestTheLanguageOfAFileComesFromItsExtension. Enough to start with, and
// the only thing available before anything has been parsed.
func TestTheLanguageOfAFileComesFromItsExtension(t *testing.T) {
	for path, want := range map[string]string{
		"internal/ui/bar.go": "go",
		"web/app.ts":         "ts",
		"web/app.tsx":        "ts",
		"scripts/run.py":     "py",
		"README.md":          "md",
		"Makefile":           "",
		"internal/ui/bar":    "",
	} {
		if got := LanguageOf(path); got != want {
			t.Errorf("LanguageOf(%q) = %q, want %q", path, got, want)
		}
	}
}
