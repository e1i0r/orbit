package verb

// The suite runs in an environment of its own.
//
// A phase started by Orbit is told which task it is running and which
// workspace it may reach, in $ORBIT_TASK and $ORBIT_WORKSPACE, and every
// process under it inherits them — including a `go test` a phase runs as a
// gate. The tests here build their own temporary workspace and expect it to
// be the only one there is, so inside a run they were answered with the
// reader's real checkouts instead: "no repository called ledger in
// /Users/.../repos, which holds [orbit ...]". It made `make check` red in
// this package for every task Orbit ran on itself, for a reason that had
// nothing to do with the change being gated — the same hole internal/task
// and internal/cli closed this way.
//
// A test that wants one of these set does it with t.Setenv, which is scoped
// to that test and restored after it.

import (
	"os"
	"testing"
)

// A run this suite starts is this binary, and it is asked to do nothing.
//
// "task start" is the command line task.Start spawns (task/start.go), with
// os.Executable() at the front of it — which under `go test` is the test
// binary rather than orbit. The flag package stops at the first argument
// that is not a flag, so the child reads none of what follows and runs the
// whole suite again, including the test that started it. internal/task and
// internal/cli both carry this guard for what it cost them: a tree of test
// binaries that went on doubling after the run that started it had gone.
func TestMain(m *testing.M) {
	if len(os.Args) > 2 && os.Args[1] == "task" && os.Args[2] == "start" {
		os.Exit(0)
	}

	for _, name := range []string{"ORBIT_TASK", "ORBIT_WORKSPACE"} {
		os.Unsetenv(name)
	}

	os.Exit(m.Run())
}
