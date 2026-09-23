//go:build windows

package lowly

import (
	"os"
	"os/exec"
)

// lower is nothing here: Windows has no niceness to set from inside.
func lower() error { return nil }

// become runs the program as a child and waits for it, since there is no
// exec to replace this process with.
func become(path string, args []string) error {
	cmd := exec.Command(path, args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	return cmd.Run()
}
