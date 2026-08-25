package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/e1i0r/orbit/internal/cli"
)

func TestMainRunInvocation(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Run([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Errorf("expected exit code 0 on --help, got %d", code)
	}
}

// TestMainCallsExitWithRunCode drives main itself, with exit swapped out for
// a closure that records the code instead of ending the test binary.
func TestMainCallsExitWithRunCode(t *testing.T) {
	oldArgs, oldExit := os.Args, exit
	defer func() { os.Args, exit = oldArgs, oldExit }()

	var code int
	exit = func(c int) { code = c }
	os.Args = []string{"orbit", "--help"}

	main()

	if code != 0 {
		t.Errorf("main() exit code = %d, want 0", code)
	}
}
