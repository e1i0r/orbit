// The terminal-width rule: what internal/ui may not do to a string.
//
// It lived in imports_test.go, which is where every arch rule started, and
// moved out when that file reached the ceiling with the map of layers
// pressed against it. Nothing about it changed in the move.
package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

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
