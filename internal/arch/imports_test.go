// Import direction is enforced here, checked against the source with
// go/parser and go/ast rather than against a running build — a violation has
// to be visible to `go test ./...` before it is visible to a reviewer.
//
// The terminal-width rule was in this file too, until the two of them
// together reached the size ceiling and the map of layers had no room left
// to take a line. It is the same kind of rule read the same way and it is
// about a different thing entirely, so it is in width_test.go beside this
// one.
package arch

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// modulePath prefixes every import that is one of Orbit's own packages
// rather than a third party's.
const modulePath = "github.com/e1i0r/orbit"

// TestImportsFollowTheLayers walks every Go file, maps it to its package
// directory relative to the module root, and fails any import of
// github.com/e1i0r/orbit/... whose target is not on that package's list in
// arch.layers. A package that exists and has no entry is a failure, not a
// pass — a new package must be placed in the layering deliberately.
func TestImportsFollowTheLayers(t *testing.T) {
	modRoot := root(t)
	for _, path := range goFiles(t) {
		rel, err := filepath.Rel(modRoot, filepath.Dir(path))
		if err != nil {
			t.Fatalf("rel %s: %v", path, err)
		}

		pkg := filepath.ToSlash(rel)

		allowed, ok := layers[pkg]
		if !ok {
			t.Errorf("%s belongs to package %q, which has no entry in arch.layers — place it in the layering", path, pkg)
			continue
		}

		fset := token.NewFileSet()

		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == modulePath || !strings.HasPrefix(importPath, modulePath+"/") {
				continue
			}

			target := strings.TrimPrefix(importPath, modulePath+"/")
			if !slices.Contains(allowed, target) {
				t.Errorf("%s imports %q, which %s does not list in arch.layers", path, importPath, pkg)
			}
		}
	}
}

// teaModule is the terminal's event loop. internal/ui/layout may not import
// it, in any package under it, in test files included.
const teaModule = "charm.land/bubbletea/v2"

// TestLayoutNeverImportsTheEventLoop is the other half of the widening
// argued in arch.layers: internal/ui/layout may read a view.Task, and it may
// not read a terminal.
//
// The property is that layout is a pure function of the numbers it is given.
// An import of bubbletea is the one thing that could quietly end that — a
// window size read from a message rather than taken as a parameter, a
// background colour asked of the terminal, a command returned from what is
// supposed to be arithmetic — and none of it would look wrong in a diff. It
// would show up as a layout that cannot be table-tested, which is a thing
// you notice a plan later.
//
// lipgloss is deliberately not banned: measuring a string in cells is the
// one terminal fact the arithmetic genuinely needs, it asks the terminal
// nothing to answer, and the alternative is counting bytes, which is the
// mistake TestUIMeasuresCellsNotBytes exists to prevent.
func TestLayoutNeverImportsTheEventLoop(t *testing.T) {
	modRoot := root(t)
	for _, path := range goFiles(t) {
		rel, err := filepath.Rel(modRoot, path)
		if err != nil {
			t.Fatalf("rel %s: %v", path, err)
		}

		if !strings.HasPrefix(filepath.ToSlash(rel), "internal/ui/layout/") {
			continue
		}

		fset := token.NewFileSet()

		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == teaModule || strings.HasPrefix(importPath, teaModule+"/") {
				t.Errorf("%s imports %q — the layout is a pure function of the width and the height it is given, and an event loop is how that stops being true", rel, importPath)
			}
		}
	}
}
