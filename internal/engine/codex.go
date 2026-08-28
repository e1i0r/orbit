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
// The five names this replaced were o3-mini, o3, o1, gpt-4o and
// gpt-4.5-preview — a list from before codex shipped. internal/task checks a
// phase's model against this one before it runs anything, so the only models
// a phase could name on codex were five the account answers "not supported"
// to. It was then cut to nothing at all, which was the opposite mistake: the
// dial had one position and codex has four.
//
// These four are what `codex exec --model` was actually run with, one at a
// time, on codex-cli 0.150.1. The other slugs in the binary — gpt-5.6-sol,
// gpt-5.6-pro, gpt-5.4, gpt-5.3-codex, gpt-5.2, gpt-5.2-codex — are the
// legacy names its own picker points at config.toml for, and every one of
// them came back "not supported". Refresh this the same way, by running
// them; codex has no verb that lists them.
func (Codex) Models() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "gpt-5.6-terra", Label: "gpt-5.6-terra"},
		{ID: "gpt-5.6-luna", Label: "gpt-5.6-luna"},
		{ID: "gpt-5.5", Label: "gpt-5.5"},
		{ID: "gpt-5.4-mini", Label: "gpt-5.4-mini"},
	}
}

// Efforts returns the reasoning effort levels codex supports.
//
// Verified by running each one, because the value passes two gates: codex's
// own config enum and then the model's vocabulary. On 0.46.0 the first gate
// was minimal|low|medium|high and this list was cut to three; on 0.150.1 the
// enum has widened and minimal is the one word now rejected by every model
// tried.
//
// max is left out although it works, and that is the one judgement here.
// This list is not per-model — the dial offers it whichever model is chosen —
// so a value that works on gpt-5.6-terra and gpt-5.6-luna and is refused by
// gpt-5.5 and gpt-5.4-mini would be a position on the dial that fails
// depending on a different dial. The five below were accepted by all four.
func (Codex) Efforts() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "none", Label: "none"},
		{ID: "low", Label: "low"},
		{ID: "medium", Label: "medium"},
		{ID: "high", Label: "high"},
		{ID: "xhigh", Label: "xhigh"},
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
