package cli

// orbit export: the record written back out as files, what it refuses, and
// the round trip — an export restored into an empty state root comes back as
// the record it was taken from. That last one is here rather than in
// internal/export because this is the package that has both halves of it,
// the writer and the migration that reads what it wrote.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// exported runs an export into a directory that does not exist yet, which is
// the ordinary way to run one.
func exported(t *testing.T, args ...string) (dir, out string) {
	t.Helper()

	dir = filepath.Join(t.TempDir(), "backup")

	code, out, errOut := run(t, append(append([]string{"export"}, args...), dir)...)
	if code != 0 {
		t.Fatalf("export exited %d: %s", code, errOut)
	}

	return dir, out
}

func TestExportWritesTheRecordOutAsFiles(t *testing.T) {
	root, _ := workspace(t)
	writeTask(t, root)

	dir, out := exported(t)

	if !strings.Contains(out, "1 task") || !strings.Contains(out, dir) {
		t.Errorf("export said %q, which does not say what it wrote or where", out)
	}

	log := filepath.Join(dir, "tasks", "ACME-1", "events.jsonl")

	body, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("read the exported log: %v", err)
	}

	if !strings.Contains(string(body), "task.created") {
		t.Errorf("the exported log holds %q, want the event the task was written down as", body)
	}
}

// TestAnExportRestoredIsTheRecordItCameFrom. The whole promise: the tree
// this writes is the tree the migration reads, so a backup put down in an
// empty $ORBIT_HOME and asked a question answers what the original would.
func TestAnExportRestoredIsTheRecordItCameFrom(t *testing.T) {
	root, _ := workspace(t)
	dir := writeTask(t, root)

	backup, _ := exported(t)

	// The restore: the export is the state root, exactly as it stands.
	t.Setenv("ORBIT_HOME", backup)

	code, out, errOut := run(t, "list", "-repo", dir)
	if code != 0 {
		t.Fatalf("list against the restored record exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "ACME-1") {
		t.Errorf("the restored board says %q, want the task the export was taken of", out)
	}

	code, out, errOut = run(t, "show", "-repo", dir, "ACME-1")
	if code != 0 {
		t.Fatalf("show against the restored record exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "make the numbers add up") {
		t.Errorf("the restored task reads %q, want the text it was written down with", out)
	}
}

func TestExportOfOneTaskIsThatTask(t *testing.T) {
	root, _ := workspace(t)
	writeTask(t, root)

	dir, out := exported(t, "-task", "ACME-1")

	if !strings.Contains(out, "1 task") {
		t.Errorf("export said %q, want the one task it was asked for", out)
	}

	if _, err := os.Stat(filepath.Join(dir, "supervisor.jsonl")); err == nil {
		t.Error("a one-task export carried the supervisor thread along with it")
	}
}

func TestExportNeedsADirectoryToWriteInto(t *testing.T) {
	root, _ := workspace(t)
	writeTask(t, root)

	code, _, errOut := run(t, "export")
	if code == 0 {
		t.Fatal("export with nowhere to write exited 0")
	}

	if !strings.Contains(errOut, "directory") {
		t.Errorf("export said %q, which does not say what is missing", errOut)
	}
}

func TestExportRefusesADirectoryThatHoldsSomething(t *testing.T) {
	root, _ := workspace(t)
	writeTask(t, root)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "yesterday.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("put something in the way: %v", err)
	}

	code, _, errOut := run(t, "export", dir)
	if code == 0 {
		t.Fatal("export landed on top of what was already there")
	}

	if !strings.Contains(errOut, dir) {
		t.Errorf("the refusal is %q, want the directory it would not write into", errOut)
	}
}

func TestExportRefusesATaskTheRecordNeverHeardOf(t *testing.T) {
	root, _ := workspace(t)
	writeTask(t, root)

	code, _, errOut := run(t, "export", "-task", "ACME-9", filepath.Join(t.TempDir(), "backup"))
	if code == 0 {
		t.Fatal("exporting a task that is not there exited 0")
	}

	if !strings.Contains(errOut, "ACME-9") {
		t.Errorf("the refusal is %q, want the id that was asked for in it", errOut)
	}
}
