package ui

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// TestEveryRefusalThisWindowRaisesHasASentence reads the sentinels out of
// errsay.go rather than listing them here, because a list written by hand
// is a list that goes stale the first time a twelfth refusal is added: it
// would be raised, wrapped, drawn — and drawn in English to a Spanish
// reader, with every other test still green.
func TestEveryRefusalThisWindowRaisesHasASentence(t *testing.T) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "errsay.go", nil, 0)
	if err != nil {
		t.Fatalf("parse errsay.go: %v", err)
	}

	var declared []string

	var said string

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ValueSpec:
			// A sentinel is a var built by errors.New. Anything else this
			// file may come to declare is not a refusal and needs no
			// sentence.
			for i, name := range node.Names {
				if i < len(node.Values) && isErrorsNew(node.Values[i]) {
					declared = append(declared, name.Name)
				}
			}
		case *ast.FuncDecl:
			if node.Name.Name == "errSentence" {
				var b strings.Builder

				ast.Inspect(node.Body, func(inner ast.Node) bool {
					if id, ok := inner.(*ast.Ident); ok {
						b.WriteString(id.Name + " ")
					}

					return true
				})

				said = b.String()
			}
		}

		return true
	})

	if len(declared) == 0 || said == "" {
		t.Fatalf("read %d sentinels and %d bytes of errSentence", len(declared), len(said))
	}

	for _, name := range declared {
		if !strings.Contains(said, name+" ") {
			t.Errorf("%s is raised and errSentence has no sentence for it", name)
		}
	}
}

// isErrorsNew reports whether expr is a call to errors.New.
func isErrorsNew(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	pkg, ok := sel.X.(*ast.Ident)

	return ok && pkg.Name == "errors" && sel.Sel.Name == "New"
}

// TestARefusalIsTranslatedAndItsEvidenceIsNot.
func TestARefusalIsTranslatedAndItsEvidenceIsNot(t *testing.T) {
	english, spanish := printers(t)

	said := func(p *words.Printer, err error) string {
		t.Helper()

		return modelWith(t, p, fixtureBoard(fixtureTasks(), 4), 100, 30, &recorder{}).errSaid(err)
	}

	if en, es := said(english, errNoRecordPort), said(spanish, errNoRecordPort); en == es {
		t.Errorf("a refusal reads %q in both languages", en)
	}

	// The cause is git's, the filesystem's or an engine's own words, and it
	// is the half of the line a reader takes to whoever can fix it.
	wrapped := fmt.Errorf("%w: %w", errReadBoard, errors.New("permission denied"))
	for _, p := range []*words.Printer{english, spanish} {
		if got := said(p, wrapped); !strings.HasSuffix(got, ": permission denied") {
			t.Errorf("errSaid dropped the cause: %q", got)
		}
	}

	if got := said(spanish, errors.New("git exited 128")); got != "git exited 128" {
		t.Errorf("errSaid rewrote an error it did not raise: %q", got)
	}

	if got := said(spanish, nil); got != "" {
		t.Errorf("errSaid invented a sentence for no error: %q", got)
	}
}
