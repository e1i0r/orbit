package engine

import (
	"reflect"
	"testing"
)

func TestCodexInterface(t *testing.T) {
	c := NewCodex()
	if c.Name() != "codex" {
		t.Errorf("Name() = %q, want codex", c.Name())
	}

	if !c.CanResume() {
		t.Error("CanResume() = false, want true")
	}

	if !c.CanThink() {
		t.Error("CanThink() = false, want true")
	}

	if len(c.Models()) == 0 {
		t.Error("Models() is empty")
	}

	if len(c.Efforts()) == 0 {
		t.Error("Efforts() is empty")
	}
}

func TestCodexArgs(t *testing.T) {
	req := Request{
		Prompt:      "fix the bug",
		Model:       "o3-mini",
		Effort:      "high",
		Resume:      "sess-123",
		Permissions: []string{PermissionRead, PermissionRepo},
	}

	got, err := codexArgs(req)
	if err != nil {
		t.Fatalf("codexArgs: %v", err)
	}

	want := []string{"exec", "--model", "o3-mini", "--effort", "high", "--resume", "sess-123", "fix the bug"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("codexArgs = %v, want %v", got, want)
	}
}
