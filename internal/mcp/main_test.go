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

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		os.Exit(0)
	}

	os.Exit(m.Run())
}
