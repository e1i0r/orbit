package task

// A supervisor's engine reaching the run it starts, and stopping there.

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/repo"
)

// TestASupervisorStartsARunOnItsOwnEngine is the report of 2026-09-09: a
// reader whose claude was out of session dialled the supervisor to agy, told
// it to finish task 2900, and hit claude's session limit again — because the
// retry ran the task on the engine the task was written against.
func TestASupervisorStartsARunOnItsOwnEngine(t *testing.T) {
	t.Setenv(engine.SupervisorVar, "agy")

	tk := Task{ID: "ACME-1", Repo: repo.Repo{Name: "app", Path: "/repos/app"}}
	cmd := runCommand("/usr/local/bin/orbit", "/state", tk, "task", "")

	if !slices.Contains(cmd.Args, "-engine") || !slices.Contains(cmd.Args, "agy") {
		t.Errorf("argv = %q, want the supervisor's engine in it", cmd.Args)
	}
}

// TestAnEngineTheCallerNamedBeatsTheSupervisors. The variable is the answer
// to "nobody said", not an override: a phase pointed at one engine because
// the flow's own cannot run here is still pointed there.
func TestAnEngineTheCallerNamedBeatsTheSupervisors(t *testing.T) {
	t.Setenv(engine.SupervisorVar, "agy")

	tk := Task{ID: "ACME-1", Repo: repo.Repo{Name: "app", Path: "/repos/app"}}
	cmd := runCommand("/usr/local/bin/orbit", "/state", tk, "task", "codex")

	if slices.Contains(cmd.Args, "agy") {
		t.Errorf("argv = %q, want codex, which the caller named", cmd.Args)
	}
}

// TestWithoutASupervisorNothingChanges: every other caller — a window, a
// person at a terminal, a phase — has nothing in the variable and starts the
// run it always started.
func TestWithoutASupervisorNothingChanges(t *testing.T) {
	t.Setenv(engine.SupervisorVar, "")

	tk := Task{ID: "ACME-1", Repo: repo.Repo{Name: "app", Path: "/repos/app"}}
	cmd := runCommand("/usr/local/bin/orbit", "/state", tk, "task", "")

	if slices.Contains(cmd.Args, "-engine") {
		t.Errorf("argv = %q, want no engine flag at all", cmd.Args)
	}
}

// TestTheSupervisorsEngineTravelsOneHop. A run started by a supervisor is
// not itself a supervisor: were the name left in its environment, a task
// that started another task would start it on the supervisor's engine for
// reasons nobody could see from the board.
func TestTheSupervisorsEngineTravelsOneHop(t *testing.T) {
	t.Setenv(engine.SupervisorVar, "agy")

	tk := Task{ID: "ACME-1", Repo: repo.Repo{Name: "app", Path: "/repos/app"}}
	cmd := runCommand("/usr/local/bin/orbit", "/state", tk, "task", "")

	for _, kv := range cmd.Env {
		if strings.HasPrefix(kv, engine.SupervisorVar+"=") {
			t.Errorf("the run carries %q into its own children", kv)
		}
	}
}

// TestTheVariableIsReadAsWritten, with the spaces a shell leaves behind
// taken off, and read live rather than at start: the supervisor's engine is
// a dial a reader turns while the window is open.
func TestTheVariableIsReadAsWritten(t *testing.T) {
	t.Setenv(engine.SupervisorVar, "  agy  ")

	if got := supervisorEngine(); got != "agy" {
		t.Errorf("supervisorEngine() = %q, want %q", got, "agy")
	}

	if err := os.Unsetenv(engine.SupervisorVar); err != nil {
		t.Fatalf("unset %s: %v", engine.SupervisorVar, err)
	}

	if got := supervisorEngine(); got != "" {
		t.Errorf("supervisorEngine() = %q with nothing set, want empty", got)
	}
}
