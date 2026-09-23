package mcp

// A run this suite starts is this binary, and it is asked to do nothing.
//
// task.Start spawns os.Executable() with "run" at the front of its argument
// list, which under `go test` is the test binary rather than orbit. The flag
// package stops at the first argument that is not a flag, so the child reads
// none of what follows and runs the whole suite again — including the test
// that started it, which spawns a child of its own. Left alone it doubles,
// and it does not stop when the run that started it is over: it took
// internal/cli from seconds to a ten-minute timeout and then went on
// multiplying until the machine had nothing left to give.
//
// The suite also runs in an environment of its own. A phase started by
// Orbit is told which task it is running and which workspace it may reach,
// in $ORBIT_TASK and $ORBIT_WORKSPACE, and every process under it inherits
// them — including a `go test` a phase runs as a gate. The tests here build
// their own temporary workspace and expect it to be the only one there is,
// so inside a run they were answered with the reader's real checkouts
// instead: "no repository called ledger in /Users/.../repos, which holds
// [orbit ...]". internal/task and internal/cli close the same hole this
// way. A test that wants one of these set does it with t.Setenv, which is
// scoped to that test and restored after it.

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		os.Exit(0)
	}

	for _, name := range []string{"ORBIT_TASK", "ORBIT_WORKSPACE"} {
		os.Unsetenv(name)
	}

	os.Exit(m.Run())
}
