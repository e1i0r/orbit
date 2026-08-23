package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShowPrintsTheDayAsWellAsTheClock(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}
	code, out, errOut := run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d: %s", code, errOut)
	}
	// Without the date a task that ran last week reads as this morning's.
	if day := time.Now().Format("2006-01-02"); !strings.Contains(out, day) {
		t.Errorf("show does not print the day (%s):\n%s", day, out)
	}
}

// TestShowSaysNothingRatherThanTheYearOne is the reading end of a record
// with a line in it that would not parse: record.Read hands back a
// placeholder event, and a placeholder has no time. Printed as a date it
// would say 0001-01-01, which is a date, and a wrong one.
func TestShowSaysNothingRatherThanTheYearOne(t *testing.T) {
	root, orbitHome := workspace(t)
	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}
	log := findLog(t, orbitHome)
	f, err := os.OpenFile(log, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}
	if _, err := f.WriteString("{not json\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	code, out, errOut := run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d: %s", code, errOut)
	}
	if !strings.Contains(out, "record.unreadable") {
		t.Errorf("show hides the line it could not read:\n%s", out)
	}
	if strings.Contains(out, "0001-01-01") {
		t.Errorf("show dates an event that has no time:\n%s", out)
	}
	if !strings.Contains(out, "—") {
		t.Errorf("show leaves the time column blank instead of saying it is unknown:\n%s", out)
	}
}

// findLog is the one events.jsonl under the state root. The path is derived
// from a hash of the repository's own path, and looking for the file is
// steadier than recomputing that here.
func findLog(t *testing.T, orbitHome string) string {
	t.Helper()
	var found string
	err := filepath.WalkDir(orbitHome, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "events.jsonl" {
			found = p
		}
		return nil
	})
	if err != nil || found == "" {
		t.Fatalf("no events.jsonl under %q: %v", orbitHome, err)
	}
	return found
}

// TestFirstLineKeepsTheTableATable pins what show does to text it did not
// write. The engine's output is arbitrary and the table is tab-delimited.
func TestFirstLineKeepsTheTableATable(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"plain", "plain"},
		{"first\nsecond", "first …"},
		{"before\tafter", "before after"},
		{"progress\rdone", "progress done"},
		{"", ""},
	} {
		if got := firstLine(tc.in); got != tc.want {
			t.Errorf("firstLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestShowDoesNotLetATabInTheTextAddAColumn(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "before\tafter"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}
	code, out, errOut := run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d: %s", code, errOut)
	}
	if !strings.Contains(out, "before after") {
		t.Errorf("a tab in the text opened a column of its own:\n%q", out)
	}
}
