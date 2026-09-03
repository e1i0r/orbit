package view

import "testing"

func call(tool, args string, min int) Entry {
	return Entry{Kind: "phase.tool_call", At: at(min), Tool: tool, Text: args}
}

// TestTheWalkKeepsWhatWasChangedAndDropsWhatWasOnlyRead. The pruning rule
// the spec refuses to be built without: the noise is the files the agent
// opened and left alone, and the work is every file it changed.
func TestTheWalkKeepsWhatWasChangedAndDropsWhatWasOnlyRead(t *testing.T) {
	got := Walk([]Entry{
		call("Read", `{"file_path":"routes/items.go"}`, 1),
		call("Read", `{"file_path":"handlers/items.go"}`, 2),
		call("Read", `{"file_path":"store/items_repo.go"}`, 3),
		call("Edit", `{"file_path":"store/items_repo.go"}`, 4),
		call("Read", `{"file_path":"docs/notes.md"}`, 5),
		call("Write", `{"file_path":"store/items_test.go"}`, 6),
		call("Edit", `{"file_path":"store/items_repo.go"}`, 7),
		call("Bash", `{"command":"go test ./..."}`, 8),
	})

	if len(got) != 2 {
		t.Fatalf("the walk kept %d files, want the two it changed: %+v", len(got), got)
	}

	if got[0].Path != "store/items_repo.go" || got[1].Path != "store/items_test.go" {
		t.Errorf("the walk is %+v, want the changed files in the order they were first reached", got)
	}

	if got[0].Touches != 2 {
		t.Errorf("%s was changed %d times, want 2", got[0].Path, got[0].Touches)
	}

	if got[0].Read != 1 {
		t.Errorf("%s was read %d times before it was changed, want 1", got[0].Path, got[0].Read)
	}
}

// TestAWalkOfNothingIsNothing. A task that read and changed nothing has no
// path, and an empty tree under the story would look like one it lost.
func TestAWalkOfNothingIsNothing(t *testing.T) {
	if got := Walk([]Entry{call("Read", `{"file_path":"a.go"}`, 1)}); len(got) != 0 {
		t.Errorf("Walk = %+v, want nothing at all for a task that changed nothing", got)
	}
}

// TestAToolCallWithNoPathIsNotAFile. Bash carries a command, not a file, and
// a walk that read one as the other would list `go test ./...` as code that
// changed.
func TestAToolCallWithNoPathIsNotAFile(t *testing.T) {
	if got := Walk([]Entry{
		call("Bash", `{"command":"go build ./..."}`, 1),
		call("Edit", "not json at all", 2),
	}); len(got) != 0 {
		t.Errorf("Walk = %+v, want nothing out of calls that name no file", got)
	}
}
