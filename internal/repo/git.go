// Package repo finds repositories, reads their shape, and hands out
// throwaway worktrees. It is the only package that runs git.
package repo

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// git runs a git command in a directory and returns its trimmed stdout.
//
// Stderr is folded into the error rather than the result, because a message
// git prints while succeeding is noise, and a message it prints while failing
// is the only thing that will tell a reader what went wrong.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout

	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), msg, err)
		}

		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}
