package cli

import (
	"slices"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/view"
)

// TestAnAutopilotStepIsWrittenOntoTheTaskItIsAbout. The autopilot pass
// wrote nothing while it worked, and the band said "inspecting 2 tasks" for
// as long as it took. Each tool call goes onto the task its arguments name,
// by the whole id — ACME-1 is not named by a call about ACME-12 — and onto
// every task of the pass when it names none of them.
func TestAnAutopilotStepIsWrittenOntoTheTaskItIsAbout(t *testing.T) {
	root, _ := workspace(t)
	dir := writeTask(t, root)

	if code, _, errOut := run(t, "board", "new", "-repo", dir, "-id", "ACME-12", "the other one"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}

	r, err := repo.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	tasks := []view.Task{
		{ID: "ACME-1", Repo: r.Name, RepoPath: r.Path},
		{ID: "ACME-12", Repo: r.Name, RepoPath: r.Path},
	}
	step := autopilotSteps(s, tasks)

	step(engine.StreamEvent{Type: "tool_call", ToolCall: engine.StreamToolCall{
		Name: "orbit_inspect_task", Args: `{"id":"ACME-12"}`,
	}})

	if slices.Contains(recorded(t, dir, "ACME-1"), "deliver.step") {
		t.Error("a step about ACME-12 was written onto ACME-1")
	}

	if !slices.Contains(recorded(t, dir, "ACME-12"), "deliver.step") {
		t.Error("a step about ACME-12 was not written onto it")
	}

	step(engine.StreamEvent{Type: "tool_call", ToolCall: engine.StreamToolCall{
		Name: "orbit_board", Args: `{}`,
	}})

	if !slices.Contains(recorded(t, dir, "ACME-1"), "deliver.step") {
		t.Error("a step that names no task was not written onto every task of the pass")
	}
}
