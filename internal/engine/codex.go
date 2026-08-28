package engine

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Codex runs the codex command line in headless execution mode.
type Codex struct{}

var _ Engine = Codex{}

// NewCodex returns the Codex CLI adapter.
func NewCodex() Codex { return Codex{} }

// Name identifies the engine in the record.
func (Codex) Name() string { return "codex" }

// CanResume reports whether codex can resume a previous session.
func (Codex) CanResume() bool { return true }

// Models returns the models codex supports.
func (Codex) Models() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "o3-mini", Label: "o3-mini"},
		{ID: "o3", Label: "o3"},
		{ID: "o1", Label: "o1"},
		{ID: "gpt-4o", Label: "gpt-4o"},
		{ID: "gpt-4.5-preview", Label: "gpt-4.5-preview"},
	}
}

// Efforts returns the reasoning effort levels codex supports.
func (Codex) Efforts() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "low", Label: "low"},
		{ID: "medium", Label: "medium"},
		{ID: "high", Label: "high"},
	}
}

// CanThink reports whether codex supports reasoning models.
func (Codex) CanThink() bool { return true }

// Run invokes the codex CLI in the worktree and returns what it reported.
func (Codex) Run(ctx context.Context, req Request) (Result, error) {
	args, err := codexArgs(req)
	if err != nil {
		return Result{}, fmt.Errorf("codex in %q: %w", req.Dir, err)
	}
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Dir = req.Dir
	stdout := &boundedBuffer{max: maxStream}
	stderr := &boundedBuffer{max: maxStderr}
	pr, pw := io.Pipe()
	cmd.Stdout = io.MultiWriter(stdout, pw)
	cmd.Stderr = stderr

	var streamResult Result
	var parseErr error
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
			return out, fmt.Errorf("codex in %q: %s: %w", req.Dir, msg, runErr)
		}
		return out, fmt.Errorf("codex in %q: %w", req.Dir, runErr)
	}
	if parseErr != nil {
		// Plain text fallback if not JSON stream
		raw := strings.TrimSpace(stdout.String())
		if raw != "" {
			return Result{Output: noteDropped(raw, stdout.dropped)}, nil
		}
		return Result{}, fmt.Errorf("codex in %q: %w", req.Dir, parseErr)
	}
	out.Output = noteDropped(out.Output, stdout.dropped)
	return out, nil
}

// codexArgs builds the command line for codex exec.
func codexArgs(req Request) ([]string, error) {
	if err := Permitted(req.Permissions); err != nil {
		return nil, err
	}
	args := []string{"exec"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.Effort != "" {
		args = append(args, "--effort", req.Effort)
	}
	if req.Resume != "" {
		args = append(args, "--resume", req.Resume)
	}
	args = append(args, req.Prompt)
	return args, nil
}
