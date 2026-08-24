package words

// scan_test.go finds every call to T or P anywhere in the module, so
// words_test.go can hold them to account. Split out of words_test.go
// because walking the module and checking what was found are two jobs.

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

// root walks up from the test's working directory to find go.mod. This
// duplicates internal/arch's identical helper rather than importing it:
// arch.layers lists internal/words with nothing it may import of Orbit's
// own, and a rule that exempted its own tests would not be much of a rule.
func root(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the working directory")
		}
		dir = parent
	}
}

// callSite is one call to T or P found somewhere in the module.
type callSite struct {
	file string
	line int

	method string // "T" or "P"
	key    string
	keyOK  bool // the key argument was a string literal

	// english is T's fallback, or P's "one" form; other is P's "other"
	// form. literal reports whether both were string literals — when they
	// are not, the call cannot be checked against es.json statically, and
	// only the key is held to account.
	english string
	other   string
	literal bool
}

// collectCallSites walks every .go file in the module except
// internal/words and returns every call whose method is named T or P.
//
// internal/words is excluded because it is where T and P are defined, not
// where a screen draws a string through them — this package has no screen.
// A test inside this package that calls p.T is exercising the mechanism on
// its own terms, not shipping the untranslated string that checks 2
// through 6 exist to catch.
func collectCallSites(t *testing.T) []callSite {
	t.Helper()
	modRoot := root(t)
	wordsDir := filepath.Join(modRoot, "internal", "words")

	var sites []callSite
	err := filepath.WalkDir(modRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if outsideTheModule(modRoot, path, d) || path == wordsDir {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		sites = append(sites, scanFile(t, modRoot, path)...)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return sites
}

// scanFile parses one file and returns every T or P call it contains.
func scanFile(t *testing.T, modRoot, path string) []callSite {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	rel, err := filepath.Rel(modRoot, path)
	if err != nil {
		t.Fatalf("rel %s: %v", path, err)
	}
	rel = filepath.ToSlash(rel)

	var sites []callSite
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "T" && sel.Sel.Name != "P") {
			return true
		}
		site := callSite{
			file:   rel,
			line:   fset.Position(call.Pos()).Line,
			method: sel.Sel.Name,
		}
		if len(call.Args) > 0 {
			site.key, site.keyOK = stringLiteral(call.Args[0])
		}
		switch site.method {
		case "T":
			if len(call.Args) > 1 {
				site.english, site.literal = stringLiteral(call.Args[1])
			}
		case "P":
			if len(call.Args) > 3 {
				eng, engOK := stringLiteral(call.Args[2])
				oth, othOK := stringLiteral(call.Args[3])
				site.english, site.other, site.literal = eng, oth, engOK && othOK
			}
		}
		sites = append(sites, site)
		return true
	})
	return sites
}

// stringLiteral reports the value of e when e is a string literal.
func stringLiteral(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return v, true
}

// outsideTheModule is a directory `go build ./...` would not compile: vendor,
// and anything whose name begins with "." or "_". These walkers have to agree
// with the toolchain about where the module ends, because a directory the
// toolchain ignores can still hold a complete second copy of it — an agent's
// git worktree under .claude, an editor's index, a nested checkout — and every
// file in that copy would otherwise be read as if it were part of this one.
//
// The walk root is never outside the module, whatever it is called: a checkout
// that happens to live in a dotted directory is still the module.
func outsideTheModule(root, path string, d os.DirEntry) bool {
	if path == root || !d.IsDir() {
		return false
	}
	n := d.Name()
	return n == "vendor" || strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_")
}
