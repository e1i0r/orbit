package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinTaskFlowLoads(t *testing.T) {
	f, err := Builtin("task")
	if err != nil {
		t.Fatalf("Builtin: %v", err)
	}
	if f.Name != "task" {
		t.Errorf("Name = %q, want task", f.Name)
	}
	if len(f.Phases) == 0 {
		t.Fatal("the built-in task flow has no phases")
	}
	if f.Phases[0].Name != "implement" {
		t.Errorf("first phase is %q, want implement", f.Phases[0].Name)
	}
}

func TestBuiltinRejectsAnUnknownName(t *testing.T) {
	if _, err := Builtin("nope"); err == nil {
		t.Error("Builtin succeeded for a flow that does not exist")
	}
}

func TestBuiltinNamesListsWhatExists(t *testing.T) {
	names := BuiltinNames()
	if len(names) == 0 {
		t.Fatal("no built-in flows")
	}
	found := false
	for _, n := range names {
		if n == "task" {
			found = true
		}
	}
	if !found {
		t.Errorf("BuiltinNames() = %v, expected it to contain task", names)
	}
}

func TestLoadReadsAFlowFromDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mine.json")
	body := `{"name":"mine","phases":[{"name":"implement","engine":"claude","model":"sonnet","wait":false}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if f.Name != "mine" || len(f.Phases) != 1 || f.Phases[0].Engine != "claude" {
		t.Errorf("loaded %+v", f)
	}
}

func TestValidateRejectsAFlowWithNoPhases(t *testing.T) {
	err := Flow{Name: "empty"}.Validate()
	if err == nil {
		t.Error("a flow with no phases validated")
	}
}

func TestValidateRejectsAPhaseWithNoName(t *testing.T) {
	f := Flow{Name: "x", Phases: []Phase{{Engine: "claude"}}}
	if err := f.Validate(); err == nil {
		t.Error("a phase with no name validated")
	}
}

func TestValidateRejectsAPhaseWithNoEngine(t *testing.T) {
	f := Flow{Name: "x", Phases: []Phase{{Name: "implement"}}}
	if err := f.Validate(); err == nil {
		t.Error("a phase with no engine validated")
	}
}

func TestValidateRejectsTwoPhasesWithOneName(t *testing.T) {
	f := Flow{Name: "x", Phases: []Phase{
		{Name: "implement", Engine: "claude"},
		{Name: "implement", Engine: "claude"},
	}}
	if err := f.Validate(); err == nil {
		t.Error("two phases named the same validated — the record could not tell them apart")
	}
}

func TestWithAutopilotClearsEveryWait(t *testing.T) {
	f := Flow{Name: "x", Phases: []Phase{
		{Name: "implement", Engine: "claude", Wait: true},
		{Name: "review", Engine: "claude", Wait: true},
	}}
	got := f.WithAutopilot()
	for _, p := range got.Phases {
		if p.Wait {
			t.Errorf("phase %q still waits with autopilot on", p.Name)
		}
	}
	if !f.Phases[0].Wait {
		t.Error("WithAutopilot modified the original flow instead of returning a new one")
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "typo.json")
	body := `{"name":"mine","phases":[{"name":"implement","engine":"claude","mdoel":"sonnet"}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Error("Load succeeded despite unknown field mdoel")
	}
}

func TestWithAutopilotDoesNotWriteThroughToTheOriginalsPermissions(t *testing.T) {
	f := Flow{Name: "x", Phases: []Phase{
		{Name: "implement", Engine: "claude", Wait: true, Permissions: []string{"repo"}},
	}}
	got := f.WithAutopilot()
	got.Phases[0].Permissions[0] = "everything"
	if f.Phases[0].Permissions[0] != "repo" {
		t.Errorf("the original's permissions became %q — the copy shares its backing array, so the caller's flow was rewritten under it", f.Phases[0].Permissions[0])
	}
}

func TestLoadOfAFileThatIsNotThere(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nothing.json")); err == nil {
		t.Error("Load succeeded on a file that does not exist")
	}
}

func TestLoadOfSomethingThatIsNotJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(path, []byte("{\"name\": \"mine\", \"phases\": ["), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load accepted a file of broken JSON")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the error does not say which file will not parse: %v", err)
	}
}
