// Command orbit is a cockpit for supervising coding agents.
package main

import (
	"os"

	"github.com/e1i0r/orbit/internal/cli"
)

// exit is os.Exit, indirected so a test can call main and read the code back
// instead of ending the test binary with it.
var exit = os.Exit

func main() {
	raiseFileLimit()
	exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
