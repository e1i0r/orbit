package ui

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/logger"
)

// written runs msgs through one window and answers with what the log holds
// afterwards. The logger is a package-level global, so no test here runs in
// parallel with another.
func written(t *testing.T, msgs ...tea.Msg) []string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "orbit.log")
	if err := logger.Init(path, ""); err != nil {
		t.Fatalf("logger.Init: %v", err)
	}

	m := Model{}
	for _, msg := range msgs {
		m = m.writeDown(msg)
	}

	if err := logger.CloseGlobal(); err != nil {
		t.Fatalf("logger.CloseGlobal: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the log: %v", err)
	}

	text := strings.TrimSpace(string(b))
	if text == "" {
		return nil
	}

	return strings.Split(text, "\n")
}

// TestEveryFailureTheWindowSeesIsWrittenDown. One line per message that can
// arrive carrying a failure, and each want is the tail of the entry — what
// precedes it is the timestamp and the level, which logger's own tests own.
func TestEveryFailureTheWindowSeesIsWrittenDown(t *testing.T) {
	boom := errors.New("boom")

	for _, c := range []struct {
		name string
		msg  tea.Msg
		want string
	}{
		{"board", boardMsg{Board: board.Board{Errs: []error{boom}}}, "[ui/board] boom"},
		{"diff", diffMsg{ID: "t1", Err: boom}, "[ui/diff] t1: boom"},
		{"record", logMsg{ID: "t1", Err: boom}, "[ui/record] t1: boom"},
		{"control", controlMsg{ID: "t1", Word: "cancel", Err: boom}, "[ui/control] cancel t1: boom"},
		{"start", startedMsg{ID: "t1", Err: boom}, "[ui/start] t1: boom"},
		{"read", readMsg{ID: "t1", Err: boom}, "[ui/read] t1: boom"},
		{"session", sessionMsg{ID: "t1", Err: boom}, "[ui/session] t1: boom"},
		{"session ended", sessionEndedMsg{ID: "t1", Err: boom}, "[ui/session] t1: boom"},
		{"cli ended", cliEndedMsg{Engine: "claude", Repo: "/r", Err: boom}, "[ui/session] claude in /r: boom"},
		{"command", commandMsg{Name: "pr", Err: boom}, "[ui/command] pr: boom"},
		{"editor", editorMsg{Err: boom}, "[ui/editor] boom"},
		{"supervisor", supervisorReplyMsg{Err: boom}, "[ui/supervisor] boom"},
	} {
		lines := written(t, c.msg)
		if len(lines) != 1 {
			t.Errorf("%s: wrote %d lines, want 1: %v", c.name, len(lines), lines)
			continue
		}

		if !strings.HasSuffix(lines[0], c.want) {
			t.Errorf("%s: wrote %q, want it to end %q", c.name, lines[0], c.want)
		}
	}
}

// TestAMessageThatWentWellSaysNothing. The log is read by somebody looking
// for what broke, and a window that writes a line every time it succeeded is
// a window whose failures cannot be found.
func TestAMessageThatWentWellSaysNothing(t *testing.T) {
	lines := written(t,
		boardMsg{Board: board.Board{}},
		diffMsg{ID: "t1"},
		logMsg{ID: "t1"},
		controlMsg{ID: "t1", Word: "cancel"},
		editorMsg{},
		outputMsg{Name: "pr", Text: "still going"},
	)
	if len(lines) != 0 {
		t.Errorf("wrote %d lines for messages that carried no failure: %v", len(lines), lines)
	}
}

// TestAClockedFailureIsWrittenOnceAndAgainWhenItComesBack.
//
// The board is read twice a second and the record once a second, so a
// repository that cannot be read would put the same sentence in the file
// sixty times a minute for as long as the window is open — which is a file
// nobody can find anything in. It is written when it changes. Clearing and
// coming back is a change, because the second failure happened.
func TestAClockedFailureIsWrittenOnceAndAgainWhenItComesBack(t *testing.T) {
	gone := errors.New("no such file or directory")
	other := errors.New("permission denied")

	lines := written(t,
		diffMsg{ID: "t1", Err: gone},
		diffMsg{ID: "t1", Err: gone},
		diffMsg{ID: "t1", Err: gone},
		diffMsg{ID: "t1", Err: other},
		diffMsg{ID: "t1"},
		diffMsg{ID: "t1", Err: gone},
	)

	// Named tails and not want, because internal/arch reads this package
	// for len() on anything it can prove is a string, and the table above
	// declares a field called want that is one.
	tails := []string{
		"[ui/diff] t1: no such file or directory",
		"[ui/diff] t1: permission denied",
		"[ui/diff] t1: no such file or directory",
	}
	if len(lines) != len(tails) {
		t.Fatalf("wrote %d lines, want %d: %v", len(lines), len(tails), lines)
	}

	for i, tail := range tails {
		if !strings.HasSuffix(lines[i], tail) {
			t.Errorf("line %d is %q, want it to end %q", i, lines[i], tail)
		}
	}
}

