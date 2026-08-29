// Package repo finds repositories, reads their shape, and hands out
// throwaway worktrees. It is the only package that runs git, and the only
// one that runs gh.
package repo

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// deadline is the backstop on any one command this package runs.
	//
	// It is not a budget for the work. A first push of a large branch over a
	// bad connection is minutes, and a deadline that cut that off would
	// break a delivery that was going to succeed. It is here for the other
	// case, where the far end has stopped answering and nothing is ever
	// going to arrive: a fetch against a host that dropped the connection, a
	// push waiting on a credential helper that will not return. Without it
	// those wait for as long as the window is open, and the task they belong
	// to says "delivering" for exactly that long.
	deadline = 10 * time.Minute
	// commandChars is how much of a command line is quoted back in an error.
	// git's arguments are short and all of them fit; `gh pr create --body`
	// carries the whole body of a pull request, and an error that reprints it
	// buries the sentence that says what went wrong.
	commandChars = 120
)

// overrides are the environment variables that decide, ahead of cmd.Dir,
// which repository git is talking to.
//
// cmd.Dir is not the whole answer to "which repository". git reads these
// first, so a caller whose own environment carries one — which is any
// process started from a git hook, and any shell where somebody has been
// working on a bare clone — turns every command in this package into a
// command against a repository nobody named. The worktree the run was given
// would be left untouched and another repository would be committed to.
var overrides = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_INDEX_FILE",
	"GIT_COMMON_DIR",
	"GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_NAMESPACE",
}

// environ is the caller's environment without the variables that would point
// git somewhere else, and with prompting turned off.
//
// git does not ask for a password on stdin: it opens the terminal directly,
// which for a subprocess of the window is a question nobody will ever see
// waiting on an answer nobody can type. It is the likeliest way one of these
// commands hangs and the one the deadline handles worst — ten minutes of a
// task saying "pushing" before it says what a missing credential would have
// said at once.
func environ() []string {
	env := os.Environ()
	kept := make([]string, 0, len(env)+1)

	for _, kv := range env {
		if name, _, found := strings.Cut(kv, "="); !found || !slices.Contains(overrides, name) {
			kept = append(kept, kv)
		}
	}

	return append(kept, "GIT_TERMINAL_PROMPT=0")
}

// run executes one program in a directory and returns its trimmed stdout.
//
// Stderr is folded into the error rather than the result, because a message
// a program prints while succeeding is noise, and a message it prints while
// failing is the only thing that will tell a reader what went wrong.
//
// Keeping the two streams apart is why this is one function and not two. gh
// used to be run with CombinedOutput and its answer handed back as it stood,
// so the pull request URL a reader was given was whatever gh had written to
// both streams at once. One upgrade notice on stderr and "Pull Request
// created: %s" is a link to nothing — the same fault CreatePR's own comment
// says it already fixed once, left in place two lines above it.
func run(dir, program string, args ...string) (string, error) {
	return runWithin(deadline, dir, program, args...)
}

// runWithin is run with the deadline named, so that a test can prove what
// happens at it without waiting ten minutes to find out.
func runWithin(within time.Duration, dir, program string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), within)
	defer cancel()

	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = dir
	cmd.Env = environ()

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return strings.TrimSpace(stdout.String()), nil
	}

	line := clipCommand(program + " " + strings.Join(args, " "))
	if ctx.Err() != nil {
		// What Run answers when the context ended the process is "signal:
		// killed", which is a sentence about a signal rather than about a
		// command that never finished.
		return "", fmt.Errorf("%s: gave up waiting after %s", line, within)
	}

	if msg := strings.TrimSpace(stderr.String()); msg != "" {
		return "", fmt.Errorf("%s: %s: %w", line, msg, err)
	}

	return "", fmt.Errorf("%s: %w", line, err)
}

// clipCommand shortens a command line to what an error can carry.
func clipCommand(line string) string {
	if utf8.RuneCountInString(line) <= commandChars {
		return line
	}

	return string([]rune(line)[:commandChars]) + "…"
}

// git runs a git command in a directory and returns its trimmed stdout.
func git(dir string, args ...string) (string, error) {
	return run(dir, "git", args...)
}

// gh runs a GitHub CLI command in a directory and returns its trimmed
// stdout. It goes through the same runner as git because everything said up
// there is as true of one as it is of the other.
func gh(dir string, args ...string) (string, error) {
	return run(dir, "gh", args...)
}
