// Import direction and terminal-width discipline are enforced here, both
// checked against the source with go/parser and go/ast, rather than against
// a running build — a violation has to be visible to `go test ./...` before
// it is visible to a reviewer.
package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// layers says which of Orbit's own packages each package may import.
// A package may import anything on its list, and nothing else of Orbit's.
//
// The load-bearing entries are the absences. internal/ui does not list
// internal/record, so the window cannot append an event; it does not list
// internal/store, so the window cannot build a path under the state root;
// it does not list internal/engine, so the window cannot start a model.
// "The window derives everything and holds no authority" is those three
// missing lines, and nothing else makes it true.
var layers = map[string][]string{
	"cmd/orbit":     {"internal/cli"},
	"internal/arch": {},
	// internal/task is on internal/board's list for one function: task.Alive,
	// which reads the run marker and asks the operating system whether the
	// pid it names is still there. It is a widening, and it was argued
	// rather than assumed. The alternative was a second implementation of
	// the marker's format and the liveness check living inside
	// internal/board, and two readers of one file drift — the very class of
	// defect the record exists to prevent, arriving through the back door.
	// The direction is safe: nothing in internal/task imports
	// internal/board, so there is no cycle, and internal/ui already lists
	// internal/task because that is how every gesture reaches the function
	// its subcommand calls. What is not widened is the line above:
	// internal/board still does not append anything itself, and internal/ui
	// still cannot reach internal/record, internal/store or internal/engine.
	"internal/board":  {"internal/record", "internal/repo", "internal/store", "internal/task", "internal/view"},
	"internal/cli":    {"internal/board", "internal/engine", "internal/flow", "internal/repo", "internal/store", "internal/task", "internal/ui", "internal/view", "internal/words"},
	"internal/engine": {},
	"internal/flow":   {},
	"internal/record": {},
	"internal/repo":   {"internal/store"},
	"internal/store":  {},
	"internal/task":   {"internal/engine", "internal/flow", "internal/record", "internal/repo", "internal/store"},
	"internal/ui":     {"internal/board", "internal/flow", "internal/repo", "internal/task", "internal/ui/layout", "internal/view", "internal/words"},
	// internal/ui/layout is widened to internal/view for one reason:
	// layout.Columns plans a row's columns from the board it is about to
	// draw, and the board is []view.Task. It is a widening, and it was
	// argued rather than assumed. internal/view imports only
	// internal/record and is pure data with no behaviour of its own, so
	// there is no cycle — nothing in internal/view imports anything under
	// internal/ui — and there is nothing to leak: a Task carries no handle
	// to the record it was folded from. The alternative was a width-only
	// struct built by internal/ui and handed down, which is a second
	// description of a row living one package away from the first, and two
	// descriptions of one thing drift. What must stay true and stay tested
	// is the line below this one: no tea import anywhere in
	// internal/ui/layout, so the geometry can never become a function of
	// anything but the numbers it was given.
	"internal/ui/layout": {"internal/view"},
	"internal/view":      {"internal/record"},
	"internal/words":     {},
}

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

// TestUIMeasuresCellsNotBytes enforces the one rule that keeps internal/ui
// honest about terminal width: no file under internal/ui may call len() on
// a string, or slice a string with [a:b]. Both are the same mistake — they
// count bytes where the terminal counts cells — and the mistake is
// invisible until an accented word, a box-drawing character or a styled
// string slides one column and the whole table looks crooked. v1 measured
// its columns with len() and its misalignment was found by squinting at a
// screenshot rather than by a test. The approved calls are lipgloss.Width
// to measure and ansi.Truncate to cut.
//
// go/ast finds both shapes in one walk. len() on a slice or map is
// untouched; this rule tests the argument's inferred kind only where it is
// unambiguously a string literal, a string-typed parameter, or a call to a
// method that unambiguously returns a string, and errs toward not firing —
// a rule with a false positive gets deleted, and this one has to survive.
func TestUIMeasuresCellsNotBytes(t *testing.T) {
	modRoot := root(t)
	for _, path := range goFiles(t) {
		rel, err := filepath.Rel(modRoot, path)
		if err != nil {
			t.Fatalf("rel %s: %v", path, err)
		}
		rel = filepath.ToSlash(rel)
		if !strings.HasPrefix(rel, "internal/ui/") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		checkWidthDiscipline(t, fset, f, rel)
	}
}

// checkWidthDiscipline walks one file's AST for len() calls and slice
// expressions whose operand is unambiguously a string.
func checkWidthDiscipline(t *testing.T, fset *token.FileSet, f *ast.File, rel string) {
	t.Helper()
	strIdents := stringTypedNames(f)
	ast.Inspect(f, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.CallExpr:
			id, ok := expr.Fun.(*ast.Ident)
			if ok && id.Name == "len" && len(expr.Args) == 1 && isStringExpr(strIdents, expr.Args[0]) {
				pos := fset.Position(expr.Pos())
				t.Errorf("%s:%d calls len() on a string — use lipgloss.Width; the terminal counts cells, not bytes", rel, pos.Line)
			}
		case *ast.SliceExpr:
			if isStringExpr(strIdents, expr.X) {
				pos := fset.Position(expr.Pos())
				t.Errorf("%s:%d slices a string with [a:b] — use ansi.Truncate; the terminal counts cells, not bytes", rel, pos.Line)
			}
		}
		return true
	})
}

// stringTypedNames collects the names this file declares as string:
// function parameters and results typed string, and short variable
// declarations assigned a string literal. This is intentionally shallow —
// a false negative just means the rule looks elsewhere for a clearer case,
// and a false positive is the one failure mode this rule cannot afford.
func stringTypedNames(f *ast.File) map[string]bool {
	names := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.Field:
			if id, ok := decl.Type.(*ast.Ident); ok && id.Name == "string" {
				for _, name := range decl.Names {
					names[name.Name] = true
				}
			}
		case *ast.AssignStmt:
			if decl.Tok != token.DEFINE {
				return true
			}
			for i, lhs := range decl.Lhs {
				if i >= len(decl.Rhs) {
					continue
				}
				if lit, ok := decl.Rhs[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if id, ok := lhs.(*ast.Ident); ok {
						names[id.Name] = true
					}
				}
			}
		}
		return true
	})
	return names
}

// isStringExpr reports whether e is unambiguously a string: a string
// literal, an identifier this file recorded as string-typed, or a call to
// a method named String, Sprintf, Sprint or Sprintln.
func isStringExpr(strIdents map[string]bool, e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.BasicLit:
		return v.Kind == token.STRING
	case *ast.Ident:
		return strIdents[v.Name]
	case *ast.CallExpr:
		fn, ok := v.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		switch fn.Sel.Name {
		case "String", "Sprintf", "Sprint", "Sprintln":
			return true
		default:
			return false
		}
	default:
		return false
	}
}
