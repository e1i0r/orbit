package ui

// main_test.go is internal/words' guard, applied to the package that calls
// words.For the most.
//
// Every test here builds a Printer, and words.For overlays whatever
// $ORBIT_HOME/lang holds on top of the embedded catalogue. Without this,
// running the suite on a machine that has an Orbit installed would render
// the goldens through that machine's overrides — which passes locally,
// fails in CI, and is the exact bug internal/words closed the same way.
//
// The pseudolocale test sets $ORBIT_HOME itself through t.Setenv, which
// takes effect for that test alone; this only decides what "unset" means.

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "ui-test-home-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ui: TestMain: mkdtemp:", err)
		os.Exit(1)
	}
	os.Setenv("HOME", home)
	os.Unsetenv("ORBIT_HOME")

	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
