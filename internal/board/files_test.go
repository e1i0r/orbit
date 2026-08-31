package board

// The two questions the artifacts tab asks: what is in a task's own
// directory, and what is inside one of those files. Both are read against a
// real state root rather than through the window's fake, because a port
// asserted only against a fake is a port nobody has run.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

// taskDir is where a task's own files are kept, and fails the test if the
// store cannot say.
func taskDir(t *testing.T, s *store.Store, repoPath, id string) string {
	t.Helper()

	dir, err := s.TaskDir(id)
	if err != nil {
		t.Fatalf("task directory of %s: %v", id, err)
	}

	return dir
}

// writeIn puts one file into a task's directory.
func writeIn(t *testing.T, dir, name, body string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// TestTheListingIsWhatTheDirectoryHolds. The tab draws a name and a size per
// row and nothing else, so those two are what this has to be right about.
func TestTheListingIsWhatTheDirectoryHolds(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"))

	dir := taskDir(t, s, repoPath, "ACME-1")
	writeIn(t, dir, "task.md", "Retry the webhook on 5xx\n")
	writeIn(t, dir, "control", "pause")

	// A directory is not a file of the listing: the tab opens what it lists,
	// and a row that cannot be opened is a row that lies about its arrow.
	if err := os.Mkdir(filepath.Join(dir, "sessions"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	files, err := NewReader(s, work).Files(repoPath, "ACME-1")
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	var names []string
	for _, f := range files {
		names = append(names, f.Name)
	}

	// In name order, which is the order the pane draws: a listing that came
	// back arranged differently between two polls of the same second would
	// move the row a reader is reaching for.
	// The marker naming the repositories the task is worked in is a file of
	// the task like any other, and the tab shows what is there.
	if got := strings.Join(names, " "); got != "control events.jsonl repos task.md" {
		t.Fatalf("the listing is %q, want the four files in name order", got)
	}

	for _, f := range files {
		if want := sizeOf(t, filepath.Join(dir, f.Name)); f.Size != want {
			t.Errorf("%s is listed at %d bytes, want the %d on disk", f.Name, f.Size, want)
		}
	}
}

// TestATaskThatHasNotRunListsNothing. A task written down and not yet
// started has no directory at all, and that is an empty listing rather than
// a failure — it is the same answer a run that left nothing gets, and it is
// the true one.
func TestATaskThatHasNotRunListsNothing(t *testing.T) {
	s, work, repoPath := oneRepo(t)

	files, err := NewReader(s, work).Files(repoPath, "ACME-404")
	if err != nil {
		t.Fatalf("Files of a task with no directory: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("a task that has not run lists %d files, want none", len(files))
	}
}

// TestAFileIsReadWholeAndSaysSo. Whole is what the pane decides on: a file
// read to its end shows nothing under it, and one that was cut says the rest
// was not read.
func TestAFileIsReadWholeAndSaysSo(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"))
	writeIn(t, taskDir(t, s, repoPath, "ACME-1"), "control", "pause")

	got, err := NewReader(s, work).FileText(repoPath, "ACME-1", "control")
	if err != nil {
		t.Fatalf("FileText: %v", err)
	}

	if got.Text != "pause" || !got.Whole {
		t.Errorf("FileText = %q whole=%v, want the file it holds, read to the end", got.Text, got.Whole)
	}
}

// TestALongFileIsCutAtTheCap. A record that grew all day is not a thing a
// window is made to hold in memory by opening one row of a listing.
func TestALongFileIsCutAtTheCap(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"))

	dir := taskDir(t, s, repoPath, "ACME-1")
	writeIn(t, dir, "big", strings.Repeat("x", fileTextCap+500))

	r := NewReader(s, work)

	got, err := r.FileText(repoPath, "ACME-1", "big")
	if err != nil {
		t.Fatalf("FileText: %v", err)
	}

	if len(got.Text) != fileTextCap || got.Whole {
		t.Errorf("a long file read %d bytes whole=%v, want the cap and a file known to be cut", len(got.Text), got.Whole)
	}

	// Exactly the cap is the boundary the extra byte exists for: read to the
	// cap and no further, a file this size is whole and a guess would call
	// it cut.
	writeIn(t, dir, "exact", strings.Repeat("x", fileTextCap))

	got, err = r.FileText(repoPath, "ACME-1", "exact")
	if err != nil {
		t.Fatalf("FileText: %v", err)
	}

	if len(got.Text) != fileTextCap || !got.Whole {
		t.Errorf("a file exactly the cap read %d bytes whole=%v, want the cap, whole", len(got.Text), got.Whole)
	}
}

// TestAFileListedAndGoneReadsEmpty. The listing and the read are two asks,
// and the run goes on writing between them.
func TestAFileListedAndGoneReadsEmpty(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"))

	got, err := NewReader(s, work).FileText(repoPath, "ACME-1", "control")
	if err != nil {
		t.Fatalf("FileText of a file that is not there: %v", err)
	}

	if got.Text != "" || !got.Whole {
		t.Errorf("FileText = %q whole=%v, want nothing, read to the end", got.Text, got.Whole)
	}
}

// TestANameThatIsAPathIsRefused. The name comes from the listing this same
// port produced, so a name carrying a separator is not a file of the task at
// all — it is a way to read a directory the window is not entitled to.
func TestANameThatIsAPathIsRefused(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"))

	outside := filepath.Join(work, "secret")
	if err := os.WriteFile(outside, []byte("not yours"), 0o600); err != nil {
		t.Fatalf("write the file outside: %v", err)
	}

	r := NewReader(s, work)

	for _, name := range []string{"", ".", "..", "/", "../secret", "sessions/one.json", outside} {
		got, err := r.FileText(repoPath, "ACME-1", name)
		if err == nil {
			t.Errorf("FileText(%q) answered %q, want a refusal", name, got.Text)
			continue
		}

		// Refused for being a path, and not by whatever the machine said
		// about opening one. A dot and two dots name a directory, and a
		// directory is a thing os.Open opens and io.ReadFull then fails on
		// — so an assertion that only asked for an error would pass with
		// the guard taken out.
		if !strings.Contains(err.Error(), "not a file of its own directory") {
			t.Errorf("FileText(%q) failed with %q, want the refusal and not a failure to read", name, err)
		}
	}
}
