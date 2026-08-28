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

	// Models is deliberately just "default": codex's catalogue is a fact
	// about the account behind the login, not about the binary, so this
	// package holds no list of its own to go stale.
	if len(c.Models()) != 1 || c.Models()[0].ID != "" {
		t.Errorf("Models() = %v, want only the default choice", c.Models())
	}

	if len(c.Efforts()) == 0 {
		t.Error("Efforts() is empty")
	}
}

// TestCodexEffortsAreOnesCodexTakes. The values pass two gates, not one:
// codex parses them against its own enum — minimal, low, medium, high — and
// the model behind it answers with its own, which for gpt-5.6 is none, low,
// medium, high, xhigh and max. Only the overlap can ever run.
func TestCodexEffortsAreOnesCodexTakes(t *testing.T) {
	codexTakes := map[string]bool{"minimal": true, "low": true, "medium": true, "high": true}
	modelTakes := map[string]bool{"none": true, "low": true, "medium": true, "high": true, "xhigh": true, "max": true}

	for _, e := range NewCodex().Efforts() {
		if e.ID == "" {
			continue
		}

		if !codexTakes[e.ID] {
			t.Errorf("effort %q is offered, and codex answers it with `unknown variant`", e.ID)
		}

		if !modelTakes[e.ID] {
			t.Errorf("effort %q parses, and then the model answers it with `unsupported value`", e.ID)
		}
	}
}

// TestCodexArgs is the exact argv, in the exact order, that was run against
// codex 0.46.0 and parsed.
//
// The three it replaces were invented. `codex exec --effort high` answers
// "error: unexpected argument '--effort' found", so every phase that named
// an effort died before a model saw it; `--resume` answers the same, because
// resume is a subcommand; and the options belong to exec and must come
// before that subcommand — `codex exec resume --sandbox` is refused too.
func TestCodexArgs(t *testing.T) {
	req := Request{
		Prompt:      "fix the bug",
		Model:       "gpt-5.1-codex",
		Effort:      "high",
		Resume:      "sess-123",
		Permissions: []string{PermissionRead, PermissionRepo},
	}

	got, err := codexArgs(req)
	if err != nil {
		t.Fatalf("codexArgs: %v", err)
	}

	want := []string{
		"exec", "--json",
		"--sandbox", "workspace-write",
		"-c", "sandbox_workspace_write.network_access=false",
		"--model", "gpt-5.1-codex",
		"-c", "model_reasoning_effort=high",
		"resume", "sess-123",
		"fix the bug",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("codexArgs = %v, want %v", got, want)
	}
}

// TestCodexStatesItsPosture. codexArgs used to validate the permission names
// and then build an argv that said nothing about them, so every codex phase
// ran at whatever the binary defaulted to while the record claimed a
// posture. Each line below is a sandbox codex names.
func TestCodexStatesItsPosture(t *testing.T) {
	for _, c := range []struct {
		perms []string
		want  []string
		why   string
	}{
		{
			nil,
			[]string{"--sandbox", "read-only"},
			"a phase that asked for nothing gets codex's floor, which is read",
		},
		{
			[]string{PermissionRead},
			[]string{"--sandbox", "read-only"},
			"read is read-only",
		},
		{[]string{PermissionRepo}, []string{
			"--sandbox", "workspace-write",
			"-c", "sandbox_workspace_write.network_access=false",
		}, "repo may write the worktree and is told, out loud, that it may not dial out"},
		{[]string{PermissionRepo, PermissionNetwork}, []string{
			"--sandbox", "workspace-write",
			"-c", "sandbox_workspace_write.network_access=true",
		}, "network is the config key that opens the writing sandbox up"},
	} {
		got, err := codexPermissionArgs(c.perms)
		if err != nil {
			t.Errorf("codexPermissionArgs(%v): %v", c.perms, err)
			continue
		}

		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("codexPermissionArgs(%v) = %v, want %v — %s", c.perms, got, c.want, c.why)
		}
	}
}

// TestCodexRefusesNetworkItCannotGrantAlone. codex has no sandbox that
// reaches the network without also granting writes, so honouring a network
// posture on its own would mean handing out write access nobody asked for.
// A posture this package cannot state is a run that does not start.
func TestCodexRefusesNetworkItCannotGrantAlone(t *testing.T) {
	if _, err := codexPermissionArgs([]string{PermissionNetwork}); err == nil {
		t.Error("codex accepted a network-only posture it has no sandbox for")
	}

	if _, err := codexArgs(Request{Prompt: "x", Permissions: []string{PermissionNetwork}}); err == nil {
		t.Error("codexArgs built a command line for a posture codex cannot state")
	}
}
