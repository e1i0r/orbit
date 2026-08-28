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

	// Every model the dial offers was run. A list nobody ran is what put
	// o3-mini and gpt-4o on this dial in the first place.
	want := []string{"", "gpt-5.6-terra", "gpt-5.6-luna", "gpt-5.5", "gpt-5.4-mini"}
	got := make([]string, 0, len(c.Models()))

	for _, m := range c.Models() {
		got = append(got, m.ID)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Models() = %v, want %v", got, want)
	}

	if len(c.Efforts()) == 0 {
		t.Error("Efforts() is empty")
	}
}

// TestCodexEffortsAreOnesEveryModelTakes.
//
// An effort passes two gates: codex's own config enum, and then the model's
// vocabulary. The dial is not per-model, so a value only some models accept
// is a position that fails depending on a different dial — which is why max
// is absent although codex takes it on gpt-5.6-terra and gpt-5.6-luna.
//
// Both maps are what `codex exec` answered on 0.150.1, one run per value.
func TestCodexEffortsAreOnesEveryModelTakes(t *testing.T) {
	everyModelTakes := map[string]bool{
		"none": true, "low": true, "medium": true, "high": true, "xhigh": true,
	}
	someModelsRefuse := map[string]string{
		"minimal": "unknown variant",
		"max":     "only gpt-5.6-terra and gpt-5.6-luna take it",
	}

	for _, e := range NewCodex().Efforts() {
		if e.ID == "" {
			continue
		}

		if why, refused := someModelsRefuse[e.ID]; refused {
			t.Errorf("effort %q is offered, and %s", e.ID, why)
			continue
		}

		if !everyModelTakes[e.ID] {
			t.Errorf("effort %q is offered and was never run against codex", e.ID)
		}
	}

	if len(NewCodex().Efforts()) != len(everyModelTakes)+1 {
		t.Errorf("Efforts() offers %d, want the default plus %d verified", len(NewCodex().Efforts()), len(everyModelTakes))
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
