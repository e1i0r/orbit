package mcp

// The verbs a tool call reaches, asked the way a tool call asks them.
//
// askVerb is every generated tool's whole doing: the arguments arrive as
// JSON, the verb runs against this session's world, and the answer goes
// back the way it came. What a hand-written tool would do differently is
// the tool's, and is tested where the tool is.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/task"
)

// asked runs one generated verb for a task of the fixture, failing the
// test on a refusal. Refusing is covered where the verb is.
func asked(t *testing.T, sn Session, name, id string, args map[string]any) string {
	t.Helper()

	if args == nil {
		args = map[string]any{}
	}

	args["task_id"] = id

	got := sn.askVerb(name, args)
	if got.IsError {
		t.Fatalf("%s: %s", name, textOf(got))
	}

	return textOf(got)
}

// unanswered runs one generated verb that is meant to refuse, and fails
// the test when it does not.
func unanswered(t *testing.T, sn Session, name, id string, args map[string]any) string {
	t.Helper()

	if args == nil {
		args = map[string]any{}
	}

	args["task_id"] = id

	got := sn.askVerb(name, args)
	if !got.IsError {
		t.Fatalf("%s was accepted, want a refusal", name)
	}

	return textOf(got)
}

// textOf is the whole text of an answer, for assertions that read what
// a verb said rather than the shape around it. The package's own said
// clips for the log; this one does not clip, so a match cannot hide past
// the cut.
func textOf(got CallToolResult) string {
	var b strings.Builder

	for _, c := range got.Content {
		b.WriteString(c.Text)
	}

	return b.String()
}

// TestGeneratedVerbsReachTheTask. Saying, learning, reading, marking,
// deleting, permitting, marking critical and joining: each through
// askVerb, each about the same task.
func TestGeneratedVerbsReachTheTask(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "pay the thing"})

	asked(t, sn, "say", "ACME-1", map[string]any{"text": "never force-push"})
	asked(t, sn, "learn", "ACME-1", map[string]any{"text": "amounts are cents"})
	asked(t, sn, "history", "ACME-1", nil)
	asked(t, sn, "read", "ACME-1", nil)
	asked(t, sn, "critical", "ACME-1", map[string]any{"on": true})
	asked(t, sn, "critical", "ACME-1", map[string]any{"on": false})

	unanswered(t, sn, "permit", "ACME-1", nil)
	unanswered(t, sn, "approve", "ACME-1", nil)
	unanswered(t, sn, "join", "ACME-1", map[string]any{"name": "nowhere"})
	unanswered(t, sn, "take", "ACME-1", nil)

	asked(t, sn, "export", "ACME-1", map[string]any{"into": t.TempDir() + "/out"})
	asked(t, sn, "delete", "ACME-1", nil)
	unanswered(t, sn, "read", "ACME-1", nil)
}

// TestReconcilingThroughAToolCall. Nothing running and nothing dead: the
// sweep says everything is accounted for.
func TestReconcilingThroughAToolCall(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "ACME-2", record.Event{Kind: record.TaskCreated, Text: "leave no process behind"})

	if out := asked(t, sn, "reconcile", "ACME-2", nil); out == "" {
		t.Error("reconcile said nothing at all")
	}
}

// TestTheWorldBehindAToolCall. The ports every generated verb reaches
// through: the board already folded, what Orbit was told, one task's
// record, and the two refusals — delivering and taking, which are a
// person's and need a terminal.
func TestTheWorldBehindAToolCall(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "ACME-4", record.Event{Kind: record.TaskCreated, Text: "read me back"})

	sb, err := sn.readBoard()
	if err != nil {
		t.Fatalf("read the board: %v", err)
	}

	defer sb.close()

	w := sn.world(sb)

	if _, err := w.Board(); err != nil {
		t.Fatalf("board: %v", err)
	}

	if _, err := w.Facts(); err != nil {
		t.Fatalf("facts: %v", err)
	}

	if _, err := w.Log(r.Path, "ACME-4"); err != nil {
		t.Fatalf("log: %v", err)
	}

	if _, err := w.Unread(""); err != nil {
		t.Fatalf("unread: %v", err)
	}

	if err := w.Looked(); err != nil {
		t.Fatalf("looked: %v", err)
	}

	if _, err := w.Deliver(sn.context(), mustTask(t, sb, r, "ACME-4"), "pr"); err == nil {
		t.Error("delivering from a tool call was accepted")
	}

	if _, err := w.Take("ACME-4", r.Path); err == nil {
		t.Error("taking a keyboard a tool call has none of was accepted")
	}

	if paths := repoPathsOf(sb.board.RepoList); len(paths) != len(sb.board.RepoList) {
		t.Errorf("repository paths came back as %v", paths)
	}
}

