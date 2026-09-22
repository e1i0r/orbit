package cli

// Every command the window asks for is a command that exists.
//
// The window does not run anything itself: it names a command and the port
// in window.go looks that name up. A name with no command behind it fails
// at the one moment nobody is looking — the form closes, the band carries a
// sentence the reader has already scrolled past, and nothing is written.
//
// That is not a hypothetical. `writeTask` asked for "new" for as long as
// writing a task lived under `board`, so every task written from the form
// came back "no such command: new" and no task was ever written. The form
// was fine. The click was fine. The name was wrong, and nothing in the
// build could say so.
//
// So the names are read out of the window's own source and looked up here,
// where the table is.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestEveryCommandTheWindowNamesExists.
func TestEveryCommandTheWindowNamesExists(t *testing.T) {
	named := windowNames(t)
	if len(named) == 0 {
		t.Fatal("no command names were found in the window's source, so this test proves nothing")
	}

	for name, where := range named {
		c, found := lookup(name)
		if !found {
			t.Errorf("%s asks for the command %q and there is none; a child verb is run as its "+
				"parent with the child as the first argument, the way `pr merge` is", where, name)

			continue
		}

		// A command the window may name and may not run is the same dead
		// end one step further along.
		if c.InWindow == WindowRefuses {
			t.Errorf("%s asks for %q, which the window refuses to run", where, name)
		}
	}
}

// windowNames is every `Command{Name: "x"}` written in internal/ui, by name
// and where it was written.
//
// Only the literals: a name held in a variable is one this cannot check,
// and the two that are — the delivery verbs in note.go — are covered by
// their own tests where the table of them is.
func windowNames(t *testing.T) map[string]string {
	t.Helper()

	found := map[string]string{}

	dir := filepath.Join(root(t), "internal", "ui")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the window's source: %v", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}

		path := filepath.Join(dir, e.Name())

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			if id, ok := lit.Type.(*ast.Ident); !ok || id.Name != "Command" {
				return true
			}

			for _, field := range lit.Elts {
				kv, ok := field.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				key, ok := kv.Key.(*ast.Ident)
				if !ok || key.Name != "Name" {
					continue
				}

				if said, ok := kv.Value.(*ast.BasicLit); ok && said.Kind == token.STRING {
					if name, err := strconv.Unquote(said.Value); err == nil {
						found[name] = e.Name()
					}
				}
			}

			return true
		})
	}

	return found
}

// root is the module's own directory, found by walking up from here.
func root(t *testing.T) string {
	t.Helper()

	at, err := os.Getwd()
	if err != nil {
		t.Fatalf("where am I: %v", err)
	}

	for range 6 {
		if _, err := os.Stat(filepath.Join(at, "go.mod")); err == nil {
			return at
		}

		at = filepath.Dir(at)
	}

	t.Fatal("no go.mod above this test")

	return ""
}

// TestTheWindowMayRunAChildOfAScreenCommand.
//
// `orbit board` is a screen the window already is, so the parent is
// WindowOpens; `board new` writes a task down and is a verb like any
// other. The policy was read off the parent, so the port refused the
// child — and the form's save, which is exactly that child, was refused
// from the day writing a task moved under board.
func TestTheWindowMayRunAChildOfAScreenCommand(t *testing.T) {
	said := &strings.Builder{}

	run := doPort(inSpanish{})

	// The child of a screen command: refused before, and now only refused
	// by whatever the verb itself says.
	err := run("board", []string{"new", "-id", ""}, said)
	if err != nil && strings.Contains(err.Error(), "pantalla") {
		t.Errorf("the window refused `board new` because `board` draws a screen: %v", err)
	}

	// And the parent on its own is still a screen.
	if err := run("board", nil, said); err == nil || !strings.Contains(err.Error(), "pantalla") {
		t.Errorf("`board` on its own answered %v, want the screen it opens", err)
	}

	// A word that is not one of its children is not a child.
	if err := run("board", []string{"nonsense"}, said); err == nil ||
		!strings.Contains(err.Error(), "pantalla") {
		t.Errorf("`board nonsense` answered %v, want the parent's own answer", err)
	}
}

// inSpanish is the language port these tests read refusals in.
type inSpanish struct{}

func (inSpanish) Language() string { return "es" }
