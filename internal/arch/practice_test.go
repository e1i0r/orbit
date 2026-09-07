package arch

// The house style, in the form that fails a build rather than a review.
//
// Most of what CONTRIBUTING.md asks for is judgement and stays judgement.
// What is here is the part a machine can hold: interfaces stay small, no
// package is a drawer for whatever had nowhere else to go, and a getter is
// named after what it answers rather than after the act of asking.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// methodCeiling is how many methods one interface may declare.
//
// Six, because an interface is a promise the caller has to keep and every
// method is one more thing a fake in a test has to answer. What an interface
// is for is naming the two or three things a function actually needs; the
// day one of them wants a seventh, what it wants is two interfaces.
const methodCeiling = 6

// wideInterfaces are the ones that are over the ceiling on purpose, with the
// reason each is allowed to be.
//
// They are the ports: the whole of what the window may ask the world for,
// declared in one place so that internal/cli has one thing to satisfy. They
// shrink as screens leave internal/ui — a screen that is its own package
// declares the two or three methods it needs, and the port loses them.
var wideInterfaces = map[string]string{
	"internal/ui/portread.go:Settings":        "the settings file's whole surface, until every screen that reads it declares its own half",
	"internal/ui/portread.go:Reader":          "the record as the window may read it: one door per pane, and a pane is not a method a caller picks",
	"internal/engine/engine.go:Engine":        "an engine driver: what every engine must answer for a run to be possible at all",
	"internal/ui/settings/settings.go:Store":  "two interfaces composed, which is the shape this rule asks for",
	"internal/ui/settings/settings.go:reader": "the settings file has seven settings and the table shows all of them: this is the file, not a choice of what to ask it",
	"internal/ui/settings/settings.go:writer": "the other half of the same file, for the same reason",
}

// TestInterfacesStaySmall.
func TestInterfacesStaySmall(t *testing.T) {
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		for name, methods := range interfacesIn(t, path) {
			if methods <= methodCeiling {
				continue
			}

			rel := relative(t, path)
			if why := wideInterfaces[rel+":"+name]; why != "" {
				continue
			}

			t.Errorf("%s declares %s with %d methods, over the ceiling of %d — split it, or write down why it is one thing in wideInterfaces",
				rel, name, methods, methodCeiling)
		}
	}
}

// TestNoPackageIsADrawer. util, common, helpers and misc are where code goes
// when nobody has decided what it is, and everything in them is unreachable
// by anybody looking for it by name.
func TestNoPackageIsADrawer(t *testing.T) {
	for _, path := range goFiles(t) {
		dir := filepath.Base(filepath.Dir(path))
		switch dir {
		case "util", "utils", "common", "helpers", "misc", "shared", "lib":
			t.Errorf("%s is in a package called %s, which is a drawer rather than a subject", relative(t, path), dir)
		}
	}
}

// TestNoGetterSaysGet. A getter is named after what it answers: Name, not
// GetName. The word get carries nothing the call did not already say.
func TestNoGetterSaysGet(t *testing.T) {
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, "Get") || len(fn.Name.Name) < 4 {
				continue
			}

			if next := fn.Name.Name[3]; next >= 'A' && next <= 'Z' {
				t.Errorf("%s declares %s — name it after what it answers, not after the asking", relative(t, path), fn.Name.Name)
			}
		}
	}
}

// interfacesIn is every interface a file declares, by name, with how many
// methods each one asks for. An embedded interface counts as one: composing
// two small ones is the shape this rule is asking for, not a way around it.
func interfacesIn(t *testing.T, path string) map[string]int {
	t.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	out := map[string]int{}

	ast.Inspect(f, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if iface, ok := spec.Type.(*ast.InterfaceType); ok {
			out[spec.Name.Name] = len(iface.Methods.List)
		}

		return true
	})

	return out
}

// relative is a path as this repository names it, so a failure reads like
// the file a reader would open.
func relative(t *testing.T, path string) string {
	t.Helper()

	rel, err := filepath.Rel(root(t), path)
	if err != nil {
		return path
	}

	return rel
}
