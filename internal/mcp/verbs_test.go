package mcp

// The tools built from the declaration.

import (
	"regexp"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
)

// TestAToolBuiltFromTheDeclarationDeclaresTheVerbsFields is the other half
// of the test above, for the tools nobody wrote by hand: they are honest
// because their schema is the verb's own field list, and this is what says
// so still holds.
func TestAToolBuiltFromTheDeclarationDeclaresTheVerbsFields(t *testing.T) {
	by := map[string]verb.Verb{}
	for _, v := range verb.Every() {
		by["orbit_"+strings.ReplaceAll(v.Name, "-", "_")] = v
	}

	for _, tool := range verbTools() {
		v, known := by[tool.Name]
		if !known {
			t.Errorf("tool %s is built from no verb", tool.Name)
			continue
		}

		want := map[string]bool{}
		if v.OnTask {
			want["task_id"] = true
		}

		for _, f := range v.Takes {
			want[f.Name] = true
		}

		for name := range tool.InputSchema.Properties {
			if !want[name] {
				t.Errorf("%s declares %q, which %s does not take", tool.Name, name, v.Name)
			}
		}

		for name := range want {
			if _, ok := tool.InputSchema.Properties[name]; !ok {
				t.Errorf("%s does not declare %q, which %s takes", tool.Name, name, v.Name)
			}
		}
	}
}

// TestEveryDeclaredArgumentIsOneAHandlerReads guards the defect that made
// this list dishonest once already: orbit_create_task declared engine and
// model, neither of which reached task.Create, so a model passed them, read
// a success, and believed they took effect.
//
// It reads both ways. A row here naming an argument no tool declares is a
// handler still reaching for something a model can no longer pass, which is
// how the repo argument on the task tools would have outlived the pair
// identity that needed it.
func TestEveryDeclaredArgumentIsOneAHandlerReads(t *testing.T) {
	read := map[string]map[string]bool{
		"orbit_get_board_summary":  {},
		"orbit_list_tasks":         {"band": true, "repo": true},
		"orbit_inspect_task":       {"task_id": true},
		"orbit_create_task":        {"title": true, "prompt": true, "repo": true, "flow": true, "id": true},
		"orbit_retry_task":         {"task_id": true, "corrective_prompt": true, "flow": true},
		"orbit_add_note":           {"task_id": true, "text": true},
		"orbit_pause_task":         {"task_id": true},
		"orbit_direct_task":        {"task_id": true, "message": true, "restart": true},
		"orbit_cancel_task":        {"task_id": true},
		"orbit_requeue_task":       {"task_id": true, "why": true},
		"orbit_list_flows":         {},
		"orbit_get_flow":           {"name": true},
		"orbit_save_flow":          {"name": true, "description": true, "from": true, "phases": true, "attempts": true},
		"orbit_delete_flow":        {"name": true, "force": true},
		"orbit_list_repos":         {},
		"orbit_inspect_repo":       {"repo": true},
		"orbit_add_repo":           {"path": true},
		"orbit_forget_repo":        {"repo": true, "delete_tasks": true},
		"orbit_learn":              {"phrase": true, "repo": true, "lang": true, "stops": true, "check": true, "task_id": true},
		"orbit_knowledge":          {"repo": true},
		"orbit_supervisor_say":     {"message": true, "by": true, "channel": true, "task_id": true, "repo": true},
		"orbit_supervisor_history": {"limit": true},
	}
	// A tool built from the declaration is honest by construction: its
	// arguments are the verb's own fields, and askVerb hands every one of
	// them to the verb by that name. What is checked for those is that the
	// schema still says what the declaration says — see below.
	built := map[string]bool{}
	for _, one := range verbTools() {
		built[one.Name] = true
	}

	for _, tool := range Tools() {
		if built[tool.Name] {
			continue
		}

		handled, ok := read[tool.Name]
		if !ok {
			t.Errorf("tool %s is advertised and this test does not say which of its arguments a handler reads", tool.Name)
			continue
		}

		for name := range tool.InputSchema.Properties {
			if !handled[name] {
				t.Errorf("%s declares %q and no handler reads it", tool.Name, name)
			}
		}

		for _, name := range tool.InputSchema.Required {
			if _, ok := tool.InputSchema.Properties[name]; !ok {
				t.Errorf("%s requires %q and does not declare it", tool.Name, name)
			}
		}

		for name := range handled {
			if _, ok := tool.InputSchema.Properties[name]; !ok {
				t.Errorf("%s is said to read %q and does not declare it", tool.Name, name)
			}
		}
	}
}

// TestNoToolIsListedTwice. The list is two halves joined, and a name in
// both would be dispatched to whichever case came first in the switch while
// the client showed the model two tools it could not tell apart.
func TestNoToolIsListedTwice(t *testing.T) {
	seen := map[string]bool{}
	for _, tool := range Tools() {
		if seen[tool.Name] {
			t.Errorf("%s is listed twice", tool.Name)
		}

		seen[tool.Name] = true
	}
}

