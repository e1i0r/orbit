package words

// main_test.go is the guard against the whole class of bug the review's
// Critical finding named: a test that calls For (directly, or through
// Available/loadCatalog) without setting $ORBIT_HOME reads the developer's
// real ~/.orbit instead of nothing. Two tests were fixed individually
// (printer_test.go), but a fix to two call sites does not stop a third
// from being added the same way later. TestMain closes the class instead
// of the instances: before a single test in this package runs, $HOME is
// redirected to a fresh, empty temporary directory and $ORBIT_HOME is
// unset, so overlayDir's fallback — os.UserHomeDir() joined with .orbit —
// can never resolve to anything but that empty directory, no matter which
// test forgets to set $ORBIT_HOME itself.
//
// A test that wants a specific overlay still sets $ORBIT_HOME explicitly
// via t.Setenv, which takes effect for that test alone and is restored
// afterward — this only changes what "unset" falls back to.

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "words-test-home-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "words: TestMain: mkdtemp:", err)
		os.Exit(1)
	}

	os.Setenv("HOME", home)
	os.Unsetenv("ORBIT_HOME")

	code := m.Run()

	os.RemoveAll(home)
	os.Exit(code)
}
