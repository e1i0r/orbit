package engine

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Claude runs the claude command line in headless mode.
//
// This adapter captures stdout and nothing else. Session identifiers and cost
// are reported by claude in its streaming JSON output, and reading that comes
// with the gestures — ask, take, hand — in a later plan. Leaving the fields
// empty is honest; guessing at them would not be.
type Claude struct{}

var _ Engine = Claude{}

// NewClaude returns the adapter.
func NewClaude() Claude { return Claude{} }

// Name identifies the engine in the record.
func (Claude) Name() string { return "claude" }

// Run invokes claude in the worktree and returns what it printed.
func (Claude) Run(ctx context.Context, req Request) (Result, error) {
	cmd := exec.CommandContext(ctx, "claude", claudeArgs(req)...)
	// The engine's working directory is the task's worktree, which lives
	// inside the Orbit state root by design. The record that is the
	// product's whole trust model, and the credentials file the design puts
	// in the same root, are therefore reachable from here by relative path.
	// The control is not the layout, which buys nothing against a process
	// running as the same user: it is that no engine is ever handed a
	// directory permission at or above the state root. No --add-dir, and no
	// equivalent on any engine added later, at store.Root() or above it.
	cmd.Dir = req.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	// Trimmed once, for both paths. Whether the caller gets a trailing
	// newline should not depend on whether the run failed.
	out := Result{Output: strings.TrimSpace(stdout.String())}
	if runErr != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return out, fmt.Errorf("claude in %q: %s: %w", req.Dir, msg, runErr)
		}
		return out, fmt.Errorf("claude in %q: %w", req.Dir, runErr)
	}
	return out, nil
}

// claudeArgs is separate from Run so the command line can be tested without
// a claude binary present and without spending anything.
func claudeArgs(req Request) []string {
	args := []string{"-p", req.Prompt}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	return args
}
