package mcp

// Writing a task down through the server: the id it gets, the repository it
// lands in, and the two questions it refuses to answer by guessing.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

// TestCreateMintsAnIdFromTheRepositoryAndTheNextFreeNumber is the fourth
// finding: the id was fmt.Sprintf("TASK-%d", os.Getpid()), which is the same
// id for every task one process writes, so the second create in a session
// was refused as a duplicate.
func TestCreateMintsAnIdFromTheRepositoryAndTheNextFreeNumber(t *testing.T) {
	_, sn, r := oneRepo(t)
	first := call(t, sn, "orbit_create_task", map[string]any{"title": "one"})
	second := call(t, sn, "orbit_create_task", map[string]any{"title": "two"})

	if first["id"] == second["id"] {
		t.Fatalf("two tasks were both minted %v", first["id"])
	}
	for _, got := range []any{first["id"], second["id"]} {
		id, ok := got.(string)
		if !ok {
			t.Fatalf("an id came back as %#v, want a string", got)
		}
		if !strings.HasPrefix(id, "PAYMENTS-") {
			t.Errorf("id %q does not name the repository it was written against", id)
		}
		if err := store.ValidTaskID(id); err != nil {
			t.Errorf("id %q is not one the store will accept: %v", id, err)
		}
	}
	if first["repo"] != r.Name {
		t.Errorf("the task was written against %v, want %q", first["repo"], r.Name)
	}
	if first["started"] != false {
		t.Error("orbit_create_task reports the task as started; writing one down and spending money on it are two decisions")
	}
}

func TestCreateHonoursAnIdTheCallerChose(t *testing.T) {
	_, sn, _ := oneRepo(t)
	got := call(t, sn, "orbit_create_task", map[string]any{"title": "one", "id": "ACME-7"})
	if got["id"] != "ACME-7" {
		t.Errorf("the caller asked for ACME-7 and got %v", got["id"])
	}
}

// TestCreateRefusesAnIdThatWouldEscapeTheStore. store.ValidTaskID is the
// only thing between a typed id and the filesystem, and a second front door
// has to ask it too.
func TestCreateRefusesAnIdThatWouldEscapeTheStore(t *testing.T) {
	_, sn, _ := oneRepo(t)
	for _, id := range []string{"../escape", "a/b", ".."} {
		said := refused(t, sn, "orbit_create_task", map[string]any{"title": "one", "id": id})
		if !strings.Contains(said, id) {
			t.Errorf("the refusal of %q does not name it: %s", id, said)
		}
	}
}

func TestCreateNeedsATitle(t *testing.T) {
	_, sn, _ := oneRepo(t)
	if said := refused(t, sn, "orbit_create_task", map[string]any{"prompt": "a body and no title"}); !strings.Contains(said, "title") {
		t.Errorf("the refusal does not say what is missing: %s", said)
	}
}

// TestCreateJoinsTheTitleAndThePromptIntoTheTaskItself: the text of task.md
// is everything the engines are told, and a prompt that did not reach it
// would be an instruction the run never sees.
func TestCreateJoinsTheTitleAndThePromptIntoTheTask(t *testing.T) {
	_, sn, _ := oneRepo(t)
	created := call(t, sn, "orbit_create_task", map[string]any{"title": "fix the parser", "prompt": "start from the failing test"})
	got := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": created["id"]})
	body, ok := got["text"].(string)
	if !ok {
		t.Fatalf("the inspection carries the task as %#v, want a string", got["text"])
	}
	if !strings.Contains(body, "fix the parser") || !strings.Contains(body, "start from the failing test") {
		t.Errorf("the task was written as %q, want both the title and the prompt", body)
	}
}

// TestCreateSaysWhichRepositoryWhenThereIsMoreThanOne. Writing a task into
// whichever repository sorted first is the kind of helpfulness that files
// work against the wrong project.
func TestCreateSaysWhichRepositoryWhenThereIsMoreThanOne(t *testing.T) {
	_, work := newRoot(t)
	gitRepo(t, work, "payments")
	gitRepo(t, work, "ledger")
	sn := Session{Root: work, Version: "test"}

	said := refused(t, sn, "orbit_create_task", map[string]any{"title": "one"})
	for _, name := range []string{"payments", "ledger"} {
		if !strings.Contains(said, name) {
			t.Errorf("the refusal does not offer %q: %s", name, said)
		}
	}
	got := call(t, sn, "orbit_create_task", map[string]any{"title": "one", "repo": "ledger"})
	if got["repo"] != "ledger" {
		t.Errorf("the task was written against %v, want ledger", got["repo"])
	}
}
