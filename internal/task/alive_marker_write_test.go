package task

// How the marker is written, which is a different question from what it says.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

// markerFixture is a task whose marker names pid, and the path to it.
func markerFixture(t *testing.T, id string, pid int) (*store.Store, Task, string) {
	t.Helper()
	s, r := fixture(t)

	tk, err := Create(s, r, id, "marker test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := mark(s, tk, pid); err != nil {
		t.Fatalf("mark: %v", err)
	}

	path, err := s.RunPath(tk.ID)
	if err != nil {
		t.Fatalf("RunPath: %v", err)
	}

	return s, tk, path
}

// TestTheMarkerIsReplacedRatherThanRewritten, with the hard link making it a
// test rather than a race to lose.
//
// os.WriteFile truncates first and writes second. Everything between those
// two is a marker with no pid line,
// and readMarker calls that damage rather than "not running" — deliberately,
// because a claim that cannot be read is a claim that cannot be ruled out.
// The board asks Alive about every task twice a second, so that instant is
// sampled constantly: it surfaces as an error under a run that is starting
// normally and, worse, as hold refusing to start a run at all.
//
// Racing two goroutines to catch a window microseconds wide is a test that
// passes when it feels like it. The link is the same fact stated so it holds
// still: it names the file that was there before, so whatever that inode
// holds afterwards is what a reader who opened the marker first would see. A
// rewrite in place would show them the new pid — or none. A rename shows them
// the whole old marker, which is the property being bought.
func TestTheMarkerIsReplacedRatherThanRewritten(t *testing.T) {
	s, tk, path := markerFixture(t, "MARKER-1", 4711)

	aside := path + ".before"
	if err := os.Link(path, aside); err != nil {
		t.Fatalf("hard link: %v", err)
	}

	if _, err := mark(s, tk, 4712); err != nil {
		t.Fatalf("the second mark: %v", err)
	}

	before, err := os.ReadFile(aside)
	if err != nil {
		t.Fatalf("read the file that was there first: %v", err)
	}

	pid, err := parsePid(string(before))
	if err != nil || pid != 4711 {
		t.Errorf("the marker a reader already had reads %q (%v), want the whole pid 4711: it was rewritten in place", before, err)
	}

	now, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the marker: %v", err)
	}

	if pid, err := parsePid(string(now)); err != nil || pid != 4712 {
		t.Errorf("the marker now reads %q (%v), want pid 4712", now, err)
	}
}

// TestTheRenameLeavesNothingBesideTheMarker. The temporary the rename needs
// is made in the task's own directory, because a rename is only atomic within
// one filesystem — so a failure to clean it up litters the directory Orbit
// tells people to cat, one file per run.
func TestTheRenameLeavesNothingBesideTheMarker(t *testing.T) {
	_, _, path := markerFixture(t, "MARKER-2", 4711)

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("read the task directory: %v", err)
	}

	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("%q was left behind by the write", e.Name())
		}
	}
}
