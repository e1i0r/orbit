// Package lowly starts what orbit runs on its own at the lowest priority the
// system has, so that the machine stays the reader's while it works.
//
// A run's engine and gates, and a session opened on a task, run the
// repository's own tools, and some of them take every core there is:
// golangci-lint at 1000% froze Elio's whole machine, orbit included, while
// a session opened from the cockpit ran make check. Priority does not make
// the work slower on an idle machine. It decides who is served first when
// two things want the CPU at once, and the window and the reader's own
// programs should always be first.
//
// The mechanism is orbit starting itself as a step in front of the work:
// `orbit __lowered -- <program> <args>` lowers its own priority and then
// becomes the program, keeping the pid, the process group and the terminal.
// Everything the program starts inherits the priority, which is the point:
// the engine is lowered and so is every make check it runs.
package lowly

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// Arg is the first argument that makes orbit the step in front of a
// program. It is not a command a reader types, so it is not in the table.
const Arg = "__lowered"

// Niceness is how far the priority is lowered: all the way.
const Niceness = 19

// Args is the argument list that runs name with args behind the step, for
// orbit's own binary to be started with.
func Args(name string, args ...string) []string {
	return append([]string{Arg, "--", name}, args...)
}

// Command is name run with args behind the step, started from this
// process's own binary. A binary that cannot find itself runs the program
// as it would have without the step.
func Command(name string, args ...string) *exec.Cmd {
	self, err := os.Executable()
	if err != nil {
		return exec.Command(name, args...)
	}

	return exec.Command(self, Args(name, args...)...)
}

// Behind is the program a command runs and its arguments, looking through
// the step when there is one: what a reader of the command means by "what
// does this run".
func Behind(cmd *exec.Cmd) []string {
	if len(cmd.Args) >= 3 && cmd.Args[1] == Arg && cmd.Args[2] == "--" {
		return cmd.Args[3:]
	}

	return cmd.Args
}

// Exec is the step itself: lower this process, then become args[0] with the
// rest as its arguments. It returns only when it could not become the
// program.
func Exec(args []string) error {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	if len(args) == 0 {
		return errors.New("no program to run behind the lowered step")
	}

	// A priority that could not be lowered is not a reason not to run the
	// program: it runs as it would have without this step, and says so.
	if err := lower(); err != nil {
		fmt.Fprintf(os.Stderr, "orbit: the priority could not be lowered: %v\n", err)
	}

	path, err := exec.LookPath(args[0])
	if err != nil {
		return err
	}

	return become(path, args)
}
