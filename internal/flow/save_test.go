package flow

// The writing half. resolve_test.go covers reading a flow directory; this
// covers putting a file in one and taking it out again, and the two rules
// that make that safe: a flow is saved only if it would run, and a built-in
// is not a file anybody can remove.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aFlow is the smallest thing Save will accept, as a value rather than as
// the JSON onePhase writes.
func aFlow(name string) Flow {
	return Flow{Name: name, Phases: []Phase{{Name: "implement", Engine: "claude"}}}
}

func TestSaveWritesAFlowResolveReadsBack(t *testing.T) {
	src := flowsIn(t)
	want := Flow{
		Name:        "review",
		Description: "one pass over somebody else's work",
		Phases: []Phase{
			{Name: "read", Engine: "claude", Permissions: []string{PermissionRead}},
			{Name: "report", Engine: "claude", Prompt: "say what is wrong", Gates: []Gate{{Name: "build", Command: "go build ./..."}}},
		},
	}
	path, err := Save(src, want)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if dir := filepath.Dir(path); dir != src.FlowDir() {
		t.Errorf("saved into %q, want the flow directory %q", dir, src.FlowDir())
	}

	got, err := Resolve(src, "review")
	if err != nil {
		t.Fatalf("Resolve after Save: %v", err)
	}
	if got.Description != want.Description || len(got.Phases) != 2 {
		t.Fatalf("read back %+v, want the flow that was saved", got)
	}
	if got.Phases[1].Gates[0].Command != "go build ./..." {
		t.Errorf("gate command = %q, want the one that was saved", got.Phases[1].Gates[0].Command)
	}
	if perms := got.Phases[0].Permissions; len(perms) != 1 || perms[0] != PermissionRead {
		t.Errorf("permissions = %v, want the ones that were saved", perms)
	}
}

