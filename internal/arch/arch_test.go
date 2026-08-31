// Package arch holds the project's structural rules, expressed as tests.
//
// A rule that lives only in a document is a rule that has already been
// broken, so the size ceiling and the banned package names are enforced by
// `go test ./...` like any other behaviour.
package arch

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// maxLines is the ceiling for any single Go file. It exists because the
// project this replaces had one file of 6,014 lines and another of 2,842,
// which together were a third of its running code for a single window.
//
// Blank lines do not count against it. The linter decides where those go —
// wsl and nlreturn put one between blocks and before a return — so counting
// them would make a file fail for being easier to read, and would leave a
// contributor choosing between the two rules. What the ceiling is about is
// how much a file does, and a blank line does nothing.
const maxLines = 300

// junk names that invite a file to become a drawer for anything that did not
// fit elsewhere.
var banned = []string{"utils", "util", "helpers", "helper", "common", "misc", "base"}

// root walks up from the test's working directory to the module root.
func root(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the working directory")
		}

		dir = parent
	}
}

func goFiles(t *testing.T) []string {
	t.Helper()

	var found []string

	r := root(t)

	err := filepath.WalkDir(r, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if outsideTheModule(r, path, d) {
			return filepath.SkipDir
		}

		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			found = append(found, path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	return found
}

func TestNoFileOverTheCeiling(t *testing.T) {
	for _, path := range goFiles(t) {
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}

		n := 0

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) != "" {
				n++
			}
		}

		closeErr := f.Close()

		if err := scanner.Err(); err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		if closeErr != nil {
			t.Fatalf("close %s: %v", path, closeErr)
		}

		if n > maxLines {
			t.Errorf("%s is %d lines of code and comment, over the ceiling of %d — split it", path, n, maxLines)
		}
	}
}

func TestNoJunkDrawerPackages(t *testing.T) {
	r2 := root(t)

	err := filepath.WalkDir(r2, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		if outsideTheModule(r2, path, d) {
			return filepath.SkipDir
		}

		for _, bad := range banned {
			if strings.EqualFold(d.Name(), bad) {
				t.Errorf("%s is a junk drawer with a tidy name — give it the noun of its domain", path)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

// approved is every module this repository may depend on directly, by path.
//
// The rule this replaces was "no require block at all". That was the right
// rule for the spine and the wrong one the moment the window needed a
// terminal library — but deleting it would have thrown away what it was
// actually protecting. The property was never the absence of dependencies;
// it was the absence of a dependency nobody decided on. Adding a line here
// is that decision. Do it in its own commit, with the reason in the message.
//
// Indirect requires are not listed and not checked. They are chosen by the
// modules above, not by whoever is editing this repository — which is only
// true while go.mod is tidy, so `go mod tidy -diff` runs in `make check`
// beside this test and the two are one guard.
var approved = []string{
	"charm.land/bubbles/v2",
	"charm.land/bubbletea/v2",
	"charm.land/lipgloss/v2",
	"github.com/charmbracelet/colorprofile",
	"github.com/charmbracelet/x/ansi",
	"github.com/charmbracelet/x/exp/golden",
	// modernc.org/sqlite is the derived index, and it is the pure-Go
	// translation of SQLite rather than the cgo binding on purpose:
	// releases are built with CGO_ENABLED=0 for four platforms, and a
	// dependency that needs a C toolchain would make `go install` a
	// question about the machine it is run on. What it costs is a large
	// module; what it buys is that the index stays a file the same binary
	// opens everywhere.
	"modernc.org/sqlite",
}

func TestGoModTakesOnlyApprovedDependencies(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(root(t), "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	found := map[string]bool{}
	inBlock := false

	for i, line := range strings.Split(string(body), "\n") {
		text := strings.TrimSpace(line)
		switch {
		case text == "require (":
			inBlock = true
			continue
		case inBlock && text == ")":
			inBlock = false
			continue
		case strings.HasPrefix(text, "require "):
			text = strings.TrimPrefix(text, "require ")
		case !inBlock:
			continue
		}

		if text == "" || strings.HasPrefix(text, "//") || strings.Contains(text, "// indirect") {
			continue
		}

		path := strings.Fields(text)[0]

		found[path] = true
		if !slices.Contains(approved, path) {
			t.Errorf("go.mod:%d requires %q, which nobody approved — add it to arch.approved in a commit that says why, or take it out", i+1, path)
		}
	}
}

// outsideTheModule is a directory `go build ./...` would not compile: vendor,
// and anything whose name begins with "." or "_". These walkers have to agree
// with the toolchain about where the module ends, because a directory the
// toolchain ignores can still hold a complete second copy of it — an agent's
// git worktree under .claude, an editor's index, a nested checkout — and every
// file in that copy would otherwise be read as if it were part of this one.
//
// The walk root is never outside the module, whatever it is called: a checkout
// that happens to live in a dotted directory is still the module.
func outsideTheModule(root, path string, d os.DirEntry) bool {
	if path == root || !d.IsDir() {
		return false
	}

	n := d.Name()

	return n == "vendor" || strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_")
}
