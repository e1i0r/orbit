package engine

import (
	"reflect"
	"slices"
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

// TestOpenCodeArgs is the argv opencode 1.18.23 actually parses.
//
// opencode has no --effort — the flag is --variant — and it answers --effort
// by printing its help and exiting one, so a phase that names an effort that
// way fails before a model sees it. And without a flag asking for JSON,
// opencode prints formatted prose, which fed to claude's stream parser
// records no session id and no cost.
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

	want := []string{
		"run", "--format", "json", "--auto",
		"--model", "opencode/claude-sonnet-5",
		"--variant", "medium",
		"--session", "sess-456",
		"refactor the handler",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("openCodeArgs = %v, want %v", got, want)
	}
}

// TestOpenCodeRefusesAPostureItCannotKeep.
//
// This was run against the binary rather than reasoned about: asked to
// create a file, with no --auto and no terminal to prompt at, `opencode run`
// created the file. Headless opencode approves everything, so a read phase
// on opencode is a phase that can write — and running it would put a
// sentence in the record that the engine had already contradicted.
func TestOpenCodeRefusesAPostureItCannotKeep(t *testing.T) {
	for _, perms := range [][]string{
		nil,
		{PermissionRead},
		{PermissionNetwork},
		{PermissionRead, PermissionNetwork},
	} {
		if _, err := openCodeArgs(Request{Prompt: "x", Permissions: perms}); err == nil {
			t.Errorf("opencode accepted the posture %v, which it has no way to keep", perms)
		}
	}

	got, err := openCodeArgs(Request{Prompt: "x", Permissions: []string{PermissionRepo}})
	if err != nil {
		t.Fatalf("opencode refused repo, the one posture it can carry: %v", err)
	}

	if !slices.Contains(got, "--auto") {
		t.Errorf("args = %v, want --auto so the argv states what opencode does anyway", got)
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
