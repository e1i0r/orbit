package arch

// A package is entered through its doors.
//
// Go has no visibility inside a package: every file sees every other, so
// "this file is the entrance and the rest are its workings" is a convention,
// and a convention nobody checks is gone in three months. This checks it.
//
// The rule is one line: everything a package exports is declared in one of
// its doors. A door is an action the world can ask for — Open, Key, Apply,
// View — or the vocabulary those actions share. Everything else in the
// directory is a satellite: it holds the workings of one door and is named
// after it, and it exports nothing.
//
// The list below is also the index of each package. Reading the doors of
// internal/ui/settings tells you what that screen does without opening a
// file, which is the other half of why the rule is worth keeping.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestEveryExportedNameIsBehindADoor.
func TestEveryExportedNameIsBehindADoor(t *testing.T) {
	root := root(t)

	for pkg, allowed := range doors {
		dir := filepath.Join(root, pkg)
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("%s is listed as taken apart and is not there: %v", pkg, err)
			continue
		}

		for _, file := range sourceIn(t, dir) {
			name := filepath.Base(file)
			if slices.Contains(allowed, name) {
				continue
			}

			for _, exported := range exportedIn(t, file) {
				t.Errorf("%s/%s exports %s, and it is not a door of %s — put it in one of %v, or leave it unexported",
					pkg, name, exported, pkg, allowed)
			}
		}
	}
}

// exportedIn is every name a file declares that another package could say.
func exportedIn(t *testing.T, path string) []string {
	t.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var out []string

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// A method on an exported type is part of that type's
			// surface, so it belongs in the door the type was declared
			// in — the same rule, one level down.
			if d.Name.IsExported() {
				out = append(out, d.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				out = append(out, exportedNames(spec)...)
			}
		}
	}

	return out
}

// exportedNames is what one const, var or type declaration puts in reach.
func exportedNames(spec ast.Spec) []string {
	var out []string

	switch s := spec.(type) {
	case *ast.TypeSpec:
		if s.Name.IsExported() {
			out = append(out, s.Name.Name)
		}
	case *ast.ValueSpec:
		for _, n := range s.Names {
			if n.IsExported() {
				out = append(out, n.Name)
			}
		}
	}

	return out
}

// sourceIn is the source of one package, without its tests: a test file is
// not a door and nothing outside the package can see it.
func sourceIn(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	var out []string

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		out = append(out, filepath.Join(dir, name))
	}

	return out
}
