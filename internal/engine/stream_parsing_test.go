package engine

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestParseStreamWithThinkingAndToolCalls(t *testing.T) {
	streamData := strings.Join([]string{
		`{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"Analyzing the AST"}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"thinking","text":"Second thought with text field"}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"grep","input":{"pattern":"func main"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","name":"psql","content":"permission denied","is_error":true}]}}`,
		`{"type":"refusal","subtype":"bash","result":"command blocked"}`,
		`{"type":"result","result":"Refactored successfully","session_id":"sess-xyz","total_cost_usd":0.082}`,
	}, "\n")

	res, err := ParseStream(strings.NewReader(streamData))
	if err != nil {
		t.Fatalf("ParseStream: %v", err)
	}

	if res.Output != "Refactored successfully" {
		t.Errorf("res.Output = %q, want Refactored successfully", res.Output)
	}
	if res.SessionID != "sess-xyz" {
		t.Errorf("res.SessionID = %q, want sess-xyz", res.SessionID)
	}
	if res.Cost != 0.082 {
		t.Errorf("res.Cost = %f, want 0.082", res.Cost)
	}
	if len(res.Thoughts) != 2 {
		t.Errorf("res.Thoughts = %v, want 2 thoughts", res.Thoughts)
	}
	if len(res.ToolCalls) != 1 || res.ToolCalls[0].Name != "grep" {
		t.Errorf("res.ToolCalls = %v, want 1 grep call", res.ToolCalls)
	}
	if len(res.Refusals) != 2 {
		t.Errorf("res.Refusals = %v, want 2 refusals", res.Refusals)
	}
}

func TestParseStreamEdgeCases(t *testing.T) {
	// Empty reader reports missing result object
	_, err := ParseStream(strings.NewReader(""))
	if err == nil {
		t.Error("expected ParseStream(empty) to report missing result object error")
	}

	// Non-JSON noise without result object
	noise := "non-json string\n{broken json\n{}\n"
	_, err2 := ParseStream(strings.NewReader(noise))
	if err2 == nil {
		t.Error("expected ParseStream(noise) to report missing result object error")
	}

	// noteDropped helper
	if noteDropped("out", 0) != "out" {
		t.Error("noteDropped with 0 should return string unchanged")
	}
	if !strings.Contains(noteDropped("out", 500), "dropped") {
		t.Error("noteDropped with >0 should append dropped notice")
	}
}

func TestEngineInvalidPermissionsRejection(t *testing.T) {
	invalidReq := Request{
		Prompt:      "test",
		Permissions: []string{"invalid-permission-xyz"},
	}

	c := NewCodex()
	if _, err := c.Run(context.Background(), invalidReq); err == nil {
		t.Error("expected Codex.Run to fail on invalid permissions")
	}

	o := NewOpenCode()
	if _, err := o.Run(context.Background(), invalidReq); err == nil {
		t.Error("expected OpenCode.Run to fail on invalid permissions")
	}

	cl := NewClaude()
	if _, err := cl.Run(context.Background(), invalidReq); err == nil {
		t.Error("expected Claude.Run to fail on invalid permissions")
	}
}

func TestOpenCodeArgsAndOptions(t *testing.T) {
	o := NewOpenCode()
	if o.Name() != "opencode" {
		t.Errorf("o.Name() = %q, want opencode", o.Name())
	}
	if !o.CanResume() || !o.CanThink() {
		t.Error("OpenCode should support CanResume and CanThink")
	}
	if len(o.Models()) == 0 || len(o.Efforts()) == 0 {
		t.Error("OpenCode Models() and Efforts() should not be empty")
	}

	req := Request{
		Prompt:      "build feature",
		Model:       "qwen-2.5-coder",
		Effort:      "high",
		Resume:      "sess-99",
		Permissions: []string{PermissionRead},
	}
	args, err := openCodeArgs(req)
	if err != nil {
		t.Fatalf("openCodeArgs: %v", err)
	}
	if len(args) == 0 || args[0] != "run" {
		t.Errorf("args = %v, want run command", args)
	}
}

func FuzzParseStream(f *testing.F) {
	f.Add([]byte(`{"type":"result","result":"ok","total_cost_usd":0.01}`))
	f.Add([]byte(`{"type":"assistant","message":{"content":[{"type":"thinking","text":"think"}]}}`))
	f.Add([]byte("random unformatted byte stream\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseStream(bytes.NewReader(data)) //nolint:errcheck // fuzz testing against arbitrary input
	})
}
