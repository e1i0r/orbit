package ui

// message_test.go is one claim, made against the source rather than against a
// running window: every message this package declares is answered somewhere in
// Update.
//
// It is a file of its own because it is the only test in the package that
// parses Go instead of pressing keys, and because gesture_test.go is at the
// 300-line ceiling without it.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestEveryMessageTheWindowRaisesIsHandledInUpdate is the structural claim.
//
// It parses msg.go for every type whose name ends in Msg and ui.go for every
// case of Update's type switch, and fails on a message that is declared and
// never answered. A Cmd whose Msg falls through the switch is a gesture that
// silently does nothing — the one defect in this layer that no rendering test
// and no transition table can see, because there is nothing to look at.
func TestEveryMessageTheWindowRaisesIsHandledInUpdate(t *testing.T) {
	declared := declaredMessages(t, "msg.go")
	if len(declared) == 0 {
		t.Fatal("msg.go declares no messages, so this test is checking nothing")
	}
	handled := handledMessages(t, "ui.go")
	for _, name := range declared {
		if !handled[name] {
			t.Errorf("msg.go declares %s and Update has no case for it", name)
		}
	}
}

// declaredMessages is every type in one file whose name ends in Msg.
func declaredMessages(t *testing.T, file string) []string {
	t.Helper()
	var names []string
	ast.Inspect(parse(t, file), func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if ok && strings.HasSuffix(spec.Name.Name, "Msg") {
			names = append(names, spec.Name.Name)
		}
		return true
	})
	return names
}

// handledMessages is every local type named in a case of Update's type
// switch. Qualified names — tea.KeyPressMsg and the rest — are the event
// loop's own and are not what this test is about.
func handledMessages(t *testing.T, file string) map[string]bool {
	t.Helper()
	handled := map[string]bool{}
	for _, decl := range parse(t, file).Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Update" {
			continue
		}
		ast.Inspect(fn, func(n ast.Node) bool {
			clause, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range clause.List {
				if ident, ok := expr.(*ast.Ident); ok {
					handled[ident.Name] = true
				}
			}
			return true
		})
	}
	return handled
}

func parse(t *testing.T, file string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	return f
}
