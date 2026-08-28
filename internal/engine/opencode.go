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
func (OpenCode) Models() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "deepseek-r1", Label: "deepseek-r1"},
		{ID: "qwen-2.5-coder", Label: "qwen-2.5-coder"},
		{ID: "llama-3.3-70b", Label: "llama-3.3-70b"},
		{ID: "claude-3-5-sonnet", Label: "claude-3-5-sonnet"},
		{ID: "gemini-2.5-flash", Label: "gemini-2.5-flash"},
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
