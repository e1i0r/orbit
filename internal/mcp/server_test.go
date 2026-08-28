package mcp

// The protocol itself: the handshake, the tool list, and the two ways a call
// can fail. What the tools answer is handlers_test.go's.

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// exchange runs a session made of the given request lines and answers the
// responses that came back, decoded.
func exchange(t *testing.T, sn Session, lines ...string) []JSONRPCResponse {
	t.Helper()
	var in, out bytes.Buffer
	for _, line := range lines {
		in.WriteString(line + "\n")
	}
	if err := NewServer(&in, &out, sn).Serve(); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var responses []JSONRPCResponse
	for line := range strings.SplitSeq(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("a response is not JSON: %v\n%s", err, line)
		}
		responses = append(responses, resp)
	}
	return responses
}

func TestInitializeAnswersTheHandshake(t *testing.T) {
	_, sn, _ := oneRepo(t)
	got := exchange(t, sn, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if len(got) != 1 {
		t.Fatalf("initialize drew %d responses, want 1", len(got))
	}
	result, ok := got[0].Result.(map[string]any)
	if !ok {
		t.Fatalf("initialize answered %#v", got[0])
	}
	if result["protocolVersion"] != protocolVersion {
		t.Errorf("protocolVersion = %v, want %q", result["protocolVersion"], protocolVersion)
	}
	info, ok := result["serverInfo"].(map[string]any)
	if !ok || info["name"] != "orbit" {
		t.Errorf("serverInfo = %v, want it to name orbit", result["serverInfo"])
	}
	if info["version"] != "test" {
		t.Errorf("serverInfo version = %v, want the version the session was built with", info["version"])
	}
	caps, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities = %#v, want an object", result["capabilities"])
	}
	if _, ok := caps["tools"]; !ok {
		t.Errorf("the handshake does not declare the tools capability: %v", caps)
	}
}

// TestNotificationsAreNotAnswered is the rule a client enforces by treating
// an unsolicited response as a fault: a request with no id gets nothing
// back, not even a refusal.
func TestNotificationsAreNotAnswered(t *testing.T) {
	_, sn, _ := oneRepo(t)
	got := exchange(t, sn,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":1}}`,
		`{"jsonrpc":"2.0","method":"a_notification_this_server_has_never_heard_of"}`,
	)
	if len(got) != 0 {
		t.Errorf("notifications drew %d responses, want none: %+v", len(got), got)
	}
}

func TestToolsListNamesEveryTool(t *testing.T) {
	_, sn, _ := oneRepo(t)
	got := exchange(t, sn, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	result, ok := got[0].Result.(map[string]any)
	if !ok {
		t.Fatalf("tools/list answered %#v", got[0])
	}
	listed, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools/list carried no tools: %v", result)
	}
	if len(listed) != len(Tools()) {
		t.Errorf("tools/list carried %d tools, want %d", len(listed), len(Tools()))
	}
	for _, entry := range listed {
		tool, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("a tool is not an object: %#v", entry)
		}
		if tool["description"] == "" || tool["description"] == nil {
			t.Errorf("tool %v has no description; a tool a model cannot tell apart from another is a tool it will call wrongly", tool["name"])
		}
		schema, ok := tool["inputSchema"].(map[string]any)
		if !ok || schema["type"] != "object" {
			t.Errorf("tool %v has no object input schema: %v", tool["name"], tool["inputSchema"])
			continue
		}
		if _, ok := schema["properties"].(map[string]any); !ok {
			t.Errorf("tool %v encodes no properties object; a client reading null has to guess whether that means none", tool["name"])
		}
	}
}

// TestEveryToolInTheListIsOneTheServerRuns is the promise the schema makes.
// A tool advertised and not dispatched is a call that fails after the model
// has committed to it.
func TestEveryToolInTheListIsOneTheServerRuns(t *testing.T) {
	_, sn, _ := oneRepo(t)
	for _, tool := range Tools() {
		res := sn.Call(tool.Name, nil)
		if res.IsError && strings.Contains(text(t, res), "no tool is called") {
			t.Errorf("tools/list advertises %s and the server does not run it", tool.Name)
		}
	}
}

// TestEveryDeclaredArgumentIsOneAHandlerReads guards the defect that made
// this list dishonest once already: orbit_create_task declared engine and
// model, neither of which reached task.Create, so a model passed them, read
// a success, and believed they took effect.
func TestEveryDeclaredArgumentIsOneAHandlerReads(t *testing.T) {
	read := map[string]map[string]bool{
		"orbit_get_board_summary": {},
		"orbit_list_tasks":        {"band": true, "repo": true},
		"orbit_inspect_task":      {"task_id": true, "repo": true},
		"orbit_create_task":       {"title": true, "prompt": true, "repo": true, "flow": true, "id": true},
		"orbit_retry_task":        {"task_id": true, "repo": true, "corrective_prompt": true, "flow": true},
		"orbit_add_note":          {"task_id": true, "text": true, "repo": true},
		"orbit_pause_task":        {"task_id": true, "repo": true},
		"orbit_cancel_task":       {"task_id": true, "repo": true},
		"orbit_list_flows":        {},
		"orbit_get_flow":          {"name": true},
		"orbit_save_flow":         {"name": true, "description": true, "from": true, "phases": true},
		"orbit_delete_flow":       {"name": true, "force": true},
		"orbit_list_repos":        {},
		"orbit_inspect_repo":      {"repo": true},
		"orbit_add_repo":          {"path": true},
		"orbit_forget_repo":       {"repo": true, "delete_tasks": true},
	}
	for _, tool := range Tools() {
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
