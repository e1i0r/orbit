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
