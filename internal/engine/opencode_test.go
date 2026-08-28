package engine

import (
	"reflect"
	"strings"
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
		Model:       "opencode/claude-sonnet-5",
		Effort:      "medium",
		Resume:      "sess-456",
		Permissions: []string{PermissionRead, PermissionRepo},
	}

	got, err := openCodeArgs(req)
	if err != nil {
		t.Fatalf("openCodeArgs: %v", err)
	}

	want := []string{"run", "--model", "opencode/claude-sonnet-5", "--effort", "medium", "--session", "sess-456", "refactor the handler"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("openCodeArgs = %v, want %v", got, want)
	}
}

// TestEveryOpenCodeModelIsOneOpenCodeWouldTake. `--model` goes to opencode
// unchanged, and opencode takes provider-qualified names: a bare one is
// refused. internal/task checks a phase's model against this list before
// running anything, so a list of names opencode rejects is a dial where
// every position is a task that cannot start.
func TestEveryOpenCodeModelIsOneOpenCodeWouldTake(t *testing.T) {
	var free int

	for _, m := range NewOpenCode().Models() {
		if m.ID == "" {
			// The empty ID is "whatever opencode is configured for",
			// which is not a model name and is not passed at all.
			continue
		}

		name, ok := strings.CutPrefix(m.ID, "opencode/")
		if !ok {
			t.Errorf("model %q is not provider-qualified; opencode refuses it", m.ID)
			continue
		}

		if m.Label != name {
			t.Errorf("model %q is labelled %q; the label is the name without the provider, so that what is picked and what is sent cannot drift", m.ID, m.Label)
		}

		if strings.HasSuffix(name, "-free") {
			free++
		}
	}

	paid := len(NewOpenCode().Models()) - 1 - free
	if free < 5 || paid < 5 {
		t.Errorf("opencode offers %d paid and %d free models, want at least five of each", paid, free)
	}
}
