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
// It asks for streaming JSON rather than plain text, because the session id
// and the cost of a run are reported in that stream and nowhere else. Those
// two fields have been on Result since the first plan and had never once
// been non-empty from a real run: the record said every phase cost nothing
// and none of them could be resumed. Without a session id there is no taking
// the keyboard, and taking the keyboard is the gesture the window is built
// around. The price is that stdout is no longer prose — ParseStream turns it
// back into the one sentence a reader wants.
type Claude struct{}

var _ Engine = Claude{}

// NewClaude returns the adapter.
func NewClaude() Claude { return Claude{} }

// Name identifies the engine in the record.
func (Claude) Name() string { return "claude" }

// Run invokes claude in the worktree and returns what it reported.
func (Claude) Run(ctx context.Context, req Request) (Result, error) {
	args, err := claudeArgs(req)
	if err != nil {
		// A posture this adapter cannot state is a run that does not
		// start. Widening it to whatever the binary does by default would
		// be the one failure mode the vocabulary exists to prevent, and it
		// would happen at the moment nobody is looking.
		return Result{}, fmt.Errorf("claude in %q: %w", req.Dir, err)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	// The engine's working directory is the task's worktree, which lives
	// inside the Orbit state root by design. The record that is the
	// product's whole trust model, and the credentials file the design puts
	// in the same root, are therefore reachable from here by relative path.
	// The control is not the layout, which buys nothing against a process
	// running as the same user: it is that no engine is ever handed a
	// directory permission at or above the state root. No --add-dir, and no
	// equivalent on any engine added later, at store.Root() or above it.
	// root_test.go is where that stopped being a comment and became a rule.
	cmd.Dir = req.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	out, parseErr := ParseStream(bytes.NewReader(stdout.Bytes()))
	if runErr != nil {
		if parseErr != nil {
			// The run died before claude summarised it, so there is no
			// result object and the raw stream is the only evidence of what
			// happened before it stopped. That evidence matters most on the
			// cancellation path, where it is the only account of a run
			// somebody interrupted, so it is kept — as JSON lines, which is
			// worse to read than the prose this used to hand back, and far
			// better than the nothing the alternative records.
			out = Result{Output: strings.TrimSpace(stdout.String())}
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return out, fmt.Errorf("claude in %q: %s: %w", req.Dir, msg, runErr)
		}
		return out, fmt.Errorf("claude in %q: %w", req.Dir, runErr)
	}
	if parseErr != nil {
		// The process exited zero and still said nothing this adapter
		// understands, which means the stream's shape has moved under us.
		// Reporting it is the only way that is ever noticed; a zero Result
		// would look exactly like a quiet phase.
		return Result{}, fmt.Errorf("claude in %q: %w", req.Dir, parseErr)
	}
	return out, nil
}

// claudeArgs is separate from Run so the command line can be tested without
// a claude binary present and without spending anything.
//
// It returns an error because a posture nobody defined must not be able to
// become a command line at all. That is a stronger guarantee than checking
// in Run: there is no path through this package that builds argv for a
// permission the vocabulary does not hold, so the tests that assert what
// never appears on a command line have something total to assert against.
func claudeArgs(req Request) ([]string, error) {
	perms, err := claudePermissionArgs(req.Permissions)
	if err != nil {
		return nil, err
	}
	// --verbose is not a preference. claude refuses --output-format
	// stream-json under -p without it, so it is part of the same decision
	// as the format itself.
	args := []string{"-p", req.Prompt, "--output-format", "stream-json", "--verbose"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.Resume != "" {
		args = append(args, "--resume", req.Resume)
	}
	return append(args, perms...), nil
}
