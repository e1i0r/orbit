package ui

// Every target the form can answer reaches the form.
//
// A click is answered in two places that have to agree: internal/ui/compose
// decides which cell is which target, and mouse.go decides what the window
// does with one. The second is a switch, and a kind left off it is a button
// that is drawn, answers a hit, and does nothing at all when it is pressed.
//
// That is not a hypothetical. `ComposeClear` was added with its pill, its
// hit test, its key and its own tests, and every one of them passed while a
// click on the button did nothing, because the one line that routes it was
// never written. Nothing in the build could say so: both halves were
// correct and they had never been introduced.
//
// So the kinds are read out of internal/ui/point and looked for in the arm
// that hands them over.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryComposeTargetIsRouted.
func TestEveryComposeTargetIsRouted(t *testing.T) {
	kinds := composeKinds(t)
	if len(kinds) == 0 {
		t.Fatal("no Compose target kinds were found, so this test proves nothing")
	}

	routing, err := os.ReadFile(filepath.Join(uiRoot(t), "mouse.go"))
	if err != nil {
		t.Fatalf("read the mouse routing: %v", err)
	}

	where := string(routing)

	for _, kind := range kinds {
		if !strings.Contains(where, "point."+kind) {
			t.Errorf("point.%s is a target the form can answer and mouse.go never names it: "+
				"a click on it does nothing, and only a person pressing the button would find out", kind)
		}
	}
}

// composeKinds is every Compose* constant internal/ui/point declares.
func composeKinds(t *testing.T) []string {
	t.Helper()

	path := filepath.Join(filepath.Dir(uiRoot(t)), "ui", "point", "point.go")

	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var found []string

	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}

		for _, name := range spec.Names {
			if strings.HasPrefix(name.Name, "Compose") {
				found = append(found, name.Name)
			}
		}

		return true
	})

	return found
}

// uiRoot is this package's own directory.
func uiRoot(t *testing.T) string {
	t.Helper()

	at, err := os.Getwd()
	if err != nil {
		t.Fatalf("where am I: %v", err)
	}

	return at
}