// A flow directory nobody has made yet is the ordinary case — nothing
// creates $ORBIT_HOME/flows — so the first save is the one that makes it,
// and it makes it as narrow as the rest of the state root.
func TestSaveMakesTheFlowDirectory(t *testing.T) {
	src := homeDir(filepath.Join(t.TempDir(), "flows"))
	if _, err := Save(src, aFlow("first")); err != nil {
		t.Fatalf("Save into a directory that is not there: %v", err)
	}
	info, err := os.Stat(src.FlowDir())
	if err != nil {
		t.Fatalf("stat the flow directory: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("flow directory mode = %o, want 700", perm)
	}
	file, err := os.Stat(filepath.Join(src.FlowDir(), "first.json"))
	if err != nil {
		t.Fatalf("stat the flow: %v", err)
	}
	if perm := file.Mode().Perm(); perm != 0o600 {
		t.Errorf("flow file mode = %o, want 600", perm)
	}
}

func TestSaveReplacesAFlowOfTheSameName(t *testing.T) {
	src := flowsIn(t)
	if _, err := Save(src, aFlow("review")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	second := aFlow("review")
	second.Phases = append(second.Phases, Phase{Name: "check", Engine: "codex"})
	if _, err := Save(src, second); err != nil {
		t.Fatalf("Save again: %v", err)
	}
	got, err := Resolve(src, "review")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(got.Phases) != 2 {
		t.Errorf("the flow has %d phases, want the 2 of the flow that was saved second", len(got.Phases))
	}
}

// Nothing unrunnable reaches the directory: a flow that fails Validate is
// refused before a file exists, so a reader never has to delete one that
// Resolve would then fail on.
func TestSaveRefusesAFlowThatCouldNotRun(t *testing.T) {
	src := flowsIn(t)
	for _, f := range []Flow{
		{Name: "empty"},
		{Name: "nameless", Phases: []Phase{{Engine: "claude"}}},
		{Name: "engineless", Phases: []Phase{{Name: "implement"}}},
		{Name: "wide", Phases: []Phase{{Name: "implement", Engine: "claude", Permissions: []string{"everything"}}}},
	} {
		if _, err := Save(src, f); err == nil {
			t.Errorf("Save(%q) was accepted, want it refused", f.Name)
		}
		if _, err := os.Stat(filepath.Join(src.FlowDir(), f.Name+".json")); !os.IsNotExist(err) {
			t.Errorf("Save(%q) left a file behind after refusing", f.Name)
		}
	}
}

func TestSaveRefusesANameThatIsAPath(t *testing.T) {
	src := flowsIn(t)
	for _, name := range []string{"", "../escape", "sub/flow", ".."} {
		if _, err := Save(src, Flow{Name: name, Phases: []Phase{{Name: "implement", Engine: "claude"}}}); err == nil {
			t.Errorf("Save(%q) was accepted, want it refused", name)
		}
	}
}

func TestSaveRefusesACallerWithNowhereToPutIt(t *testing.T) {
	if _, err := Save(nil, aFlow("review")); err == nil {
		t.Error("Save with no flow directory was accepted, want it refused")
	}
}

// Decode is the strict door the MCP server writes through: a field nobody
// declared is an error the caller reads, not a phase saved without it.
func TestDecodeRefusesAFieldNobodyDeclared(t *testing.T) {
	_, err := Decode([]byte(`{"name":"review","phases":[{"name":"implement","engines":"claude"}]}`), "review")
	if err == nil {
		t.Fatal("a phase with \"engines\" was accepted, want it refused")
	}
	if !strings.Contains(err.Error(), "engines") {
		t.Errorf("the error does not name the field that was wrong: %v", err)
	}
}

func TestDecodeChecksTheFlowItRead(t *testing.T) {
	if _, err := Decode([]byte(`{"name":"review","phases":[]}`), "review"); err == nil {
		t.Error("a flow with no phases was accepted, want it refused")
	}
}

func TestDeleteRemovesAFlowOfYourOwn(t *testing.T) {
	src := flowsIn(t)
	path, err := Save(src, aFlow("review"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	revealed, err := Delete(src, "review")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if revealed {
		t.Error("Delete says a built-in was revealed, but nothing ships under that name")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the file is still there after Delete: %v", err)
	}
	if _, err := Resolve(src, "review"); err == nil {
		t.Error("the name still resolves after the only flow answering to it was deleted")
	}
}

// The answer a caller cannot work out afterwards: deleting a shadow does not
// remove the flow, it puts the shipped one back, and a task written against
// that name goes on running — differently.
func TestDeletingAShadowRestoresTheBuiltin(t *testing.T) {
	src := flowsIn(t)
	writeFlow(t, src, "quick", `{"name":"quick","phases":[{"name":"mine","engine":"codex"}]}`)
	shadow, err := Resolve(src, "quick")
	if err != nil {
		t.Fatalf("Resolve the shadow: %v", err)
	}
	if shadow.Phases[0].Name != "mine" {
		t.Fatalf("the file does not shadow the built-in: %+v", shadow)
	}
	revealed, err := Delete(src, "quick")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !revealed {
		t.Error("Delete says nothing was revealed, but quick is a flow orbit ships")
	}
	got, err := Resolve(src, "quick")
	if err != nil {
		t.Fatalf("Resolve after deleting the shadow: %v", err)
	}
	if got.Phases[0].Name == shadow.Phases[0].Name {
		t.Error("the name still resolves to the shadow, want the flow orbit ships")
	}
	if got.Description == "" {
		t.Error("the flow that came back has no description; the shipped quick has one")
	}
}

func TestDeleteRefusesAFlowThatIsInsideTheBinary(t *testing.T) {
	src := flowsIn(t)
	_, err := Delete(src, "quick")
	if err == nil {
		t.Fatal("a built-in was deleted, want it refused")
	}
	if !strings.Contains(err.Error(), "built into orbit") {
		t.Errorf("the error does not say why it refused: %v", err)
	}
	if _, err := Resolve(src, "quick"); err != nil {
		t.Errorf("the built-in stopped resolving after a refused delete: %v", err)
	}
}

func TestDeleteSaysWhenThereIsNoSuchFlow(t *testing.T) {
	src := flowsIn(t)
	if _, err := Delete(src, "review"); err == nil {
		t.Error("deleting a flow nobody wrote was reported as a success")
	}
}

func TestUserPathIsTheFileTheFlowWouldBeIn(t *testing.T) {
	src := flowsIn(t)
	got, err := UserPath(src, "review")
	if err != nil {
		t.Fatalf("UserPath: %v", err)
	}
	if want := filepath.Join(src.FlowDir(), "review.json"); got != want {
		t.Errorf("UserPath = %q, want %q", got, want)
	}
	if _, err := UserPath(nil, "review"); err == nil {
		t.Error("UserPath answered for a caller with no flow directory")
	}
}
