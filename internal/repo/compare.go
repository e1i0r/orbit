package repo

// Running the same checks on both sides of the change.
//
// The diff says what the work wrote and the impact pane says what usually
// moves with it. Neither of them can say whether it still works — and the
// cheapest way to know is not to invent inputs and call functions, which
// needs a harness per language and a sandbox nothing here has. It is to run
// the commands the repository already trusts, twice: once on the branch the
// work was cut from, once on what the work produced, and say which of them
// changed their answer.
//
// A check that passes on both sides is not news. A check that passed before
// and fails now is a regression somebody has to look at, and one that failed
// before and passes now is the work. Both are facts with an exit code
// behind them, which is the whole difference between this and a claim.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// checkDeadline is how long one command is given. A test suite is minutes
// and not seconds; past this it is a suite that hung, and saying so is worth
// more than waiting for it.
const checkDeadline = 10 * time.Minute

// keptOutput is how much of what a command printed is carried back. Enough
// to see which assertion failed, and not the whole of a suite's chatter.
const keptOutput = 4 << 10

// Check is one command the repository already trusts, by the name the flow
// gave it.
type Check struct {
	Name    string
	Command string
}

// Ran is what a command answered on one side.
type Ran struct {
	Exit int
	Out  string
	// Failed is why the command could not be run at all — a shell that is
	// not there, a directory that is gone. It is not a check that failed:
	// a run that never happened has no exit code to compare.
	Failed error
}

// Passed reports whether the command exited zero and actually ran.
func (r Ran) Passed() bool { return r.Failed == nil && r.Exit == 0 }

// Divergence is one check, run on both sides.
type Divergence struct {
	Check
	Base Ran
	Now  Ran
}

// Same reports whether both sides answered alike. It compares the verdict
// and not the output: a suite that prints a timestamp is not a divergence.
func (d Divergence) Same() bool {
	if d.Base.Failed != nil || d.Now.Failed != nil {
		return d.Base.Failed == nil && d.Now.Failed == nil
	}

	return d.Base.Passed() == d.Now.Passed()
}

// Broke reports the case somebody has to look at: it passed before and does
// not now.
func (d Divergence) Broke() bool { return d.Base.Passed() && !d.Now.Passed() }

// Fixed reports the other one: it failed before and passes now.
func (d Divergence) Fixed() bool { return !d.Base.Passed() && d.Now.Passed() }

// Compare runs every check on the branch the work was cut from and on the
// work, and answers what each side said.
//
// The base is checked out into a directory of its own, thrown away
// afterwards, and detached: it is read, never committed to, and a branch
// left behind in somebody's repository is bookkeeping they did not ask for.
func (r Repo) Compare(wtDir string, checks []Check) ([]Divergence, error) {
	if len(checks) == 0 {
		return nil, nil
	}

	if r.Base == "" {
		return nil, fmt.Errorf("%q is not on a branch, so there is nothing to compare against", r.Path)
	}

	dir, err := os.MkdirTemp("", "orbit-base-")
	if err != nil {
		return nil, fmt.Errorf("make somewhere to check the base out: %w", err)
	}

	defer func() { _ = os.RemoveAll(dir) }() //nolint:errcheck // a temporary directory the OS will take back

	if err := r.detached(dir); err != nil {
		return nil, err
	}

	defer func() { _ = r.RemoveWorktree(dir) }() //nolint:errcheck // the checkout is going either way

	out := make([]Divergence, 0, len(checks))
	for _, c := range checks {
		out = append(out, Divergence{Check: c, Base: runCheck(dir, c.Command), Now: runCheck(wtDir, c.Command)})
	}

	return out, nil
}

// detached checks the base out with no branch of its own.
func (r Repo) detached(dir string) error {
	if _, err := git(r.Path, "worktree", "add", "--detach", dir, r.Base); err != nil {
		return fmt.Errorf("check %s out at %q: %w", r.Base, dir, err)
	}

	return nil
}

// runCheck runs one command in one directory and reports what it answered.
//
// Through a shell, because that is what the command was written for: the
// checks a flow carries are shell lines with pipes and redirections in them,
// and splitting one on spaces would run a program called `go test ./...`.
func runCheck(dir, command string) Ran {
	ctx, cancel := context.WithTimeout(context.Background(), checkDeadline)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.Env = environ()

	out, err := cmd.CombinedOutput()
	got := Ran{Out: tail(string(out))}

	var exited *exec.ExitError

	switch {
	case err == nil:
		return got
	case errors.As(err, &exited):
		got.Exit = exited.ExitCode()
		return got
	default:
		got.Failed = err
		return got
	}
}

// tail is the end of what a command printed, which is where a test runner
// says what failed.
func tail(out string) string {
	out = strings.TrimSpace(out)
	if len(out) <= keptOutput {
		return out
	}

	return "…" + out[len(out)-keptOutput:]
}
