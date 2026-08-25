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

func TestEngineMockSuccessfulRun(t *testing.T) {
	dir := t.TempDir()
	mockScript := "#!/bin/sh\necho '{\"type\":\"result\",\"result\":\"mock success\",\"total_cost_usd\":0.05}'\n"

	for _, name := range []string{"claude", "codex", "opencode"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(mockScript), 0o755); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	req := Request{
		Prompt:      "run test",
		Dir:         dir,
		Permissions: []string{PermissionRead},
	}

	// 1. Claude
	cl := NewClaude()
	resCl, err := cl.Run(context.Background(), req)
	if err != nil || resCl.Output != "mock success" {
		t.Errorf("Claude.Run mock failed: %v, res=%+v", err, resCl)
	}

	// 2. Codex
	co := NewCodex()
	resCo, err := co.Run(context.Background(), req)
	if err != nil || resCo.Output != "mock success" {
		t.Errorf("Codex.Run mock failed: %v, res=%+v", err, resCo)
	}

	// 3. OpenCode
	op := NewOpenCode()
	resOp, err := op.Run(context.Background(), req)
	if err != nil || resOp.Output != "mock success" {
		t.Errorf("OpenCode.Run mock failed: %v, res=%+v", err, resOp)
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
