package words

// methodvalue_test.go closes the review's Important finding: t := p.T;
// t("key", "english") reaches a real Printer method but compiles to a call
// on a plain identifier, which collectCallSites (scan_test.go) never sees
// — it only recognizes a direct x.T(...) call. Rather than resolve what a
// bound method value refers to, which is a dataflow analysis that would
// still miss cases, and each miss would be silent, this file makes the
// shape itself illegal: T and P must always be called directly, so the
// key and its English stay in the same syntax collectCallSites reads.
//
// The ban is total, not Printer-specific: this file has no type
// information (it walks go/ast, not go/types), so it cannot tell a
// Printer's T from an unrelated method or field that happens to be named
// exactly T or P. A struct field named T, read as a plain value, would
// also be flagged — that is accepted, not solved; a two-line closure is a
// fine price for a check that never lies about the case it exists for.
// The one case checked for and excluded is a type reference, because it is
// both common and unambiguous without type information: see typePosition.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// methodValueUse is one place T or P was taken as a value instead of
// called directly.
type methodValueUse struct {
	file   string
	line   int
	method string // "T" or "P"
}

// collectMethodValueUses walks every .go file in the module, internal/words
// included: aliasing a Printer's T or P is not safe anywhere, so the ban
// is not scoped to the packages that consume T and P the way
// collectCallSites' honesty checks are.
func collectMethodValueUses(t *testing.T) []methodValueUse {
	t.Helper()
	modRoot := root(t)

	var uses []methodValueUse

	err := filepath.WalkDir(modRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if outsideTheModule(modRoot, path, d) {
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()

		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		rel, err := filepath.Rel(modRoot, path)
		if err != nil {
			t.Fatalf("rel %s: %v", path, err)
		}

		uses = append(uses, scanFileForMethodValues(fset, f, filepath.ToSlash(rel))...)

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	return uses
}

// scanFileForMethodValues reports every SelectorExpr named T or P in f
// that is neither called directly nor a type reference.
func scanFileForMethodValues(fset *token.FileSet, f *ast.File, rel string) []methodValueUse {
	var (
		uses  []methodValueUse
		stack []ast.Node
	)

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}

		if sel, ok := n.(*ast.SelectorExpr); ok && (sel.Sel.Name == "T" || sel.Sel.Name == "P") {
			if !calledDirectly(sel, stack) && !typePosition(sel, stack) {
				uses = append(uses, methodValueUse{
					file:   rel,
					line:   fset.Position(sel.Pos()).Line,
					method: sel.Sel.Name,
				})
			}
		}

		stack = append(stack, n)

		return true
	})

	return uses
}

// calledDirectly reports whether sel is the Fun of the CallExpr
// immediately around it — x.T(...), not t := x.T.
func calledDirectly(sel *ast.SelectorExpr, stack []ast.Node) bool {
	if len(stack) == 0 {
		return false
	}

	call, ok := stack[len(stack)-1].(*ast.CallExpr)

	return ok && call.Fun == sel
}

// typePosition reports whether sel names a type rather than a value.
// *testing.T is the case that matters in practice: it appears in every
// test function in this module, wrapped in a *ast.StarExpr as a
// parameter's declared type, and must never be mistaken for a Printer's T
// taken as a value. The walk unwraps the sugar Go's grammar allows around
// a type (pointer, array element, ellipsis, channel value) and stops at
// the first ancestor with a named Type field, checking identity against
// that field rather than merely its presence.
func typePosition(sel *ast.SelectorExpr, stack []ast.Node) bool {
	child := ast.Expr(sel)

	for i := len(stack) - 1; i >= 0; i-- {
		switch p := stack[i].(type) {
		case *ast.StarExpr:
			if p.X != child {
				return false
			}

			child = p
		case *ast.ArrayType:
			return p.Elt == child
		case *ast.Ellipsis:
			return p.Elt == child
		case *ast.ChanType:
			return p.Value == child
		case *ast.Field:
			return p.Type == child
		case *ast.ValueSpec:
			return p.Type == child
		case *ast.TypeSpec:
			return p.Type == child
		case *ast.TypeAssertExpr:
			return p.Type == child
		case *ast.CompositeLit:
			return p.Type == child
		case *ast.MapType:
			return p.Key == child || p.Value == child
		default:
			return false
		}
	}

	return false
}

// parseSnippet parses a minimal, self-contained source file so
// scanFileForMethodValues can be tested directly, without touching the
// filesystem or walking the module.
func parseSnippet(t *testing.T, src string) (*token.FileSet, *ast.File) {
	t.Helper()

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, "snippet.go", src, 0)
	if err != nil {
		t.Fatalf("parse snippet: %v", err)
	}

	return fset, f
}

// TestMethodValueOfTIsCaught is the new error the review asked for: t :=
// p.T; t(...) must be flagged, because collectCallSites cannot see it —
// the call it eventually reaches is on a plain identifier, not a
// SelectorExpr.
func TestMethodValueOfTIsCaught(t *testing.T) {
	src := `package demo

func use(p *Printer) string {
	call := p.T
	return call("demo.untranslated", "this slipped past the scanner")
}
`
	fset, f := parseSnippet(t, src)

	uses := scanFileForMethodValues(fset, f, "demo.go")
	if len(uses) != 1 {
		t.Fatalf("got %d method-value uses, want 1: %+v", len(uses), uses)
	}

	if uses[0].method != "T" {
		t.Errorf("method = %q, want T", uses[0].method)
	}

	if uses[0].line == 0 {
		t.Errorf("line = 0, want the line p.T appears on")
	}
}

// TestMethodValueScanIgnoresATestingDotTParameter is the false-positive
// edge this scanner can and must avoid: *testing.T is a type reference,
// declared on every test function in this module, and must never be
// mistaken for a Printer's T taken as a value.
func TestMethodValueScanIgnoresATestingDotTParameter(t *testing.T) {
	src := `package demo

import "testing"

func TestSomething(t *testing.T) {
	t.Helper()
}
`
	fset, f := parseSnippet(t, src)

	uses := scanFileForMethodValues(fset, f, "demo_test.go")
	if len(uses) != 0 {
		t.Errorf("got %d method-value uses on a *testing.T parameter, want 0: %+v", len(uses), uses)
	}
}

// TestNoMethodValueOfTOrPExistsYet walks the whole module and fails the
// moment any code takes T or P as a value instead of calling it directly.
// It passes trivially today: nothing in this repository does this yet.
func TestNoMethodValueOfTOrPExistsYet(t *testing.T) {
	for _, u := range collectMethodValueUses(t) {
		t.Errorf("%s:%d: %s is taken as a value instead of being called directly — write %s(key, \"english\", ...) so the key and its English stay visible to the honesty checks", u.file, u.line, u.method, u.method)
	}
}
