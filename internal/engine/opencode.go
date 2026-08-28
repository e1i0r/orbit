package engine

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// OpenCode runs the opencode command line in headless mode.
type OpenCode struct{}

var _ Engine = OpenCode{}

// NewOpenCode returns the OpenCode CLI adapter.
func NewOpenCode() OpenCode { return OpenCode{} }

// Name identifies the engine in the record.
func (OpenCode) Name() string { return "opencode" }

// CanResume reports whether opencode can carry on a previous session.
func (OpenCode) CanResume() bool { return true }

// Models returns the models opencode supports.
//
// The IDs are what `opencode models` prints and what --model takes:
// provider-qualified, and rejected by opencode without the qualifier. The
// list this replaced carried five bare names — deepseek-r1, qwen-2.5-coder,
// llama-3.3-70b, claude-3-5-sonnet, gemini-2.5-flash — none of which
// opencode has ever accepted, and internal/task checks a phase's model
// against this list before running anything, so the only models a phase
// could name on opencode were five that could not work.
//
// Half of them are free, and they say so in their own names rather than in
// a mark this program invents: the label is the ID without the provider, so
// what a reader picks from and what opencode is told are the same string
// with nothing in between deciding.
//
// It is a written-down copy of a catalogue that moves, which is a cost taken
// deliberately: reading it at run time means shelling out to opencode before
// a dial can be drawn, on a machine where opencode may not be installed at
// all. Refresh it with `opencode models`.
func (OpenCode) Models() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		// Paid.
		{ID: "opencode/claude-opus-5", Label: "claude-opus-5"},
		{ID: "opencode/claude-sonnet-5", Label: "claude-sonnet-5"},
		{ID: "opencode/gpt-5.3-codex", Label: "gpt-5.3-codex"},
		{ID: "opencode/gemini-3.1-pro", Label: "gemini-3.1-pro"},
		{ID: "opencode/grok-4.6", Label: "grok-4.6"},
		// Free.
		{ID: "opencode/nemotron-3-ultra-free", Label: "nemotron-3-ultra-free"},
		{ID: "opencode/nemotron-3.5-lightning-free", Label: "nemotron-3.5-lightning-free"},
		{ID: "opencode/mimo-v2.5-free", Label: "mimo-v2.5-free"},
		{ID: "opencode/ling-3.0-flash-fin-free", Label: "ling-3.0-flash-fin-free"},
		{ID: "opencode/hy3-free", Label: "hy3-free"},
		{ID: "opencode/muse-spark-1.2-contributor-free", Label: "muse-spark-1.2-contributor-free"},
	}
}

// Efforts returns the effort choices opencode supports.
func (OpenCode) Efforts() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "low", Label: "low"},
		{ID: "medium", Label: "medium"},
		{ID: "high", Label: "high"},
	}
}

// CanThink reports whether opencode supports thinking mode.
func (OpenCode) CanThink() bool { return true }

// Run invokes opencode in the worktree and returns what it reported.
func (OpenCode) Run(ctx context.Context, req Request) (Result, error) {
	args, err := openCodeArgs(req)
	if err != nil {
		return Result{}, fmt.Errorf("opencode in %q: %w", req.Dir, err)
	}

	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = req.Dir
	stdout := &boundedBuffer{max: maxStream}
	stderr := &boundedBuffer{max: maxStderr}
	pr, pw := io.Pipe()
	cmd.Stdout = io.MultiWriter(stdout, pw)
	cmd.Stderr = stderr

	var (
		streamResult Result
		parseErr     error
	)

	done := make(chan struct{})
	go func() {
		defer close(done)

		streamResult, parseErr = ParseStreamWithCallback(pr, req.OnEvent)
		if closeErr := pr.Close(); closeErr != nil && parseErr == nil {
			parseErr = closeErr
		}
	}()

	runErr := cmd.Run()
	if closeErr := pw.Close(); closeErr != nil && runErr == nil {
		runErr = closeErr
	}

	<-done

	out := streamResult

	if runErr != nil {
		if parseErr != nil {
			if streamResult.Output == "" {
				streamResult.Output = strings.TrimSpace(stdout.String())
			}

			out = streamResult
		}

		out.Output = noteDropped(out.Output, stdout.dropped)
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return out, fmt.Errorf("opencode in %q: %s: %w", req.Dir, msg, runErr)
		}

		return out, fmt.Errorf("opencode in %q: %w", req.Dir, runErr)
	}

	if parseErr != nil {
		raw := strings.TrimSpace(stdout.String())
		if raw != "" {
			return Result{Output: noteDropped(raw, stdout.dropped)}, nil
		}

		return Result{}, fmt.Errorf("opencode in %q: %w", req.Dir, parseErr)
	}

	out.Output = noteDropped(out.Output, stdout.dropped)

	return out, nil
}

// openCodeArgs builds the command line for opencode run.
func openCodeArgs(req Request) ([]string, error) {
	if err := Permitted(req.Permissions); err != nil {
		return nil, err
	}

	args := []string{"run"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	if req.Effort != "" {
		args = append(args, "--effort", req.Effort)
	}

	if req.Resume != "" {
		args = append(args, "--session", req.Resume)
	}

	args = append(args, req.Prompt)

	return args, nil
}