// mustTask loads the task the fixture wrote, failing the test when the
// record does not have it.
func mustTask(t *testing.T, sb *storeAndBoard, r repo.Repo, id string) task.Task {
	t.Helper()

	found, err := task.Load(sb.store, r, id)
	if err != nil {
		t.Fatalf("load task %s: %v", id, err)
	}

	return found
}

func TestJoiningTheSecondCheckout(t *testing.T) {
	s, sn, r := oneRepo(t)
	second := gitRepo(t, sn.Root, "ledger")

	addTask(t, s, r, "ACME-3", record.Event{Kind: record.TaskCreated, Text: "reach into both"})

	out := asked(t, sn, "join", "ACME-3", map[string]any{"name": second.Name})
	if !strings.Contains(out, second.Name) {
		t.Errorf("join answered %q, want the directory", out)
	}
}

// TestFlowsAreWrittenReadAndTakenAway. Saving, reading, listing and
// deleting through the tools, with a refusal before the second asking.
func TestFlowsAreWrittenReadAndTakenAway(t *testing.T) {
	_, sn, _ := oneRepo(t)

	doc := map[string]any{
		"name":        "shaped",
		"description": "a flow the test wrote",
		"phases": []any{map[string]any{
			"name": "work", "engine": "claude", "prompt": "work",
		}},
	}

	if got := sn.Call("orbit_save_flow", doc); got.IsError {
		t.Fatalf("save flow: %s", textOf(got))
	}

	if got := sn.Call("orbit_get_flow", map[string]any{"name": "shaped"}); got.IsError {
		t.Fatalf("get flow: %s", textOf(got))
	}

	if got := sn.Call("orbit_list_flows", nil); got.IsError {
		t.Fatalf("list flows: %s", textOf(got))
	}

	if got := sn.Call("orbit_save_flow", map[string]any{"name": ""}); !got.IsError {
		t.Error("saving a flow with no name was accepted")
	}

	if got := sn.Call("orbit_save_flow", map[string]any{"name": "shapeless"}); !got.IsError {
		t.Error("saving a flow with no phases was accepted")
	}

	if got := sn.Call("orbit_delete_flow", map[string]any{"name": "shaped"}); got.IsError {
		t.Fatalf("delete flow: %s", textOf(got))
	}

	if got := sn.Call("orbit_get_flow", map[string]any{"name": "shaped"}); !got.IsError {
		t.Error("reading a deleted flow was accepted")
	}
}

// TestReposAreAddedListedAndForgotten. The workspace through the tools:
// listed, inspected, added by path, and forgotten only when asked twice.
func TestReposAreAddedListedAndForgotten(t *testing.T) {
	_, sn, r := oneRepo(t)

	if got := sn.Call("orbit_list_repos", nil); got.IsError {
		t.Fatalf("list repos: %s", textOf(got))
	}

	if got := sn.Call("orbit_inspect_repo", map[string]any{"repo": r.Name}); got.IsError {
		t.Fatalf("inspect repo: %s", textOf(got))
	}

	if got := sn.Call("orbit_add_repo", map[string]any{"path": r.Path}); got.IsError {
		t.Fatalf("add repo: %s", textOf(got))
	}

	if got := sn.Call("orbit_forget_repo", map[string]any{"name": r.Name}); !got.IsError {
		t.Error("forgetting a repo on the first asking was accepted")
	}
}
