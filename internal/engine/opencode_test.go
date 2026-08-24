package engine

import (
	"reflect"
	"testing"
)

func TestOpenCodeInterface(t *testing.T) {
	o := NewOpenCode()
	if o.Name() != "opencode" {
		t.Errorf("Name() = %q, want opencode", o.Name())
	}
	if !o.CanResume() {
		t.Error("CanResume() = false, want true")
	}
	if !o.CanThink() {
		t.Error("CanThink() = false, want true")
	}
	if len(o.Models()) == 0 {
		t.Error("Models() is empty")
	}
	if len(o.Efforts()) == 0 {
		t.Error("Efforts() is empty")
	}
}

func TestOpenCodeArgs(t *testing.T) {
	req := Request{
		Prompt:      "refactor the handler",
		Model:       "deepseek-r1",
		Effort:      "medium",
		Resume:      "sess-456",
		Permissions: []string{PermissionRead, PermissionRepo},
	}
	got, err := openCodeArgs(req)
	if err != nil {
		t.Fatalf("openCodeArgs: %v", err)
	}
	want := []string{"run", "--model", "deepseek-r1", "--effort", "medium", "--session", "sess-456", "refactor the handler"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("openCodeArgs = %v, want %v", got, want)
	}
}