// TestTwoClockedSourcesDoNotSilenceEachOther. The board and the record beat
// on the same clock, so a state root that has gone away fails both at once.
// One line of memory for the pair would let each of them look like the
// other's repeat, and the file would take both of them every beat forever.
func TestTwoClockedSourcesDoNotSilenceEachOther(t *testing.T) {
	gone := errors.New("no such file or directory")

	lines := written(t,
		boardMsg{Board: board.Board{Errs: []error{gone}}},
		logMsg{ID: "t1", Err: gone},
		boardMsg{Board: board.Board{Errs: []error{gone}}},
		logMsg{ID: "t1", Err: gone},
	)
	if len(lines) != 2 {
		t.Errorf("wrote %d lines for two sources failing the same way twice, want 2: %v", len(lines), lines)
	}
}

// TestABoardsFailuresAreOneEntry. A board carries one error per repository
// it could not read, and every entry in the log is one line with one
// timestamp on it.
func TestABoardsFailuresAreOneEntry(t *testing.T) {
	lines := written(t, boardMsg{Board: board.Board{Errs: []error{
		errors.New("read /a"),
		errors.New("read /b"),
	}}})
	if len(lines) != 1 {
		t.Fatalf("wrote %d lines for one board, want 1: %v", len(lines), lines)
	}

	if !strings.HasSuffix(lines[0], "[ui/board] read /a; read /b") {
		t.Errorf("wrote %q, want both repositories on it", lines[0])
	}
}

// TestNoMessageCarriesAFailureWriteDownHasNotHeardOf reads this package the
// way a reader adding a message would not: it finds every message type with
// an Err field and fails if writeDown does not name it.
//
// The alternative is the test above, which only knows about the messages
// somebody remembered to add to it. A message added next year with an error
// on it is exactly the failure that goes unlogged, and this is the only
// thing that notices.
func TestNoMessageCarriesAFailureWriteDownHasNotHeardOf(t *testing.T) {
	files := parsePackage(t)
	body := writeDownBody(t, files)

	for _, name := range typesWithAnErr(files) {
		if !strings.Contains(body, name) {
			t.Errorf("%s carries an Err and writeDown does not mention it, so a window that sees that failure never writes it down", name)
		}
	}
}

// parsePackage is every source file of this package, tests excluded.
func parsePackage(t *testing.T) []*ast.File {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read internal/ui: %v", err)
	}

	fset := token.NewFileSet()

	var files []*ast.File

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		files = append(files, f)
	}

	return files
}

// typesWithAnErr is every named struct type in the package with a field
// called Err. Model's diffErr and logErr are deliberately not that shape:
// they are what the window still holds, not what just arrived.
func typesWithAnErr(files []*ast.File) []string {
	var names []string

	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			spec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			st, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}

			for _, field := range st.Fields.List {
				if id, ok := field.Type.(*ast.Ident); ok && id.Name == "error" && named(field, "Err") {
					names = append(names, spec.Name.Name)
				}
			}

			return true
		})
	}

	return names
}

// named reports whether one of a field's names is want.
func named(field *ast.Field, want string) bool {
	for _, n := range field.Names {
		if n.Name == want {
			return true
		}
	}

	return false
}

// writeDownBody is every identifier inside writeDown, as one string.
func writeDownBody(t *testing.T, files []*ast.File) string {
	t.Helper()

	var b strings.Builder

	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "writeDown" {
				return true
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok {
					b.WriteString(id.Name + " ")
				}

				return true
			})

			return false
		})
	}

	if b.Len() == 0 {
		t.Fatal("writeDown was not found in this package")
	}

	return b.String()
}
