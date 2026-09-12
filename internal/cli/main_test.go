package cli

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
	"fmt"
	"os"
	"testing"
)

// A run this suite starts is this binary, and it is asked to do nothing.
//
// task.Start spawns os.Executable() with "run" at the front of its argument
// list, which under `go test` is the test binary rather than orbit. The flag
// package stops at the first argument that is not a flag, so the child would
// read none of what follows and run the whole suite again — including the
// test that started it, which would spawn a child of its own. That is what
// took this package from seconds to a ten-minute timeout, with a tree of
// itself still running underneath.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		os.Exit(0)
	}

	for _, name := range []string{"ORBIT_TASK", "ORBIT_WORKSPACE"} {
		os.Unsetenv(name)
	}

	// The suite's own state root. Unsetting ORBIT_HOME falls back to the
	// reader's real ~/.orbit, so a test that names no home reads and
	// writes where the developer works — and fails when that root was
	// written by a newer orbit, for a reason with nothing to do with the
	// change under test. A test that wants a home of its own still sets
	// it with t.Setenv, which wins over this for that test.
	root, err := os.MkdirTemp("", "orbit-cli-test")
	if err != nil {
		fmt.Fprintln(os.Stderr, "make a state root for the suite:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(root) //nolint:errcheck // the suite is over; there is nobody left to tell
	os.Setenv("ORBIT_HOME", root)

	os.Exit(m.Run())
}
