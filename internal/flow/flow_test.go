package flow

import (
	"os"
	"path/filepath"
	"slices"
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

// TestValidateAcceptsTheWholeVocabulary pins the three names a flow file may
// use. They are duplicated in internal/engine, which maps them to a real
// command line; the duplication is deliberate — neither package may import
// the other — and this test is half of what keeps the two copies honest.
//
// The second half is Permissions(), which is the set as a caller is offered
// it rather than as one value is checked against it. A list that lost a name
// would leave Validate accepting a permission nothing ever told anybody
// about.
func TestValidateAcceptsTheWholeVocabulary(t *testing.T) {
	f := Flow{Name: "x", Phases: []Phase{{
		Name:        "implement",
		Engine:      "claude",
		Permissions: Permissions(),
	}}}
	if err := f.Validate(); err != nil {
		t.Errorf("Validate refused the vocabulary it defines: %v", err)
	}

	if want := []string{PermissionRead, PermissionRepo, PermissionNetwork}; !slices.Equal(Permissions(), want) {
		t.Errorf("Permissions() is %v, want %v — a schema built from it would offer the wrong set", Permissions(), want)
	}
}

// TestValidateRejectsAPermissionNobodyDefined is this gap's exact failure
// mode. A flow file that says "repository" where it meant "repo" must not
// be accepted: it would grant nothing and leave the engine's own default in
// charge — a wider posture than the file asked for, arriving through a typo,
// with nothing anywhere saying it had happened.
func TestValidateRejectsAPermissionNobodyDefined(t *testing.T) {
	f := Flow{Name: "x", Phases: []Phase{{
		Name:        "implement",
		Engine:      "claude",
		Permissions: []string{"repository"},
	}}}

	err := f.Validate()
	if err == nil {
		t.Fatal("a permission nobody defined validated")
	}

	if !strings.Contains(err.Error(), "repository") {
		t.Errorf("the error does not name the permission it refused: %v", err)
	}

	if !strings.Contains(err.Error(), "implement") {
		t.Errorf("the error does not say which phase carries it: %v", err)
	}
}

// TestEveryBuiltinFlowStillValidates is the check that the three flows
// shipped inside the binary did not become unloadable the day the
// vocabulary was closed. Builtin validates as it decodes, so a name outside
// the set turns every one of them into a run that cannot start.
func TestEveryBuiltinFlowStillValidates(t *testing.T) {
	for _, name := range BuiltinNames() {
		f, err := Builtin(name)
		if err != nil {
			t.Errorf("the built-in flow %q no longer loads: %v", name, err)
			continue
		}

		if err := f.Validate(); err != nil {
			t.Errorf("the built-in flow %q no longer validates: %v", name, err)
		}
	}
}
