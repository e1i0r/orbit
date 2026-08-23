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
	"strings"
	"testing"
)

// maxLines is the ceiling for any single Go file. It exists because the
// project this replaces had one file of 6,014 lines and another of 2,842,
// which together were a third of its running code for a single window.
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
	err := filepath.WalkDir(root(t), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "vendor") {
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
			n++
		}
		closeErr := f.Close()
		if err := scanner.Err(); err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if closeErr != nil {
			t.Fatalf("close %s: %v", path, closeErr)
		}
		if n > maxLines {
			t.Errorf("%s is %d lines, over the ceiling of %d — split it", path, n, maxLines)
		}
	}
}

func TestNoJunkDrawerPackages(t *testing.T) {
	err := filepath.WalkDir(root(t), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == ".git" || d.Name() == "vendor" {
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

// TestGoModTakesNoDependency enforces the plan's own constraint, which
// nothing else checked: the spine is standard library only, and Charm — the
// whole reason the constraint is temporary — arrives with the window in plan
// 2. A constraint stated in a document and nowhere else is one an agent
// executing a task at three in the morning will break with `go get`, in a
// commit that otherwise looks fine.
func TestGoModTakesNoDependency(t *testing.T) {
	path := filepath.Join(root(t), "go.mod")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for i, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "require") {
			t.Errorf("go.mod:%d is a dependency — %q — and plan 1 is standard library only", i+1, strings.TrimSpace(line))
		}
	}
}