func TestUnparseableJSONIsAParseError(t *testing.T) {
	_, sn, _ := oneRepo(t)

	got := exchange(t, sn, `{not json at all`)
	if len(got) != 1 {
		t.Fatalf("a broken line drew %d responses, want 1", len(got))
	}

	if got[0].Error == nil || got[0].Error.Code != CodeParseError {
		t.Errorf("a broken line answered %+v, want a parse error", got[0])
	}

	if got[0].ID != nil {
		t.Errorf("a parse error answered with id %v, want null: the id is what failed to parse", got[0].ID)
	}
}

func TestAnUnknownMethodIsRefusedByName(t *testing.T) {
	_, sn, _ := oneRepo(t)

	got := exchange(t, sn, `{"jsonrpc":"2.0","id":9,"method":"tools/invent"}`)
	if got[0].Error == nil || got[0].Error.Code != CodeMethodNotFound {
		t.Fatalf("an unknown method answered %+v, want method-not-found", got[0])
	}

	if !strings.Contains(got[0].Error.Message, "tools/invent") {
		t.Errorf("the refusal does not name the method: %q", got[0].Error.Message)
	}
}

// TestAToolThatSaysNoIsNotATransportError is the distinction the protocol
// draws and the reason this server keeps them apart: a client turns an error
// object into a failure the model never sees, and "there is no task ORB-9"
// is exactly the thing the model has to read and correct.
func TestAToolThatSaysNoIsNotATransportError(t *testing.T) {
	_, sn, _ := oneRepo(t)

	got := exchange(t, sn, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"orbit_inspect_task","arguments":{"task_id":"NOPE-1"}}}`)
	if got[0].Error != nil {
		t.Fatalf("a tool that refused came back as a transport error: %+v", got[0].Error)
	}

	result, ok := got[0].Result.(map[string]any)
	if !ok || result["isError"] != true {
		t.Errorf("a refusal did not set isError: %v", got[0].Result)
	}
}

func TestACallWithNoToolNameIsInvalidParams(t *testing.T) {
	_, sn, _ := oneRepo(t)

	got := exchange(t, sn, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"arguments":{}}}`)
	if got[0].Error == nil || got[0].Error.Code != CodeInvalidParams {
		t.Errorf("a nameless call answered %+v, want invalid params", got[0])
	}
}

func TestPingIsAnswered(t *testing.T) {
	_, sn, _ := oneRepo(t)

	got := exchange(t, sn, `{"jsonrpc":"2.0","id":5,"method":"ping"}`)
	if got[0].Error != nil {
		t.Errorf("ping answered %+v", got[0].Error)
	}
}

// TestBlankLinesAreSkipped: a client that flushes a newline of its own must
// not end the session with a parse error.
func TestBlankLinesAreSkipped(t *testing.T) {
	_, sn, _ := oneRepo(t)

	got := exchange(t, sn, ``, `{"jsonrpc":"2.0","id":6,"method":"ping"}`, ``)
	if len(got) != 1 {
		t.Errorf("blank lines drew %d responses, want 1", len(got))
	}
}

// TestAWriteThatFailsEndsTheSession. A response that could not be written is
// a client waiting for ever, which is worse than a server that exits.
func TestAWriteThatFailsEndsTheSession(t *testing.T) {
	_, sn, _ := oneRepo(t)
	in := strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"ping"}` + "\n")

	err := NewServer(in, brokenWriter{}, sn).Serve()
	if err == nil {
		t.Error("a response that could not be written was dropped and the session carried on")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errBroken }

var errBroken = errWriter("the client is gone")

type errWriter string

func (e errWriter) Error() string { return string(e) }

// TestEveryToolTheInstructionsNameIsOneThisServerHas. The instructions are
// the only prose a model reads before it has called anything, so a tool
// renamed in the list and not here leaves the first thing the model is told
// pointing at nothing.
func TestEveryToolTheInstructionsNameIsOneThisServerHas(t *testing.T) {
	named := regexp.MustCompile(`orbit_[a-z_]+`).FindAllString(instructions, -1)
	if len(named) == 0 {
		t.Fatal("the instructions name no tool at all")
	}

	has := make(map[string]bool, len(toolNames()))
	for _, n := range toolNames() {
		has[n] = true
	}

	for _, n := range named {
		if !has[n] {
			t.Errorf("the instructions send a model to %q, which this server does not have", n)
		}
	}
}

// TestEveryToolThatStartsARunIsNamedAsOneThatCostsMoney. Instructions that
// warn about retry and cancel and say nothing about direct leave out a tool
// whose restart starts a run exactly as retry does — the one a supervisor
// reaches for mid-run would be the one it had not been told to announce.
func TestEveryToolThatStartsARunIsNamedAsOneThatCostsMoney(t *testing.T) {
	costly := regexp.MustCompile(`([^.]*cost money[^.]*\.)`).FindString(instructions)
	if costly == "" {
		t.Fatal("the instructions no longer say which tools cost money")
	}

	for _, name := range []string{"orbit_retry_task", "orbit_direct_task", "orbit_cancel_task"} {
		if !strings.Contains(costly, name) {
			t.Errorf("%q starts a run and is not in the sentence that says which tools cost money: %s", name, costly)
		}
	}
}
