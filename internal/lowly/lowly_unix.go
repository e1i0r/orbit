//go:build !windows

package lowly

import (
	"os"
	"syscall"
)

// lower sets this process to the lowest priority. Children inherit it.
func lower() error { return syscall.Setpriority(syscall.PRIO_PROCESS, 0, Niceness) }

// become replaces this process with the program: same pid, same process
// group, same terminal, so whatever started it sees no difference.
func become(path string, args []string) error { return syscall.Exec(path, args, os.Environ()) }
