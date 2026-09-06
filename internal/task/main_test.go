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
)

func TestMain(m *testing.M) {
	for _, name := range []string{"ORBIT_TASK", "ORBIT_WORKSPACE", "ORBIT_HOME"} {
		os.Unsetenv(name)
	}

	os.Exit(m.Run())
}
