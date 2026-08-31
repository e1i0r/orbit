package task

// task.go's own error returns: the ones Create, Load, chosenFlow, List,
// Events and emit take when the store refuses a path or the filesystem
// refuses a write, none of which the hand-written task_test.go or the other
// coverage files provoke.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestCreateChosenFlowErrorPropagates covers Create's first error return: an
// empty flow name sends it to the settings file, and that file cannot be
// read.
func TestCreateChosenFlowErrorPropagates(t *testing.T) {
	s, r := fixture(t)

	settingsPath := filepath.Join(s.Root(), "settings.json")
	if err := os.Mkdir(settingsPath, 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	if _, err := Create(s, r, "CREATE-FLOW-ERR-1", "text", ""); err == nil {
		t.Error("Create should have failed when settings.json cannot be read")
	}
}

// TestCreateTaskFilePathErrorPropagates covers Create's second error return:
// an id the store refuses.
func TestCreateTaskFilePathErrorPropagates(t *testing.T) {
	s, r := fixture(t)
	if _, err := Create(s, r, "has/slash", "text", "quick"); err == nil {
		t.Error("Create with a slash in the id should have failed")
	}
}

// TestCreateTaskDirErrorPropagates covers Create's CreateTaskDir error
// return: the repository's directory exists as a plain file, so MkdirAll
// cannot make the tree Create needs underneath it.
func TestCreateTaskDirErrorPropagates(t *testing.T) {
	s, r := fixture(t)

	repoDir, err := s.RepoDir(r.Path)
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(repoDir), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(repoDir, []byte("blocking"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Create(s, r, "CREATE-DIR-ERR-1", "text", "quick"); err == nil {
		t.Error("Create should have failed when the repo directory is blocked by a file")
	}
}

// TestCreateWriteFileErrorPropagates covers Create's task.md write error: the
// task directory exists but has lost its write bit.
func TestCreateWriteFileErrorPropagates(t *testing.T) {
	s, r := fixture(t)
	id := "CREATE-WRITE-ERR-1"

	dir, err := s.CreateTaskDir(r.Path, id)
	if err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:errcheck

	if _, err := Create(s, r, id, "text", "quick"); err == nil {
		t.Error("Create should have failed writing task.md into a read-only directory")
	}
}

// TestLoadTaskFilePathErrorPropagates covers Load's first error return.
func TestLoadTaskFilePathErrorPropagates(t *testing.T) {
	s, r := fixture(t)
	if _, err := Load(s, r, "has/slash"); err == nil {
		t.Error("Load with a slash in the id should have failed")
	}
}

// TestLoadReadFileErrorPropagates covers Load's second, non-ErrNotExist,
// ReadFile error return: task.md exists as a directory rather than a file.
func TestLoadReadFileErrorPropagates(t *testing.T) {
	s, r := fixture(t)
	id := "LOAD-READ-ERR-1"

	path, err := s.TaskFilePath(id)
	if err != nil {
		t.Fatalf("TaskFilePath: %v", err)
	}

	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if _, err := Load(s, r, id); err == nil {
		t.Error("Load over a directory should have failed")
	}
}

// TestChosenFlowSettingsErrorPropagates calls chosenFlow directly so the
// settings error is asserted without Create's own wrapping in the way.
func TestChosenFlowSettingsErrorPropagates(t *testing.T) {
	s, _ := fixture(t)

	settingsPath := filepath.Join(s.Root(), "settings.json")
	if err := os.Mkdir(settingsPath, 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	if _, err := chosenFlow(s, ""); err == nil {
		t.Error("chosenFlow should have failed when settings.json cannot be read")
	}
}

// TestWrittenFlowSkipsEventsThatAreNotTaskCreated exercises the loop's
// continue branch: a log with events after task.created must not let a
// later, unrelated event overwrite the flow name task.created recorded.
func TestWrittenFlowSkipsEventsThatAreNotTaskCreated(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "WRITTEN-FLOW-1", "text", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := emit(s, tk, record.Event{Kind: record.TaskStarted, Data: map[string]string{"flow": "task"}}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	if got := writtenFlow(s, tk); got != "quick" {
		t.Errorf("writtenFlow = %q, want quick — a later event overwrote what task.created said", got)
	}
}

// TestWrittenFlowEventsErrorAnswersEmpty covers writtenFlow's own error
// return: a log that cannot be read answers "" rather than failing Load.
func TestWrittenFlowEventsErrorAnswersEmpty(t *testing.T) {
	s, r := fixture(t)

	bad := Task{ID: "has/slash", Repo: r}
	if got := writtenFlow(s, bad); got != "" {
		t.Errorf("writtenFlow on a bad id = %q, want empty", got)
	}
}

// TestListTasksDirReadDirErrorPropagates covers List's ReadDir error return:
// the tasks directory exists as a plain file.
func TestListTasksDirReadDirErrorPropagates(t *testing.T) {
	s, r := fixture(t)

	tasksDir := s.TasksDir()

	if err := os.MkdirAll(filepath.Dir(tasksDir), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(tasksDir, []byte("blocking"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := List(s, r); err == nil {
		t.Error("List should have failed when the tasks directory is blocked by a file")
	}
}

// TestEventsErrorPathPropagates covers Events' own EventsPath error return.
func TestEventsErrorPathPropagates(t *testing.T) {
	s, r := fixture(t)

	bad := Task{ID: "has/slash", Repo: r}
	if _, err := Events(s, bad); err == nil {
		t.Error("Events with a slash in the id should have failed")
	}
}

// TestEmitErrorPathPropagates covers emit's own EventsPath error return.
func TestEmitErrorPathPropagates(t *testing.T) {
	s, r := fixture(t)

	bad := Task{ID: "has/slash", Repo: r}
	if err := emit(s, bad, record.Event{Kind: record.TaskCreated}); err == nil {
		t.Error("emit with a slash in the id should have failed")
	}
}
