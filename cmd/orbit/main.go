// Command orbit is a cockpit for supervising coding agents.
package main

import (
	"os"

	"github.com/e1i0r/orbit/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
