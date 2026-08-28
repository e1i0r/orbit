package engine

import (
	"context"
	"fmt"
	"slices"
)

// Codex runs the codex command line in headless execution mode.
//
// Every flag below was read off `codex exec --help` and then run against the
// binary, because the three this replaced had not been: --effort and
// --resume were invented, and codex answers both with "unexpected argument"
// before it starts. A phase that named an effort, and every attempt to carry
// a session on, died at argv parse without ever reaching a model.
type Codex struct{}

var _ Engine = Codex{}

// NewCodex returns the Codex CLI adapter.
func NewCodex() Codex { return Codex{} }

// Name identifies the engine in the record.
func (Codex) Name() string { return "codex" }

// CanResume reports whether codex can resume a previous session.
func (Codex) CanResume() bool { return true }

// Models returns the models codex supports.
//
// Empty on purpose, which leaves the dial at "default" and lets codex use
// whatever ~/.codex/config.toml names. The list this replaced was o3-mini,
// o3, o1, gpt-4o and gpt-4.5-preview — five names from before codex shipped,
// and internal/task checks a phase's model against this list before running
// anything, so the only models a phase could name on codex were five the
// account behind the CLI answers "not supported" to.
//
// A written-down catalogue is what went stale, and codex's own catalogue is
// a function of the account rather than of the binary: the same `codex exec`
// on two logins takes different names. There is no list this package can
// hold that is true for both, so it holds none and asks for nothing.
func (Codex) Models() []Choice {
	return []Choice{{ID: "", Label: "default"}}
}

// Efforts returns the reasoning effort levels codex supports.
//
// Three, and they are the intersection of two gates rather than of one.
// codex parses the value against its own enum — minimal, low, medium, high —
// and then the model behind it answers with its own vocabulary, which for
// gpt-5.6 is none, low, medium, high, xhigh and max. minimal passes the
// first gate and fails the second; xhigh and max pass the second and fail
// the first. What survives both is what is offered here.
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

// spec is how codex is driven.
func (Codex) spec() spec {
	return spec{
		name:  "codex",
		dirs:  []string{".codex/bin", ".local/bin", "bin"},
		args:  codexArgs,
		parse: ParseCodexStream,
	}
}

// Locate is where this machine keeps codex.
func (c Codex) Locate() (string, error) { return c.spec().locate() }

// Run invokes the codex CLI in the worktree and returns what it reported.
func (c Codex) Run(ctx context.Context, req Request) (Result, error) {
	return c.spec().run(ctx, req)
}

// codexArgs builds the command line for codex exec.
//
// --json is not optional here. codex prints prose by default, and this
// adapter used to pipe that prose into claude's stream parser, so every
// codex run fell through to the plain-text path with no session id and no
// token count. The event stream is where codex says which thread it started,
// and a thread id is the only thing a later resume can be built from.
//
// Resume is a subcommand and not a flag — `codex exec resume <id>` — and the
// options belong to exec, before it: codex answers `exec resume --sandbox`
// with "unexpected argument", so the order below is the order that parses.
func codexArgs(req Request) ([]string, error) {
	sandbox, err := codexPermissionArgs(req.Permissions)
	if err != nil {
		return nil, err
	}

	args := append([]string{"exec", "--json"}, sandbox...)

	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	if req.Effort != "" {
		// Effort is a config override rather than a flag of its own.
		args = append(args, "-c", "model_reasoning_effort="+req.Effort)
	}

	if req.Resume != "" {
		args = append(args, "resume", req.Resume)
	}

	return append(args, req.Prompt), nil
}

// codexPermissionArgs turns a posture into codex's sandbox policy.
//
// codex states a posture in one word — read-only, workspace-write or
// danger-full-access — plus one config key for whether a writing sandbox
// also reaches the network. This adapter emitted none of them: it validated
// the permission names and then built an argv that said nothing about them,
// which is precisely the silent widening the head of permission.go says must
// never happen. Every codex phase ran at whatever the binary's own default
// was, and the record said it had a posture.
//
// The mapping, and what each line costs:
//
// repo becomes workspace-write, and network is the config key that decides
// whether that sandbox may also reach out. Both are named on the argv, false
// included, for the reason claudePermissionArgs names "default" instead of
// omitting the flag: a posture stated out loud cannot be moved later by a
// change to somebody's config.toml.
//
// network without repo is refused. codex has no sandbox that reaches the
// network without also granting writes, so honouring it would mean handing
// out write access nobody asked for, and the rule this package keeps is that
// a posture it cannot state is a run that does not start.
//
// Everything narrower — a read posture, and one that asks for nothing —
// becomes read-only, which is codex's floor. That floor is wider than
// nothing: a phase that asked for no permissions at all still gets to read
// the filesystem, because codex has no way to say otherwise. It is written
// here rather than left to be discovered, and it is not a refusal, since
// there is nothing narrower to fall back to.
func codexPermissionArgs(names []string) ([]string, error) {
	if err := Permitted(names); err != nil {
		return nil, err
	}

	repo := slices.Contains(names, PermissionRepo)
	network := slices.Contains(names, PermissionNetwork)

	if network && !repo {
		return nil, fmt.Errorf(
			"codex cannot grant %s without also granting %s: no sandbox it has reaches the network read-only",
			PermissionNetwork, PermissionRepo)
	}

	if !repo {
		return []string{"--sandbox", "read-only"}, nil
	}

	return []string{
		"--sandbox", "workspace-write",
		"-c", fmt.Sprintf("sandbox_workspace_write.network_access=%t", network),
	}, nil
}
