package cli

// The browser's ports, filled the way `orbit web` fills them.
//
// webPorts is everything the browser reads through and asks through, over
// one store and the board behind it. These drive the ports directly — no
// server, no port — with a state root and a reader of the test's own.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// portsOf is webPorts over a state root of the test's own, read against
// a workspace with one repository in it.
func portsOf(t *testing.T) (*store.Store, *board.Reader, *words.Printer) {
	t.Helper()

	t.Setenv("ORBIT_HOME", t.TempDir())

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = s.Close() })

	return s, board.NewReader(s, t.TempDir()), words.For("en")
}

// TestTheBrowserReadsWhatOrbitKnows. Facts, the thread, the engines and
// the settled default: the same answers the cockpit draws, read through
// the ports.
func TestTheBrowserReadsWhatOrbitKnows(t *testing.T) {
	s, r, p := portsOf(t)
	ports := webPorts(r, s, newEngines(), t.TempDir(), p)

	if facts := ports.Knows.Facts(); facts == nil {
		t.Error("facts reads nothing at all, not even an empty list")
	}

	if _, _, err := ports.Talks.Thread(); err != nil {
		t.Fatalf("thread: %v", err)
	}

	if got := ports.Roster.Engines(); len(got) == 0 {
		t.Error("engines reads nothing at all")
	}

	if settled := ports.Roster.Settled(); settled == "" {
		t.Error("no engine is settled and none is first either")
	}
}

// TestSourceAndActionNameEveryKind. Where a fact came from and what it
// does, in a word a reader reads.
func TestSourceAndActionNameEveryKind(t *testing.T) {
	for source, want := range map[knowledge.Source]string{
		knowledge.FromCode:       "read off the code",
		knowledge.Human:          "said by a person",
		knowledge.FromRecord:     "learned from a run",
		knowledge.FromProduction: "from an incident",
	} {
		if got := sourceName(source); got != want {
			t.Errorf("source %v reads %q, want %q", source, got, want)
		}
	}

	if got := sourceName(knowledge.Source(-1)); got != "unsourced" {
		t.Errorf("no source reads %q, want it to say so", got)
	}

	if got := actionName(knowledge.Stops); got != "stops" {
		t.Errorf("a stopping fact reads %q", got)
	}

	if got := actionName(knowledge.Warns); got != "warns" {
		t.Errorf("a warning fact reads %q", got)
	}
}

// TestOneDirectoryTakesNoneOrOne. Watching is of one directory: none
// names the one the reader is in, and two name nothing coherent.
func TestOneDirectoryTakesNoneOrOne(t *testing.T) {
	ctx := Context{Words: words.For("en")}

	if dir, err := oneDirectory(ctx, nil); err != nil || dir != "." {
		t.Errorf("no directory reads %q, %v", dir, err)
	}

	if dir, err := oneDirectory(ctx, []string{"/src"}); err != nil || dir != "/src" {
		t.Errorf("one directory reads %q, %v", dir, err)
	}

	if _, err := oneDirectory(ctx, []string{"a", "b"}); err == nil {
		t.Error("two directories were accepted")
	}

	if _, err := oneDirectory(ctx, []string{"a", "b", "c"}); err == nil {
		t.Error("three directories were accepted")
	}
}
