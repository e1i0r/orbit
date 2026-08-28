package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEngineCapabilitiesAndMetadata(t *testing.T) {
	// 1. Claude
	cl := NewClaude()
	if !cl.CanResume() || !cl.CanThink() {
		t.Error("Claude should support CanResume and CanThink")
	}

	if len(cl.Models()) != 4 {
		t.Errorf("Claude Models = %v, want 4 choices", cl.Models())
	}

	if len(cl.Efforts()) != 6 {
		t.Errorf("Claude Efforts = %v, want 6 choices", cl.Efforts())
	}

	// 2. Fake
	fk := NewFake("result")
	if fk.CanResume() || fk.CanThink() {
		t.Error("Fake should return false for CanResume and CanThink")
	}

	if fk.Models() != nil || fk.Efforts() != nil {
		t.Error("Fake should return nil for Models and Efforts")
	}
}

func TestClaudeArgsComprehensive(t *testing.T) {
	req := Request{
		Prompt:      "write tests",
		Model:       "sonnet",
		Effort:      "high",
		Thinking:    "adaptive",
		Resume:      "sess-abc",
		Permissions: []string{PermissionRead, PermissionRepo, PermissionNetwork},
	}

	args, err := claudeArgs(req)
	if err != nil {
		t.Fatalf("claudeArgs: %v", err)
	}

	argStr := argsToString(args)
	if !contains(argStr, "--model") || !contains(argStr, "sonnet") {
		t.Errorf("missing model arg in %v", args)
	}

	if !contains(argStr, "--effort") || !contains(argStr, "high") {
		t.Errorf("missing effort arg in %v", args)
	}

	if !contains(argStr, "--resume") || !contains(argStr, "sess-abc") {
		t.Errorf("missing resume arg in %v", args)
	}
}

func TestEngineExecutionErrors(t *testing.T) {
	req := Request{
		Prompt: "do task",
		Dir:    "/non/existent/directory",
	}

	// Running in non-existent directory exercises command failure
	cl := NewClaude()

	_, err := cl.Run(context.Background(), req)
	if err == nil {
		t.Error("expected Claude.Run to fail in non-existent dir")
	}

	co := NewCodex()

	_, err = co.Run(context.Background(), req)
	if err == nil {
		t.Error("expected Codex.Run to fail in non-existent dir")
	}
}

// TestEngineMockSuccessfulRun gives each engine a stand-in that prints that
// engine's own event stream.
//
// One script printing claude's stream used to stand in for all three, and it
// passed — because all three parsed with claude's parser, and the plain-text
// fallback caught whatever did not. That is the bug in test form: codex says
// thread_id, opencode says sessionID and claude says session_id, so a run on
// codex or opencode recorded no session and no cost while looking exactly
// like one that had.
func TestEngineMockSuccessfulRun(t *testing.T) {
	dir := t.TempDir()

	for _, c := range []struct {
		name, line string
	}{
		{"claude", `{"type":"result","result":"mock success","session_id":"claude-sess","total_cost_usd":0.05}`},
		{"codex", `{"type":"thread.started","thread_id":"codex-sess"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"mock success"}}
{"type":"turn.completed","usage":{"output_tokens":5}}`},
		{"opencode", `{"type":"text","sessionID":"opencode-sess","part":{"type":"text","text":"mock success"}}
{"type":"step_finish","sessionID":"opencode-sess","part":{"type":"step-finish","reason":"stop","cost":0.05}}`},
	} {
		script := "#!/bin/sh\ncat <<'EOF'\n" + c.line + "\nEOF\n"
		if err := os.WriteFile(filepath.Join(dir, c.name), []byte(script), 0o755); err != nil {
			t.Fatalf("WriteFile %s: %v", c.name, err)
		}
	}

	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	for _, c := range []struct {
		eng     Engine
		session string
	}{
		{NewClaude(), "claude-sess"},
		{NewCodex(), "codex-sess"},
		{NewOpenCode(), "opencode-sess"},
	} {
		name := c.eng.Name()

		res, err := c.eng.Run(context.Background(), Request{
			Prompt:      "run test",
			Dir:         dir,
			Permissions: []string{PermissionRepo},
		})
		if err != nil {
			t.Errorf("%s.Run: %v", name, err)
			continue
		}

		if res.Output != "mock success" {
			t.Errorf("%s.Run output = %q, want %q", name, res.Output, "mock success")
		}

		if res.SessionID != c.session {
			t.Errorf("%s.Run session = %q, want %q — without it there is nothing to resume from",
				name, res.SessionID, c.session)
		}
	}
}

func argsToString(args []string) string {
	var s string
	for _, a := range args {
		s += " " + a
	}

	return s
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}

	return false
}
