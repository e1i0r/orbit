package task

// The suite runs in an environment of its own.
//
// A phase started by Orbit is told which task it is running and which
// workspace it may reach, in $ORBIT_TASK and $ORBIT_WORKSPACE, and every
// process under it inherits them — including a `go test` a phase runs as a
// gate. The tests here build their own temporary workspace and expect it to
// be the only one there is, so inside a run they were answered with the
// reader's real checkouts instead: "no repository called ledger in
// /Users/.../repos, which holds [orbit ...]".
//
// It made `make check` red in this package for every task Orbit ran on
// itself, for a reason that had nothing to do with the change being gated.
// A test that depends on what the machine happened to export is not a test,
// so this decides what unset means before any of them run.
//
// A test that wants one of these set does it with t.Setenv, which is scoped
// to that test and restored after it.

import (
	"os"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
)

// A run this suite starts is this binary, and it is asked to do nothing.
//
// Start spawns os.Executable() with "task start" at the front of its
// argument list, which under `go test` is the test binary rather than
// orbit. The flag package stops at the first argument that is not a flag,
// so the child reads none of what follows and runs the whole suite again —
// including the test that started it, which spawns a child of its own.
// Left alone it doubles: it took internal/cli from seconds to a ten-minute
// timeout and then went on multiplying after the run that started it had
// gone, until the machine had nothing left to give.
func TestMain(m *testing.M) {
	if len(os.Args) > 2 && os.Args[1] == "task" && os.Args[2] == "start" {
		os.Exit(0)
	}

	for _, name := range []string{"ORBIT_TASK", "ORBIT_WORKSPACE", "ORBIT_HOME"} {
		os.Unsetenv(name)
	}

	os.Exit(m.Run())
}

// fakes is the engine table a run is given: one engine, under the name the
// test flows ask for. It is here because every run in this package needs it
// and the literal is wider than the line it sits on.
func fakes(e engine.Engine) map[string]engine.Engine {
	return map[string]engine.Engine{"fake": e}
}
