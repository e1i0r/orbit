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
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryMessageTheWindowRaisesIsHandledInUpdate is the structural claim.
//
// It reads every non-test file in this package for types whose name ends in
// Msg, reads Update's type switch for the ones it answers, and fails on a
// message that is declared and never answered. A Cmd whose Msg falls through
// the switch is a gesture that silently does nothing — the one defect in this
// layer that no rendering test and no transition table can see, because there
// is nothing to look at.
//
// The package is globbed rather than two file names being written down here.
// It used to read "msg.go" and "ui.go", which made the test blind to a message
// declared in a third file — and the 300-line ceiling means this package gains
// files as it grows, so the drift the test exists to catch was exactly the
// drift that would hide from it.
func TestEveryMessageTheWindowRaisesIsHandledInUpdate(t *testing.T) {
	files := packageFiles(t)
	var declared []string
	for _, f := range files {
		declared = append(declared, declaredMessages(t, f)...)
	}
	if len(declared) == 0 {
		t.Fatal("this package declares no messages, so this test is checking nothing")
	}
	handled := handledMessages(t, files)
	for _, name := range declared {
		if !handled[name] {
			t.Errorf("%s is declared and Update has no case for it", name)
		}
	}
}

// packageFiles is this package's own source, tests excluded.
//
// Tests are excluded because a fixture message declared for one table is not
// something the window raises, and because this file would otherwise be
// reading itself.
func packageFiles(t *testing.T) []string {
	t.Helper()
	all, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	var files []string
	for _, f := range all {
		if !strings.HasSuffix(f, "_test.go") {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		t.Fatal("no source files found beside this test")
	}
	return files
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
//
// It is a type switch inside Update and nothing else: a case clause anywhere
// in the file would otherwise satisfy this test, and a value switch that
// happened to mention a message's name would answer for a message Update never
// sees.
func handledMessages(t *testing.T, files []string) map[string]bool {
	t.Helper()
	handled := map[string]bool{}
	found := false
	for _, file := range files {
		for _, decl := range parse(t, file).Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Update" {
				continue
			}
			found = true
			ast.Inspect(fn, func(n ast.Node) bool {
				sw, ok := n.(*ast.TypeSwitchStmt)
				if !ok {
					return true
				}
				for _, stmt := range sw.Body.List {
					clause, ok := stmt.(*ast.CaseClause)
					if !ok {
						continue
					}
					for _, expr := range clause.List {
						if ident, ok := expr.(*ast.Ident); ok {
							handled[ident.Name] = true
						}
					}
				}
				return true
			})
		}
	}
	if !found {
		t.Fatal("no Update in this package, so this test is checking nothing")
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
