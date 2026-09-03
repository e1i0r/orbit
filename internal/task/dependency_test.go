package task

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// addingEngine writes a manifest into the worktree, which is what an engine
// reaching for a library does.
type addingEngine struct {
	*engine.Fake

	file string
	body string
}

func (e addingEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	out, err := e.Fake.Run(ctx, req)
	if err != nil {
		return out, err
	}

	return out, os.WriteFile(filepath.Join(req.Dir, e.file), []byte(e.body), 0o600)
}

func dependencyFlow(allow bool) flow.Flow {
	return flow.Flow{Name: "task", AllowNewDependencies: allow, Phases: []flow.Phase{
		{Name: "implement", Engine: "fake"},
		{Name: "review", Engine: "fake"},
	}}
}

const goMod = "module example.com/x\n\ngo 1.23\n\nrequire github.com/spf13/cobra v1.8.0\n"

// TestARunStopsWhenSomethingNewIsDependedOn. What a project carries — its
// licences, its maintenance, its security updates — is not the agent's
// decision, and this is the one gate that cannot be answered by the thing
// that tripped it.
func TestARunStopsWhenSomethingNewIsDependedOn(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-15", "a task that reaches for a library", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := addingEngine{Fake: engine.NewFake("added cobra"), file: "go.mod", body: goMod}

	err = Run(context.Background(), s, tk, dependencyFlow(false), map[string]engine.Engine{"fake": eng}, nil)
	if err == nil {
		t.Fatal("Run: want an error when a new dependency appears")
	}

	if len(eng.Calls) != 1 {
		t.Errorf("the engine ran %d times, want 1 — the phase after a dependency waits on the answer", len(eng.Calls))
	}

	events, evErr := Events(s, tk)
	if evErr != nil {
		t.Fatalf("Events: %v", evErr)
	}

	last := events[len(events)-1]
	if last.Kind != record.TaskNewDependency {
		t.Fatalf("the record ends in %q, want task.new_dependency: %v", last.Kind, kindsOf(events))
	}

	if last.Data["names"] != "github.com/spf13/cobra" {
		t.Errorf("task.new_dependency names %q, want the library that was added", last.Data["names"])
	}
}

// TestAFlowMayAllowNewDependencies. A flow that has said so is not asked
// again on every phase.
func TestAFlowMayAllowNewDependencies(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-16", "a task allowed to add libraries", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := addingEngine{Fake: engine.NewFake("added cobra"), file: "go.mod", body: goMod}
	if err := Run(context.Background(), s, tk, dependencyFlow(true), map[string]engine.Engine{"fake": eng}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(eng.Calls) != 2 {
		t.Errorf("the engine ran %d times, want both phases", len(eng.Calls))
	}
}

// TestApprovingADependencyLetsTheNextRunPast. Approval is per name and
// survives the run it was given in — the same library added again by the
// next attempt is a question that was already answered.
func TestApprovingADependencyLetsTheNextRunPast(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-17", "a library somebody said yes to", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := dependencyFlow(false)
	eng := addingEngine{Fake: engine.NewFake("added cobra"), file: "go.mod", body: goMod}
	engines := map[string]engine.Engine{"fake": eng}

	if err := Run(context.Background(), s, tk, f, engines, nil); err == nil {
		t.Fatal("Run: want the first run to stop at the gate")
	}

	pending := Pending(s, tk, f)
	if len(pending) != 1 || pending[0] != "github.com/spf13/cobra" {
		t.Fatalf("Pending = %v, want the library the run stopped on", pending)
	}

	if err := Approve(s, tk, pending); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	if err := Run(context.Background(), s, tk, f, engines, nil); err != nil {
		t.Fatalf("the run after approval: %v", err)
	}
}

// TestApprovingNothingIsRefused. A command that answers "approved" about a
// task with nothing pending has told a reader something untrue.
func TestApprovingNothingIsRefused(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-18", "nothing was added", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Approve(s, tk, nil); err == nil {
		t.Error("Approve with nothing pending answered yes")
	}
}

// TestAManifestLineIsReadForTheNameAndNothingElse walks the formats a
// dependency arrives in. What is asked of each is the name; whether it
// resolves is the package manager's business.
func TestAManifestLineIsReadForTheNameAndNothingElse(t *testing.T) {
	for _, tc := range []struct {
		manifest string
		line     string
		want     string
	}{
		{"go.mod", "\tgithub.com/spf13/cobra v1.8.0", "github.com/spf13/cobra"},
		{"go.mod", "require (", ""},
		{"package.json", `    "react": "^19.0.0",`, "react"},
		{"package.json", `    "@types/node": "22.0.0",`, "@types/node"},
		{"package.json", `  "dependencies": {`, ""},
		{"requirements.txt", "fastapi==0.111.0", "fastapi"},
		{"requirements.txt", "# a comment", ""},
		{"Gemfile", `gem "rails", "~> 7.1"`, "rails"},
		{"Cargo.toml", `serde = "1.0"`, "serde"},
		{"build.gradle", `    implementation "org.slf4j:slf4j-api:2.0.13"`, "org.slf4j:slf4j-api"},
	} {
		pattern, ok := manifests[tc.manifest]
		if !ok {
			t.Fatalf("%s is not a manifest this knows", tc.manifest)
		}

		got := strings.Join(named(pattern, []string{tc.line}), ",")
		if got != tc.want {
			t.Errorf("%s: %q reads as %q, want %q", tc.manifest, tc.line, got, tc.want)
		}
	}
}

// TestEveryManifestPatternKeepsItsName. A pattern with no capturing group
// would panic on the first line it matched, in a run, at night.
func TestEveryManifestPatternKeepsItsName(t *testing.T) {
	for name, pattern := range manifests {
		if pattern.NumSubexp() < 1 {
			t.Errorf("the pattern for %s captures nothing", name)
		}

		if _, err := regexp.Compile(pattern.String()); err != nil {
			t.Errorf("the pattern for %s does not compile: %v", name, err)
		}
	}
}
